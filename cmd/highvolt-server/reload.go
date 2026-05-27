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

	"github.com/k9io/highvolt/cmd/highvolt-server/config"
	"github.com/k9io/highvolt/cmd/highvolt-server/debug"
	"github.com/k9io/highvolt/cmd/highvolt-server/models"

	"github.com/k9io/highvolt/internal/define"
	"github.com/k9io/highvolt/internal/http_req"
	l "github.com/k9io/highvolt/internal/logger"
)

type reloadResponse struct {
	Reload string `json:"reload"`
}

func Monitor_Reload(bearerToken string) {

	var old_reload_hash string

	reload_hash, err := GetReloadHash(bearerToken)
	if err != nil {
		l.Logger(l.ERROR, "GetReloadHash: %v", err)
	}
	old_reload_hash = reload_hash

	err, debug_string := debug.GetDebugLevel(bearerToken)

	if err != nil {

		l.Logger(l.ERROR, "%v", err)
		os.Exit(1)

	}

	old_debug_string := debug_string

	for {

		/* Monitor Redis for when to reload the Highvolt
		   configuration. */

		reload_hash, err = GetReloadHash(bearerToken)
		if err != nil {
			l.Logger(l.ERROR, "GetReloadHash: %v", err)
		}

		if reload_hash != old_reload_hash {

			l.Logger(l.NOTICE, "Got reload signal.")

			var configJSON string
			configJSON, bearerToken = config.GetConfigJSON(bearerToken)
			config.Load_Config(configJSON)

			old_reload_hash = reload_hash

		}

		/* Monitor Redis for when to set a new debug
		   level */

		err, debug_string = debug.GetDebugLevel(bearerToken)

		if err != nil {
			l.Logger(l.ERROR, "%v", err)
		}

		if debug_string != old_debug_string {

			l.Logger(l.NOTICE, "Debug level change: %s", debug_string)

			debug.Set_Debug(debug_string)

			old_debug_string = debug_string

		}

		time.Sleep(60 * time.Second) /* Prevent 100% CPU usage */

	}

}

func GetReloadHash(bearerToken string) (string, error) {

	reloadURL := fmt.Sprintf("%s/api/%s/jsonair/reload", models.Env.JSONAIR_URL, define.JSONAIR_VERSION)
	jsonQuery := fmt.Sprintf(`{"type":"%s","name":"%s"}`, models.Env.JSONAIR_TYPE, models.Env.JSONAIR_NAME)

	result, _, err := http_req.HTTP(jsonQuery, reloadURL, "GET", bearerToken)
	if err != nil {
		return "", fmt.Errorf("http_req.HTTP: %v", err)
	}

	var r reloadResponse
	if err = json.Unmarshal([]byte(result), &r); err != nil {
		return "", fmt.Errorf("cannot parse reload response: %v", err)
	}

	return r.Reload, nil

}
