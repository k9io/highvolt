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
	"os"
	"time"

	"github.com/k9io/highvolt/internal/define"
	"github.com/k9io/highvolt/internal/http_req"
	"github.com/k9io/highvolt/internal/jwt"
	l "github.com/k9io/highvolt/internal/logger"
)

type Config_Struct struct {
	Code int

	Core struct {
		Max_Size       int64
		Sleep_Interval int
	}

	Operating_Systems struct {
		Unix    Unix_Struct
		Windows Windows_Struct
		MacOS   MacOS_Struct
	}

	Highvolt struct {
		URL	string
		Pat     string
		Submit  string
		Query   string

	}

	Syslog struct {
		Host  string
		Proto string
	}
}

type Unix_Struct struct {
	Directories []string
	MIMETypes   []string
	Exclude     []string
}

type Windows_Struct struct {
	Directories []string
	MIMETypes   []string
	Exclude     []string
}

type MacOS_Struct struct {
	Directories []string
	MIMETypes   []string
	Exclude     []string
}

var Config Config_Struct


func GetConfigJSON(bearerToken string) (string, string) {

	var err error
	var config_json string

	status_code := 0 

        config_url := fmt.Sprintf("%s/api/%s/jsonair/config", Env.JSONAIR_URL, define.JSONAIR_VERSION)
        config_json_tmp := fmt.Sprintf(`{"type":"%s","name":"%s","decode":true}`, Env.JSONAIR_TYPE, Env.JSONAIR_NAME)

	maxRetries := 10
	wait := time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {

		config_json, status_code, err = http_req.HTTP(config_json_tmp, config_url, "GET", bearerToken)

		if err != nil {
			l.Logger(l.ERROR, "%v", err)
		}

		if status_code == 200 {
			break
		}

		if status_code == 401 {
			l.Logger(l.NOTICE, "Bearer Token expired.  Getting a new one.")
			bearerToken = jwt.PAT_Auth("jsonair", Env.JSONAIR_URL, define.JSONAIR_VERSION, Env.JSONAIR_PAT, false)
		} else {
			l.Logger(l.ERROR, "Config fetch failed (status %d), retrying in %s", status_code, wait)
		}

		time.Sleep(wait)
		wait *= 2

	}

	if status_code != 200 {
		l.Logger(l.ERROR, "Failed to fetch config after %d attempts.", maxRetries)
		os.Exit(1)
	}

        return config_json, bearerToken
}


func LoadConfig(configJSON string) {

	Config = Config_Struct{}

        err := json.Unmarshal([]byte(configJSON), &Config)

        if err != nil {

                l.Logger(l.ERROR, "Unable to decode JSON from JSONAir config JSON. [%s]", err)
                os.Exit(1)

        }

	/* Sanity Checks */

	if Config.Core.Max_Size == 0 {

		l.Logger(l.ERROR, "Cannot find 'core.max_size' in configation JSON.")
		os.Exit(1)

	}

	if Config.Core.Sleep_Interval == 0 {
		Config.Core.Sleep_Interval = 86400
	}

	if Config.Highvolt.URL == "" {

		l.Logger(l.ERROR, "Cannot find 'highvolt.url' in configuration JSON.")
		os.Exit(1)

	}

	if Config.Highvolt.Pat == "" {

		l.Logger(l.ERROR, "Cannot find 'highvolt.pat' in configuration JSON.")
		os.Exit(1)

	}

	if Config.Syslog.Host == "" {

		l.Logger(l.ERROR, "Cannot find 'syslog.host' in configuration JSON.")
		os.Exit(1)

	}

	if Config.Syslog.Host != "local" {

		if Config.Syslog.Proto == "" {

			l.Logger(l.ERROR, "Cannot find 'syslog.proto' in configuration JSON.")
			os.Exit(1)

		}

	}

	Config.Highvolt.Submit = fmt.Sprintf("%s/api/v1/highvolt/submit", Config.Highvolt.URL)
	Config.Highvolt.Query = fmt.Sprintf("%s/api/v1/highvolt/query", Config.Highvolt.URL)

}
