/*
** Copyright (C) 2026 Key9, Inc <k9.io>
** Copyright (C) 2026 Champ Clark III <cclark@k9.io>
**
** This file is part of the HighVolt JSON analysis engine
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

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"encoding/base64"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/k9io/highvolt/internal/define"
	"github.com/k9io/highvolt/internal/device"
	"github.com/k9io/highvolt/internal/http_req"
	"github.com/k9io/highvolt/internal/jwt"

	l "github.com/k9io/highvolt/internal/logger"
)

type Tokens_Struct struct {
	Highvolt string
	JSONAir  string
}

var Tokens Tokens_Struct

var self string = "suricata"

func main() {

	var results string
	var status_code int
	var JSONAir_config string

	ctx := context.Background()

	/* Load device information into global */

	device.Get_Device_Info()

	/* Signal Handler */

	signalChannel := make(chan os.Signal, 2)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM, syscall.SIGABRT)

	go SigHandler(signalChannel)

	l.Init_Logger("local", "tcp") // Need config support for syslog

	l.Logger(l.INFO, "Firing up Highvolt Suricata.")

	LoadEnv()

	l.Logger(l.INFO, "Loading config from %s/config", Env.JSONAIR_URL)

	Tokens.JSONAir = jwt.PAT_Auth("jsonair", Env.JSONAIR_URL, define.JSONAIR_VERSION, Env.JSONAIR_PAT, false)

	JSONAir_config, Tokens.JSONAir = GetConfigJSON(Tokens.JSONAir)

	LoadConfig(JSONAir_config)

	err, debug_level := GetDebugLevel()

	if err != nil {
		l.Logger(l.ERROR, "%v", err)
		os.Exit(1)
	}

	Set_Debug(debug_level)

	Tokens.Highvolt = jwt.PAT_Auth("highvolt", Config.Highvolt.URL, define.HIGHVOLT_VERSION, Config.Highvolt.Pat, Debug.HTTP)

	l.Init_Logger(Config.Syslog.Host, Config.Syslog.Proto) /* Reload logging based off config */

	go Monitor_State(debug_level)

	/* Init Redis for Suricata sub/pub */

	Redis_Init()

	l.Logger(l.INFO, "Monitoring Suricata Redis Pub/Sub.")

	for {

		for {

			ConfigMu.RLock()
			redisClient := Config.Redis.Client
			redisKey := Config.Redis.Key
			hvQuery := Config.Highvolt.Query
			hvSubmit := Config.Highvolt.Submit
			suricataPath := Config.Core.Suricata_Path
			mimeTypes := Config.Core.MIME_Types
			ConfigMu.RUnlock()

			fileinfo_json, err := redisClient.LPop(ctx, redisKey).Result()

			/* No records, break */

			if err != nil {

				if Debug.Redis == true {
					l.Logger(l.DEBUG, "[REDIS] No data found in Surcata PUB/SUB.")
				}

				break
			}

			if Debug.Redis == true {
				l.Logger(l.DEBUG, "[REDIS] Data from PUB/SUB: %s", fileinfo_json)
			}

			if Debug.Submit == true {
				l.Logger(l.DEBUG, "Data From Suricata: %s", fileinfo_json)
			}

			sha256 := gjson.Get(fileinfo_json, "fileinfo.sha256").String()

			if sha256 == "" {
				l.Logger(l.WARN, "Cannot find 'fileinfo.sha256' in JSON: %s", fileinfo_json)
				continue
			}

			md5 := gjson.Get(fileinfo_json, "fileinfo.md5").String()

			if md5 == "" {
				l.Logger(l.WARN, "Cannot find 'fileinfo.md5' in JSON: %s", fileinfo_json)
				continue
			}

			sha1 := gjson.Get(fileinfo_json, "fileinfo.sha1").String()

			if sha1 == "" {
				l.Logger(l.WARN, "Cannot find 'fileinfo.sha1' in JSON: %s", fileinfo_json)
				continue
			}

			full_path := gjson.Get(fileinfo_json, "fileinfo.filename").String()

			if full_path == "" {
				l.Logger(l.WARN, "Cannot find 'fileinfo.filename' in JSON: %s", fileinfo_json)
				continue
			}

			filename := filepath.Base(full_path) // Get just the filename

			filestored := gjson.Get(fileinfo_json, "fileinfo.stored").Bool()

			if filestored == false {
				l.Logger(l.INFO, "Suricata did not store file %s.  Skipping.", sha256)
				continue
			}

			l.Logger(l.NOTICE, "Got incoming data from Suricata for %s.", sha256)

			/* Query to see if the file has been submitted in the past */

			highvolt_query, _ := sjson.Set("", "sha256", sha256)

			results, status_code, err = http_req.HTTP(highvolt_query, hvQuery, "POST", Tokens.Highvolt)

			if err != nil {
				l.Logger(l.WARN, "%v", err)
				continue
			}

			if status_code == 401 {
				l.Logger(l.NOTICE, "Highvolt bearer token expired. Getting a new one.")
				Tokens.Highvolt = jwt.PAT_Auth("highvolt", Config.Highvolt.URL, define.HIGHVOLT_VERSION, Config.Highvolt.Pat, Debug.HTTP)
				results, status_code, err = http_req.HTTP(highvolt_query, hvQuery, "POST", Tokens.Highvolt)
				if err != nil {
					l.Logger(l.WARN, "%v", err)
					continue
				}
			}

			if results == "" {
				l.Logger(l.WARN, "No results from Highvolt server")
				continue
			}

			if Debug.HTTP == true {
				l.Logger(l.DEBUG, "[HTTP] POST to %s", hvQuery)
				l.Logger(l.DEBUG, "[HTTP] Results: %s", results)
				l.Logger(l.DEBUG, "[HTTP] Status code: %d", status_code)
			}

			code := gjson.Get(results, "code").Int()

			if Debug.Submit == true {

				l.Logger(l.DEBUG, "[SUBMIT] Highvolt Results | SHA256: %s | Status: %d", sha256, code)
				l.Logger(l.DEBUG, "[SUBMIT] Highvolt JSON: %s", results)

			}

			if code == 200 {
				l.Logger(l.NOTICE, "%s has already been processed.  Skipping", sha256)
				continue
			}

			l.Logger(l.NOTICE, "Submitting %s for analysis", sha256)

			sha256_twobytes := sha256[:2] // First two bytes Suricata uses as part of the path.

			file := fmt.Sprintf("%s/%s/%s", suricataPath, sha256_twobytes, sha256)

			/* Mimetype */

			err, mime_flag, mimetype := MIMEType(file, sha256, mimeTypes)

			if err != nil {
				l.Logger(l.ERROR, "%v", err)
				continue
			}

			if mime_flag == false {
				l.Logger(l.WARN, "Got MIME type %s, which is not in our MIME type list.", mimetype)
				continue
			}

			fileinfo_json, _ = sjson.Set(fileinfo_json, "highvolt.mimetype", mimetype)

			data, err := os.ReadFile(file)

			if err != nil {
				l.Logger(l.ERROR, "Cannot read %s [%v].", file, err)
				continue
			}

			/* Build Highvolt JSON */

			now := time.Now()
			highvolt_timestamp := now.Format(time.RFC3339)

			highvolt_json, _ := sjson.Set("", "mimetype", mimetype)
			highvolt_json, _ = sjson.Set(highvolt_json, "timestamp", highvolt_timestamp)

			encodedStr := base64.StdEncoding.EncodeToString(data)

			highvolt_json, _ = sjson.Set(highvolt_json, "file_data", encodedStr)
			highvolt_json, _ = sjson.Set(highvolt_json, "full_path", full_path)
			highvolt_json, _ = sjson.Set(highvolt_json, "filename", filename)
			highvolt_json, _ = sjson.Set(highvolt_json, "app", "suricata")
			highvolt_json, _ = sjson.Set(highvolt_json, "sha256", sha256)
			highvolt_json, _ = sjson.Set(highvolt_json, "sha1", sha1)
			highvolt_json, _ = sjson.Set(highvolt_json, "md5", md5)

			highvolt_json, _ = sjson.Set(highvolt_json, "suricata", fileinfo_json)

			highvolt_json = device.Device_Info_JSON(highvolt_json)

			/* Submit to Highvolt/LLM for analysis */

			results, status_code, err = http_req.HTTP(highvolt_json, hvSubmit, "POST", Tokens.Highvolt)

			if err != nil {
				l.Logger(l.ERROR, "Cannot make submission: %v", err)
				continue
			}

			if status_code == 401 {
				l.Logger(l.NOTICE, "Highvolt bearer token expired. Getting a new one.")
				Tokens.Highvolt = jwt.PAT_Auth("highvolt", Config.Highvolt.URL, define.HIGHVOLT_VERSION, Config.Highvolt.Pat, Debug.HTTP)
				_, _, err = http_req.HTTP(highvolt_json, hvSubmit, "POST", Tokens.Highvolt)
				if err != nil {
					l.Logger(l.ERROR, "Cannot make submission: %v", err)
					continue
				}
			}

			if Debug.HTTP == true {
				l.Logger(l.DEBUG, "[HTTP] POST to %s", hvSubmit)
				l.Logger(l.DEBUG, "[HTTP] Results: %s", results)
				l.Logger(l.DEBUG, "[HTTP] Status code: %d", status_code)
			}

			err = os.Remove(file)

			if err != nil {
				l.Logger(l.ERROR, "Cannot delete file %s [%s]", file, err)
				continue
			}

		}

		if Debug.Sleep == true {
			l.Logger(l.DEBUG, "[SLEEP] Sleeping one second in main.")
		}

		time.Sleep(1 * time.Second) /* Prevent 100% CPU usage */

	}

}
