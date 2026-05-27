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
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/k9io/highvolt/internal/define"
	"github.com/k9io/highvolt/internal/http_req"
)

type Config_Struct struct {
	Code int

	Core struct {
		MIME_Types    []string
		Max_File_Size int64
	}

	Highvolt struct {
		URL    string
		Pat    string
		Submit string
		Query  string
	}
}

var Config Config_Struct

// GetConfigJSON authenticates with JSONAIR and fetches the config profile.
// It retries with exponential backoff and refreshes the bearer token on 401.
func GetConfigJSON() string {
	token := patAuth("jsonair", Env.JSONAIR_URL, define.JSONAIR_VERSION, Env.JSONAIR_PAT)

	configURL := fmt.Sprintf("%s/api/%s/jsonair/config", Env.JSONAIR_URL, define.JSONAIR_VERSION)
	configBody := fmt.Sprintf(`{"type":"%s","name":"%s","decode":true}`, Env.JSONAIR_TYPE, Env.JSONAIR_NAME)

	maxRetries := 10
	wait := time.Second

	var configJSON string
	var statusCode int
	var err error

	for attempt := 0; attempt < maxRetries; attempt++ {
		configJSON, statusCode, err = http_req.HTTP(configBody, configURL, "GET", token)
		if err != nil {
			log.Printf("[ERROR] Config fetch error (attempt %d): %v", attempt+1, err)
		}

		if statusCode == 200 {
			break
		}

		if statusCode == 401 {
			log.Println("[NOTICE] JSONAIR bearer token expired, refreshing")
			token = patAuth("jsonair", Env.JSONAIR_URL, define.JSONAIR_VERSION, Env.JSONAIR_PAT)
		} else {
			log.Printf("[ERROR] Config fetch failed (HTTP %d, attempt %d), retrying in %s", statusCode, attempt+1, wait)
		}

		time.Sleep(wait)
		if wait < 30*time.Second {
			wait *= 2
		}
	}

	if statusCode != 200 {
		log.Fatalf("[ERROR] Failed to fetch JSONAIR config after %d attempts", maxRetries)
	}

	return configJSON
}

func LoadConfig(configJSON string) {
	Config = Config_Struct{}

	if err := json.Unmarshal([]byte(configJSON), &Config); err != nil {
		log.Fatalf("[ERROR] Unable to decode JSONAIR config JSON: %v", err)
	}

	if len(Config.Core.MIME_Types) == 0 {
		log.Fatalf("[ERROR] core.mime_types is empty in JSONAIR config")
	}

	if Config.Core.Max_File_Size == 0 {
		log.Fatalf("[ERROR] core.max_file_size is not set in JSONAIR config")
	}

	Config.Highvolt.Submit = fmt.Sprintf("%s/api/v1/highvolt/submit", Config.Highvolt.URL)
	Config.Highvolt.Query = fmt.Sprintf("%s/api/v1/highvolt/query", Config.Highvolt.URL)

	log.Printf("[INFO] Config loaded: %d MIME types, max_file_size=%d", len(Config.Core.MIME_Types), Config.Core.Max_File_Size)
}
