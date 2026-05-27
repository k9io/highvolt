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
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/k9io/highvolt/internal/define"
	"github.com/k9io/highvolt/internal/http_req"
	"github.com/k9io/highvolt/internal/jwt"

	l "github.com/k9io/highvolt/internal/logger"
)

type Config_Struct struct {
	Code int

	Core struct {
		Suricata_Path string
		MIME_Types    []string
	}

	Highvolt struct {
		URL    string
		Pat    string
		Submit string
		Query  string
	}

	Redis struct {
		Client *redis.Client

		Host     string
		Port     int
		Database int
		Username string
		Password string
		Key      string
	}
	Syslog struct {
		Host  string
		Proto string
	}
}

var Config Config_Struct
var ConfigMu sync.RWMutex

func GetConfigJSON(bearerToken string) (string, string) {

	var err error
	var config_json string

	var status_code int

	config_url := fmt.Sprintf("%s/api/%s/jsonair/config", Env.JSONAIR_URL, define.JSONAIR_VERSION)
	config_json_tmp := fmt.Sprintf(`{"type":"%s","name":"%s","decode":true}`, Env.JSONAIR_TYPE, Env.JSONAIR_NAME)

	for attempt := 0; attempt < 10; attempt++ {

		if attempt > 0 {
			wait := time.Duration(1<<uint(attempt-1)) * time.Second
			l.Logger(l.NOTICE, "Retrying config fetch in %v (attempt %d/10).", wait, attempt+1)
			time.Sleep(wait)
		}

		config_json, status_code, err = http_req.HTTP(config_json_tmp, config_url, "GET", bearerToken)

		if err != nil {
			l.Logger(l.ERROR, "%v", err)
			continue
		}

		if status_code == 401 {
			l.Logger(l.NOTICE, "Bearer Token expired. Getting a new one.")
			bearerToken = jwt.PAT_Auth("jsonair", Env.JSONAIR_URL, define.JSONAIR_VERSION, Env.JSONAIR_PAT, Debug.HTTP)
			continue
		}

		if status_code == 200 {
			return config_json, bearerToken
		}

		l.Logger(l.ERROR, "Got bad response %v.", status_code)

	}

	l.Logger(l.ERROR, "Failed to fetch config after 10 attempts.")
	os.Exit(1)
	return "", bearerToken
}

func LoadConfig(configJSON string) {

	Config = Config_Struct{}

	err := json.Unmarshal([]byte(configJSON), &Config)

	if err != nil {

		l.Logger(l.ERROR, "Unable to decode JSON from JSONAir config JSON. [%s]", err)
		os.Exit(1)

	}

	if Config.Core.Suricata_Path == "" {
		l.Logger(l.ERROR, "Cannot find 'core.suricata_path' in configuration JSON.")
		os.Exit(1)
	}

	if len(Config.Core.MIME_Types) == 0 {
		l.Logger(l.ERROR, "Cannot find 'core.mime_types' in configuration JSON.")
		os.Exit(1)
	}

	if Config.Highvolt.URL == "" {
		l.Logger(l.ERROR, "Cannot find 'highvolt.url' in configuration JSON.")
		os.Exit(1)
	}

	if Config.Highvolt.Pat == "" {
		l.Logger(l.ERROR, "Cannot find 'highvolt.pat' in configuration JSON.")
		os.Exit(1)
	}

	if Config.Redis.Host == "" {
		l.Logger(l.ERROR, "Cannot find 'redis.host' in configuration JSON.")
		os.Exit(1)
	}

	if Config.Redis.Port == 0 {
		l.Logger(l.ERROR, "Cannot find 'redis.port' or is invalid in configuration JSON.")
		os.Exit(1)
	}

	if Config.Redis.Key == "" {
		l.Logger(l.ERROR, "Cannot find 'redis.key' in configuration JSON.")
		os.Exit(1)
	}

	if Config.Syslog.Host == "" {
		l.Logger(l.ERROR, "Cannot find 'syslog.host' in configuration JSON.")
		os.Exit(1)
	}

	if Config.Syslog.Host != "local" && Config.Syslog.Proto == "" {
		l.Logger(l.ERROR, "Cannot find 'syslog.proto' in configuration JSON.")
		os.Exit(1)
	}

	Config.Highvolt.Submit = fmt.Sprintf("%s/api/v1/highvolt/submit", Config.Highvolt.URL)
	Config.Highvolt.Query = fmt.Sprintf("%s/api/v1/highvolt/query", Config.Highvolt.URL)

}

