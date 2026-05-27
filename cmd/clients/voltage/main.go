/*
** Copyright (C) 2026 Key9, Inc <k9.io>
** Copyright (C) 2026 Champ Clark III <cclark@k9.io>
**
** This file is part of the HighVolt.
**
** This program is free software: you can redistribute it and/or modify
** it under the terms of the GNU Affero General Public License as published by
** the Free Software Foundation, either version 3 of the License, or
** (at your option) any later version.
**
** This program is distributed in the hope that it will be useful
** but WITHOUT ANY WARRANTY; without even the implied warranty of
** MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
** GNU Affero General Public License for more details.
**
** You should have received a copy of the GNU Affero General Public License
** along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

/*

NOTES: Needs "debug" and "reload"

*/

package main

import (
	"encoding/gob"
	"flag"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"

	"github.com/k9io/highvolt/internal/device"
	"github.com/k9io/highvolt/internal/jwt"
	"github.com/k9io/highvolt/internal/util"
	"github.com/k9io/highvolt/internal/define"

	l "github.com/k9io/highvolt/internal/logger"

	"github.com/kardianos/service"
)

type HashRegistry map[string]bool

type program struct {
	scan_once bool
	nuke      bool
}

func (p *program) Start(s service.Service) error {
	go p.run()
	return nil
}

func (p *program) Stop(s service.Service) error {
	l.Logger(l.NOTICE, "Service stopping...")
	return nil
}

func main() {

	scan_once := flag.Bool("once", false, "Run a single scan and exit")
	nuke := flag.Bool("nuke", false, "Clean up all local data/configs")

	flag.Parse()

	if *nuke {
		NukeDB()
		os.Exit(0)
	}

	/* Configure the service */

	svcConfig := &service.Config{
		Name:        "Voltage",
		DisplayName: "Highvolt Voltage AI Scanner.",
		Description: "This scans for sensitive documents.",
	}

	prg := &program{
		scan_once: *scan_once, // Pass flags into your program struct
		nuke:      *nuke,
	}

	s, err := service.New(prg, svcConfig)

	if err != nil {
		l.Logger(l.ERROR, "Error: %v", err)
		os.Exit(1)
	}

	if len(os.Args) > 1 {

		var schedErr error

		switch os.Args[1] {

		case "install":
			schedErr = InstallScheduler(s)
		case "uninstall":
			schedErr = UninstallScheduler(s)
		case "start":
			schedErr = StartScheduler(s)
		case "stop":
			schedErr = StopScheduler(s)
		case "restart":
			schedErr = RestartScheduler(s)
		}

		if schedErr != nil {
			l.Logger(l.ERROR, "Error: %v", schedErr)
			os.Exit(1)
		}

		if os.Args[1] == "install" || os.Args[1] == "uninstall" ||
			os.Args[1] == "start" || os.Args[1] == "stop" || os.Args[1] == "restart" {
			return
		}
	}

	/* Validate actual voltage flags before run */

	if len(os.Args) > 1 {
		if os.Args[1] != "--nuke" && os.Args[1] != "-nuke" &&
			os.Args[1] != "--once" && os.Args[1] != "-once" {

			l.Logger(l.ERROR, "Invalid option: %s", os.Args[1])
			os.Exit(1)

		}
	}

	/* Run the service (blocks until stopped) */

	err = s.Run()

	if err != nil {

		l.Logger(l.ERROR, "Error: %v", err)
		os.Exit(1)

	}
}

func (p *program) run() {

	isInteractive := service.Interactive()

	if isInteractive && p.scan_once {

		l.Logger(l.NOTICE, "Running single scan (Command Line Mode)...")

		DoWork()
		os.Exit(0)

	} else {

		l.Logger(l.NOTICE, "Starting service loop...")

		for {
			DoWork()

			l.Logger(l.INFO, "Sleeping for %d seconds.", Config.Core.Sleep_Interval)
			time.Sleep(time.Duration(Config.Core.Sleep_Interval) * time.Second)

		}
	}

}

func DoWork() {

	var err error

	var Directories []string
	var MIME_Types []string
	var Exclude []string

	l.Init_Logger("local", "tcp") // Need config support for syslog

	l.Logger(l.INFO, "Firing up Highvolt Voltage.")

	LoadEnv()

	l.Logger(l.INFO, "Loading config from %s/config", Env.JSONAIR_URL)


	JSONAIR_bearerToken := jwt.PAT_Auth("jsonair", Env.JSONAIR_URL, define.JSONAIR_VERSION, Env.JSONAIR_PAT, false ) 

	JSONAIR_config, JSONAIR_bearerToken := GetConfigJSON( JSONAIR_bearerToken )

	LoadConfig( JSONAIR_config )

	/* DEBUG only need this if the serice is running in the background! */

	HIGHVOLT_bearerToken := jwt.PAT_Auth("highvolt", Config.Highvolt.URL, define.HIGHVOLT_VERSION, Config.Highvolt.Pat, false )

        /* Start the configuration monitor - only if a daemon! */
        //go config.Monitor_Config(bearerToken, configJSON)

	
	l.Init_Logger(Config.Syslog.Host, Config.Syslog.Proto) /* Reload logging based off config */

	/* Load device information into global */

	device.Get_Device_Info()

	/* Establish where we should store our internal Voltage database.  This database
	   will store the sha256 of the files we scan.  This way,  we don't have to query
	   the server for each and every file. */

	voltageDB := GetHighvoltDBFileName()

	l.Logger(l.INFO, "Voltage SHA256 database: %s", voltageDB)
	l.Logger(l.INFO, "Operating system type: %s", device.Device_Info.OS.OS)

	/* Determine what operating system we should listen to in the configuration */

	switch device.Device_Info.OS.OS {

	case "linux", "freebsd", "openbsd", "netbsd", "aix", "solaris", "illumos", "dragonfly", "android":

		Directories = Config.Operating_Systems.Unix.Directories
		MIME_Types = Config.Operating_Systems.Unix.MIMETypes
		Exclude = Config.Operating_Systems.Unix.Exclude

	case "darwin":

		Directories = Config.Operating_Systems.MacOS.Directories

		MIME_Types = Config.Operating_Systems.MacOS.MIMETypes
		Exclude = Config.Operating_Systems.MacOS.Exclude

	case "windows":

		Directories = Config.Operating_Systems.Windows.Directories
		MIME_Types = Config.Operating_Systems.Windows.MIMETypes
		Exclude = Config.Operating_Systems.Windows.Exclude

	default:

		l.Logger(l.ERROR, "Unknown operating system.")
		os.Exit(1)

	}

	mask := os.ModeSymlink | os.ModeDevice | os.ModeNamedPipe | os.ModeSocket

	/* Init GOB */

	registry := make(HashRegistry)

	if f, err := os.Open(voltageDB); err == nil {
		if err = gob.NewDecoder(f).Decode(&registry); err != nil {
			l.Logger(l.WARN, "Failed to load registry from %s: %v. Starting fresh.", voltageDB, err)
			registry = make(HashRegistry)
		} else {
			l.Logger(l.INFO, "Registry loaded: %d entries from %s", len(registry), voltageDB)
		}
		f.Close()
	}

	for _, target := range Directories {

		l.Logger(l.NOTICE, "Scanning %s.", target)

		err = filepath.WalkDir(target, func(full_path string, d fs.DirEntry, err error) error {

			//			l.Logger(l.INFO, "%s", full_path)

			if err != nil {

				if d != nil && d.IsDir() {
					return filepath.SkipDir
				}

				return nil

			}

			if d.Type()&mask != 0 {
				return nil
			}

			/* Can't process directories, so skip them */

			if d.IsDir() {
				return nil
			}

			/* If we've made it this far,  it's a file.... Process it */

			for _, e := range Exclude {

				if strings.Contains(full_path, e) {
					//					l.Logger(l.NOTICE, "Excluding %s [%s]", full_path, e)
					return nil
				}
			}

			magic := util.GetFileMagic(full_path)

			for _, mime := range MIME_Types {

				if magic == mime {

					file_info, err := os.Stat(full_path)

					if err != nil {
						l.Logger(l.ERROR, "Error getting file stats")
						return nil
					}

					if file_info.Size() > Config.Core.Max_Size {

						l.Logger(l.ERROR, "File %s is to large (%d bytes) to process", full_path, file_info.Size())
						return nil

					}

					processFile(full_path, magic, registry, voltageDB, &HIGHVOLT_bearerToken)

				}

			}

			return nil
		})

	}

	if err != nil {
		l.Logger(l.ERROR, "Error: %v", err)
	}

}

func processFile(full_path string, magic string, registry HashRegistry, voltageDB string, bearerToken *string) {

	f, err := os.Open(full_path)
	if err != nil {
		l.Logger(l.ERROR, "Cannot open %s. [%v]", full_path, err)
		return
	}
	defer f.Close()

	sha256 := sha256.New()
	sha1 := sha1.New()
	md5 := md5.New()

	if _, err := io.Copy(io.MultiWriter(sha256, sha1, md5), f); err != nil {
		l.Logger(l.ERROR, "Cannot generate hash. [%v]", err)
		return
	}

	sha256_hex := hex.EncodeToString(sha256.Sum(nil))

	if registry[sha256_hex] {
		return
	}

	sha1_hex := hex.EncodeToString(sha1.Sum(nil))
	md5_hex := hex.EncodeToString(md5.Sum(nil))

	if !FileScan(full_path, magic, md5_hex, sha1_hex, sha256_hex, bearerToken) {
		return
	}

	registry[sha256_hex] = true

	dbf, err := os.Create(voltageDB)
	if err != nil {
		l.Logger(l.ERROR, "Cannot create registry file %s: %v", voltageDB, err)
	} else {
		if err = gob.NewEncoder(dbf).Encode(registry); err != nil {
			l.Logger(l.ERROR, "Cannot save registry to %s: %v", voltageDB, err)
		} else {
			l.Logger(l.DEBUG, "Registry saved: %d entries in %s", len(registry), voltageDB)
		}
		dbf.Close()
	}

}
