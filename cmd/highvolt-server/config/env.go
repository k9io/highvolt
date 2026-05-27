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

package config

import (
	"os"
	"strconv"
	"time"

	"github.com/k9io/highvolt/cmd/highvolt-server/models"

	"github.com/k9io/highvolt/internal/http_req"
	l "github.com/k9io/highvolt/internal/logger"

	"github.com/joho/godotenv"
)

func Load_Env() models.Environment_Struct {

	var err error
	var tmp string

	/* Defaults */

	models.Env.CONFIG_SLEEP = 600
	models.Env.RELOAD_SLEEP = 300
	models.Env.DEBUG_SLEEP = 60

	if err := godotenv.Load(); err != nil {
		l.Logger(l.NOTICE, "No .env file found, using system environment variables.")
	}

	if os.Getenv("TLS_SKIP_VERIFY") == "true" {
		l.Logger(l.WARN, "TLS certificate verification is DISABLED. Do not use in production.")
		http_req.Configure(true)
	}

	models.Env = models.Environment_Struct{} /* Clear out any previous loaded values */

	models.Env.RUNAS = os.Getenv("RUNAS")

	if models.Env.RUNAS == "" {
		l.Logger(l.ERROR, "RUNAS environment variable is not set.")
		os.Exit(1)
	}

	models.Env.JSONAIR_PAT = os.Getenv("JSONAIR_PAT")

	if models.Env.JSONAIR_PAT == "" {
		l.Logger(l.ERROR, "JSONAIR_PAT environment variable is not set.")
		os.Exit(1)
	}

	models.Env.JSONAIR_URL = os.Getenv("JSONAIR_URL")

	if models.Env.JSONAIR_URL == "" {
		l.Logger(l.ERROR, "JSONAIR_URL environment variable is not set.")
		os.Exit(1)
	}

	models.Env.JSONAIR_TYPE = os.Getenv("JSONAIR_TYPE")

	if models.Env.JSONAIR_TYPE == "" {
		l.Logger(l.ERROR, "JSONAIR_TYPE environment variable is not set.")
		os.Exit(1)
	}

	models.Env.JSONAIR_NAME = os.Getenv("JSONAIR_NAME")

	if models.Env.JSONAIR_NAME == "" {
		l.Logger(l.ERROR, "JSONAIR_NAME environment variable is not set.")
		os.Exit(1)
	}

	tmp = os.Getenv("CONFIG_SLEEP")

	if tmp != "" {

		val, err := strconv.Atoi(tmp)

		if err != nil {

			l.Logger(l.ERROR, "CONFIG_SLEEP environment variable is not set.")
			os.Exit(1)
		}

		if val == 0 {

			l.Logger(l.ERROR, "CONFIG_SLEEP must be a non-zero number.")
			os.Exit(1)

		}

		models.Env.CONFIG_SLEEP = time.Duration(val) * time.Second

	} else { 

		l.Logger(l.ERROR, "CONFIG_SLEEP environment variable is not set.")
		os.Exit(1)

	}

	tmp = os.Getenv("RELOAD_SLEEP")

	if tmp != "" {

		val, err := strconv.Atoi(tmp)

		if err != nil {

			l.Logger(l.ERROR, "RELOAD_SLEEP environment variable is not set.")
			os.Exit(1)
		}

		if val == 0 {

			l.Logger(l.ERROR, "RELOAD_SLEEP must be a non-zero number.")
			os.Exit(1)

		}

		models.Env.RELOAD_SLEEP = time.Duration(val) * time.Second

	} else {

        l.Logger(l.ERROR, "RELOAD_SLEEP environment variable is not set.")
	os.Exit(1)

        }

	tmp = os.Getenv("DEBUG_SLEEP")

	if tmp != "" {

		val, err := strconv.Atoi(tmp)

		if err != nil {

			l.Logger(l.ERROR, "DEBUG_SLEEP environment variable is not set.")
			os.Exit(1)
		}

		if val == 0 {

			l.Logger(l.ERROR, "DEBUG_SLEEP must be a non-zero number.")
			os.Exit(1)

		}

		models.Env.DEBUG_SLEEP = time.Duration(val) * time.Second

	} else {

                l.Logger(l.ERROR, "DEBUG_SLEEP environment variable is not set.")
		os.Exit(1)

        }

	tmp = os.Getenv("JWT_TOKEN_EXPIRE")

	if tmp == "" {
		l.Logger(l.ERROR, "JWT_TOKEN_EXPIRE environment variable is not set.")
		os.Exit(1)
	}

	models.Env.JWTTokenExpire, err = strconv.Atoi(tmp)

	if err != nil {
		l.Logger(l.ERROR, "JWT_TOKEN_EXPIRE environment variable is not an integer.")
		os.Exit(1)
	}

	if models.Env.JWTTokenExpire == 0 {
		l.Logger(l.ERROR, "JWT_TOKEN_EXPIRE must be greater than zero.")
		os.Exit(1)
	}

	tmp = os.Getenv("JWT_TOKEN_SECRET")

	if tmp == "" {
		l.Logger(l.ERROR, "JWT_TOKEN_SECRET environment variable is not set.")
		os.Exit(1)
	}

	models.Env.JWTTokenSecret = []byte(tmp)

	models.Env.HIGHVOLT_PAT = os.Getenv("HIGHVOLT_PAT")

	if models.Env.HIGHVOLT_PAT == "" {
		l.Logger(l.ERROR, "HIGHVOLT_PAT environment variable is not set.")
		os.Exit(1)
	}

	return models.Env

}
