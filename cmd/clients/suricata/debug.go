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
	"strings"

	"github.com/k9io/highvolt/internal/define"
	"github.com/k9io/highvolt/internal/http_req"

	l "github.com/k9io/highvolt/internal/logger"
)

type Debug_Struct struct {
	Submit bool
	HTTP   bool
	Redis  bool
	Sleep  bool
}

var Debug Debug_Struct

func Set_Debug(debug_level string) {

	/* Make sure output is sane */

	debug_level = strings.ReplaceAll(debug_level, " ", "") // Remove any stray white space

	debug_slice := strings.Split(debug_level, ",")

	Debug = Debug_Struct{}

	for _, i := range debug_slice {

		switch i {

		case "":
		case "none":

		case "http":

			Debug.HTTP = true

		case "submit":

			Debug.Submit = true

		case "redis":

			Debug.Redis = true

		case "sleep":

			Debug.Sleep = true

		case "all":

			Debug.Submit = true
			Debug.HTTP = true
			Debug.Redis = true
			Debug.Sleep = true

		default:

			l.Logger(l.ERROR, "Unrecognized debug option: '%s'", i)
			os.Exit(1)
		}

	}

	l.Logger(l.INFO, "Debugging set to: %s", debug_level)

}

func GetDebugLevel() (error, string) {

	var err error
	var debug_level string

	var D *Debug_Monitor_Struct

	debug_url := fmt.Sprintf("%s/api/%s/jsonair/debug", Env.JSONAIR_URL, define.JSONAIR_VERSION)
	json_query := fmt.Sprintf(`{"type":"%s","name":"%s"}`, Env.JSONAIR_TYPE, Env.JSONAIR_NAME)

	debug_level, _, err = http_req.HTTP(json_query, debug_url, "GET", Tokens.JSONAir)

	if Debug.HTTP == true {

		l.Logger(l.DEBUG, "[HTTP] %s", debug_level)

	}

	if err != nil {

		return fmt.Errorf("http_req.HTTP: %v", err), ""

	}

	err = json.Unmarshal([]byte(debug_level), &D)

	if err != nil {

		return fmt.Errorf("Cannot get debug_level: %v", err), ""

	}

	return nil, D.Debug

}
