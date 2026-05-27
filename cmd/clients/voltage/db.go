package main

import (
	"os"
	"path/filepath"

	l "github.com/k9io/highvolt/internal/logger"
)

func GetHighvoltDBFileName() string {

	/* Establish where we should store our internal Voltage database.  This database
	   will store the sha256 of the files we scan.  This way,  we don't have to query
	   the server for each and every file. */

	var highvoltConfigDir string

	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		// No HOME (e.g. running as a system daemon) — store next to the binary.
		exePath, exeErr := os.Executable()
		if exeErr != nil {
			l.Logger(l.ERROR, "Cannot determine executable path: %v", exeErr)
			os.Exit(1)
		}
		highvoltConfigDir = filepath.Join(filepath.Dir(exePath), "Highvolt")
	} else {
		highvoltConfigDir = filepath.Join(userConfigDir, "Highvolt")
	}

	err = os.MkdirAll(highvoltConfigDir, 0755)

	if err != nil {
		l.Logger(l.ERROR, "Cannot make dir %s: %v", highvoltConfigDir, err)
		os.Exit(1)
	}

	voltageDB := filepath.Join(highvoltConfigDir, "voltage.db")

	return voltageDB

}

func NukeDB() {

	voltageDB := GetHighvoltDBFileName()

	l.Logger(l.NOTICE, "Nuking %s.", voltageDB)

	err := os.Remove(voltageDB)

	if err != nil {

		l.Logger(l.ERROR, "Error deleting cache: %v", err)
		os.Exit(1)
	}

	l.Logger(l.NOTICE, "%s deleted successfully.", voltageDB)

}
