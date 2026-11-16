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
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/k9io/highvolt/cmd/highvolt-server/auth"
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

	old_reload_hash, err := fetchReloadHash(&bearerToken)

	if err != nil {
		l.Logger(l.ERROR, "GetReloadHash: %v", err)
	}

	old_debug_string, err := fetchDebugLevel(&bearerToken)

	if err != nil {

		l.Logger(l.ERROR, "%v", err)
		os.Exit(1)

	}

	for {

		/* Monitor JSONAir for when to reload the Highvolt
		   configuration. On error, skip the comparison so a
		   failed request is not mistaken for a reload signal. */

		reload_hash, err := fetchReloadHash(&bearerToken)

		if err != nil {

			l.Logger(l.ERROR, "GetReloadHash: %v", err)

		} else if reload_hash != old_reload_hash {

			l.Logger(l.NOTICE, "Got reload signal.")

			var configJSON string
			configJSON, bearerToken = config.GetConfigJSON(bearerToken)
			config.Load_Config(configJSON)

			old_reload_hash = reload_hash

		}

		/* Monitor JSONAir for when to set a new debug
		   level. On error, keep the current level. */

		debug_string, err := fetchDebugLevel(&bearerToken)

		if err != nil {

			l.Logger(l.ERROR, "%v", err)

		} else if debug_string != old_debug_string {

			l.Logger(l.NOTICE, "Debug level change: %s", debug_string)

			debug.Set_Debug(debug_string)

			old_debug_string = debug_string

		}

		time.Sleep(models.Env.RELOAD_SLEEP) /* Prevent 100% CPU usage */

	}

}

/* fetchReloadHash calls GetReloadHash and, if JSONAir rejects the bearer
   token, gets a new one via PAT_Auth() and retries once. The refreshed
   token is written back through bearerToken. */

func fetchReloadHash(bearerToken *string) (string, error) {

	reload_hash, err := GetReloadHash(*bearerToken)

	if errors.Is(err, auth.ErrUnauthorized) {

		l.Logger(l.NOTICE, "Bearer Token expired. Getting a new one.")
		*bearerToken = auth.PAT_Auth()
		reload_hash, err = GetReloadHash(*bearerToken)

	}

	return reload_hash, err

}

/* fetchDebugLevel is the debug-level equivalent of fetchReloadHash. */

func fetchDebugLevel(bearerToken *string) (string, error) {

	err, debug_string := debug.GetDebugLevel(*bearerToken)

	if errors.Is(err, auth.ErrUnauthorized) {

		l.Logger(l.NOTICE, "Bearer Token expired. Getting a new one.")
		*bearerToken = auth.PAT_Auth()
		err, debug_string = debug.GetDebugLevel(*bearerToken)

	}

	return debug_string, err

}

func GetReloadHash(bearerToken string) (string, error) {

	reloadURL := fmt.Sprintf("%s/api/%s/jsonair/reload", models.Env.JSONAIR_URL, define.JSONAIR_VERSION)
	jsonQuery := fmt.Sprintf(`{"type":"%s","name":"%s"}`, models.Env.JSONAIR_TYPE, models.Env.JSONAIR_NAME)

	result, status_code, err := http_req.HTTP(jsonQuery, reloadURL, "GET", bearerToken)
	if err != nil {
		return "", fmt.Errorf("http_req.HTTP: %v", err)
	}

	if status_code == http.StatusUnauthorized {
		return "", auth.ErrUnauthorized
	}

	if status_code != http.StatusOK {
		return "", fmt.Errorf("reload request returned HTTP %d", status_code)
	}

	var r reloadResponse
	if err = json.Unmarshal([]byte(result), &r); err != nil {
		return "", fmt.Errorf("cannot parse reload response: %v", err)
	}

	return r.Reload, nil

}
