package main

import (
	"encoding/gob"
	"os"
	"path/filepath"

	l "github.com/k9io/highvolt/internal/logger"
)

type HashRegistry map[string]bool

func GetHighvoltDBFileName() string {

	userConfigDir, err := os.UserConfigDir()

	if err != nil {
		l.Logger(l.ERROR, "Cannot get UserConfigDir: %v", err)
		os.Exit(1)
	}

	highvoltConfigDir := filepath.Join(userConfigDir, "Highvolt")

	err = os.MkdirAll(highvoltConfigDir, 0755)

	if err != nil {
		l.Logger(l.ERROR, "Cannot make dir %s: %v", highvoltConfigDir, err)
		os.Exit(1)
	}

	return filepath.Join(highvoltConfigDir, "aws-s3.db")

}

func LoadRegistry(dbPath string) HashRegistry {

	registry := make(HashRegistry)

	if f, err := os.Open(dbPath); err == nil {
		if err = gob.NewDecoder(f).Decode(&registry); err != nil {
			l.Logger(l.WARN, "Failed to load registry from %s: %v. Starting fresh.", dbPath, err)
			registry = make(HashRegistry)
		} else {
			l.Logger(l.INFO, "Registry loaded: %d entries from %s", len(registry), dbPath)
		}
		f.Close()
	}

	return registry

}

func SaveRegistry(registry HashRegistry, dbPath string) {

	f, err := os.Create(dbPath)

	if err != nil {
		l.Logger(l.ERROR, "Cannot create registry file %s: %v", dbPath, err)
		return
	}

	if err = gob.NewEncoder(f).Encode(registry); err != nil {
		l.Logger(l.ERROR, "Cannot save registry to %s: %v", dbPath, err)
	} else {
		l.Logger(l.DEBUG, "Registry saved: %d entries in %s", len(registry), dbPath)
	}

	f.Close()

}

func NukeDB() {

	dbPath := GetHighvoltDBFileName()

	l.Logger(l.NOTICE, "Nuking %s.", dbPath)

	err := os.Remove(dbPath)

	if err != nil {
		l.Logger(l.ERROR, "Error deleting cache: %v", err)
		os.Exit(1)
	}

	l.Logger(l.NOTICE, "%s deleted successfully.", dbPath)

}
