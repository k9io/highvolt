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
//	"fmt"
	"os"

	"path/filepath"

	"github.com/k9io/highvolt/internal/http_req"
	l "github.com/k9io/highvolt/internal/logger"

	"github.com/joho/godotenv"
)

type Environment_Struct struct {

        JSONAIR_PAT string
        JSONAIR_URL  string
        JSONAIR_TYPE string
        JSONAIR_NAME string

}

var Env Environment_Struct

func LoadEnv() {

	// Prefer the system-wide config dir; fall back to next to the binary for dev use.
	envPath := "/etc/voltage/.env"
	if _, statErr := os.Stat(envPath); os.IsNotExist(statErr) {
		exePath, err := os.Executable()
		if err != nil {
			l.Logger(l.ERROR, "Unable to determine executable path: %v", err)
			os.Exit(1)
		}
		envPath = filepath.Join(filepath.Dir(exePath), ".env")
	}

	_ = godotenv.Load(envPath) // Loads .env into the system environment

	if os.Getenv("TLS_SKIP_VERIFY") == "true" {
		l.Logger(l.WARN, "TLS certificate verification is DISABLED. Do not use in production.")
		http_req.Configure(true)
	}

	Env.JSONAIR_PAT = os.Getenv("JSONAIR_PAT")

        if Env.JSONAIR_PAT == "" {
                l.Logger(l.ERROR, "JSONAIR_PAT environment variable is not set.")
                os.Exit(1)
        }

        Env.JSONAIR_URL = os.Getenv("JSONAIR_URL")

        if Env.JSONAIR_URL == "" {
                l.Logger(l.ERROR, "JSONAIR_URL environment variable is not set.")
                os.Exit(1)
        }

        Env.JSONAIR_NAME = os.Getenv("JSONAIR_NAME")

        if Env.JSONAIR_NAME == "" {
                l.Logger(l.ERROR, "JSONAIR_NAME environment variable is not set.")
                os.Exit(1)
        }

        Env.JSONAIR_TYPE = os.Getenv("JSONAIR_TYPE")

        if Env.JSONAIR_TYPE == "" {
                l.Logger(l.ERROR, "JSONAIR_TYPE environment variable is not set.")
                os.Exit(1)
        }



}
