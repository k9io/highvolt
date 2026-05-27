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
	"os"
	"time"

	"encoding/json"

	"fmt"

	"github.com/k9io/highvolt/internal/define"
	"github.com/k9io/highvolt/internal/http_req"

	l "github.com/k9io/highvolt/internal/logger"
)

type Reload_Monitor_Struct struct {
	Reload string
}

type Debug_Monitor_Struct struct {
	Debug string
}

func Monitor_State(debug_level string) {

	var flag bool

	var reload_hash string

	var debug_level_old string
	var reload_hash_old string

	var err error

	var R *Reload_Monitor_Struct
	var D *Debug_Monitor_Struct

	debug_level_old = debug_level

	debug_url := fmt.Sprintf("%s/api/%s/jsonair/debug", Env.JSONAIR_URL, define.JSONAIR_VERSION)
	reload_url := fmt.Sprintf("%s/api/%s/jsonair/reload", Env.JSONAIR_URL, define.JSONAIR_VERSION)

	json_query := fmt.Sprintf(`{"type":"%s","name":"%s"}`, Env.JSONAIR_TYPE, Env.JSONAIR_NAME)

	for {

		if Debug.Sleep == true {

			l.Logger(l.DEBUG, "[SLEEP] Sleeping for %v", Env.SLEEP)

		}

		time.Sleep(Env.SLEEP)

		debug_level, _, err = http_req.HTTP(json_query, debug_url, "GET", Tokens.JSONAir)

		if err != nil {

			l.Logger(l.ERROR, "http_req:HTTP: %v", err)
			os.Exit(1)

		}

		err = json.Unmarshal([]byte(debug_level), &D)

		if err != nil {

			l.Logger(l.ERROR, "Cannot parse debug_level: %v", err)
			os.Exit(1)

		}

		reload_hash, _, err = http_req.HTTP(json_query, reload_url, "GET", Tokens.JSONAir)

		if err != nil {

			l.Logger(l.ERROR, "http_req:HTTP: %v", err)
			os.Exit(1)

		}

		err = json.Unmarshal([]byte(reload_hash), &R)

		if err != nil {

			l.Logger(l.ERROR, "Cannot parse reload_hash: %v", err)
			os.Exit(1)

		}

		if flag == false {

			/* First run,  we copy new = old. This only catches
			   on the first pass */

			reload_hash_old = reload_hash

			flag = true

		}

		if D.Debug != debug_level_old {

			l.Logger(l.DEBUG, "**** Debug level has changed from '%v' to '%v'  ****", D.Debug, debug_level_old)
			debug_level_old = D.Debug

			Set_Debug(D.Debug)

		}

		if reload_hash != reload_hash_old {

			l.Logger(l.DEBUG, "**** Got signal from JSONAir to reload ****: %s:%s", reload_hash, reload_hash_old)
			reload_hash_old = reload_hash

			var JSONAir_config string

			JSONAir_config, Tokens.JSONAir = GetConfigJSON(Tokens.JSONAir)

			ConfigMu.Lock()
			oldClient := Config.Redis.Client
			LoadConfig(JSONAir_config)
			if oldClient != nil {
				oldClient.Close()
			}
			Redis_Init()
			ConfigMu.Unlock()

		}

	}

}
