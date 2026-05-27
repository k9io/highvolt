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
	"log"
	"os"

	"github.com/k9io/highvolt/internal/http_req"
)

type Environment_Struct struct {
	JSONAIR_PAT         string // populated at runtime from Secrets Manager
	JSONAIR_URL         string
	JSONAIR_TYPE        string
	JSONAIR_NAME        string
	JSONAIR_SECRET_NAME string // Secrets Manager secret name holding the JSONAIR PAT
	CROSS_ACCOUNT_ROLE  string // IAM role name to assume in member accounts (e.g. OrganizationAccountAccessRole)
}

var Env Environment_Struct

func LoadEnv() {
	if os.Getenv("TLS_SKIP_VERIFY") == "true" {
		log.Println("[WARN] TLS certificate verification is DISABLED. Do not use in production.")
		http_req.Configure(true)
	}

	Env.JSONAIR_SECRET_NAME = mustEnv("JSONAIR_SECRET_NAME")
	Env.JSONAIR_URL = mustEnv("JSONAIR_URL")
	Env.JSONAIR_NAME = mustEnv("JSONAIR_NAME")
	Env.JSONAIR_TYPE = mustEnv("JSONAIR_TYPE")

	Env.CROSS_ACCOUNT_ROLE = os.Getenv("CROSS_ACCOUNT_ROLE")
	if Env.CROSS_ACCOUNT_ROLE == "" {
		Env.CROSS_ACCOUNT_ROLE = "OrganizationAccountAccessRole"
	}
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("[ERROR] Required environment variable %s is not set", key)
	}
	return v
}
