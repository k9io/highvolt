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

package debug

import (
	"fmt"
	"os"
	"strings"
	"encoding/json"

	"github.com/k9io/highvolt/internal/http_req"
	"github.com/k9io/highvolt/internal/define"


	l "github.com/k9io/highvolt/internal/logger"

	"github.com/k9io/highvolt/cmd/highvolt-server/models"



//	"github.com/tidwall/gjson"
//	"github.com/tidwall/sjson"
)

type Debug_Struct struct {
	Opensearch bool
	LLM        bool
	Submit     bool
	Config     bool
	Redis      bool
	Workers    bool
	Health     bool
	HTTP	   bool

	Debug  bool
	Reload bool
}

type Debug_Monitor_Struct struct {
	Debug string
}

var X Debug_Struct

func Set_Debug(debug_level string) {

	debug_level = strings.ReplaceAll(debug_level, " ", "") // Remove any stray white spaces
	debug_slice := strings.Split(debug_level, ",")

	X = Debug_Struct{}

	for _, i := range debug_slice {

		switch i {

		case "":
		case "none":

		case "opensearch":
			X.Opensearch = true

		case "llm":

			X.LLM = true

		case "submit":

			X.Submit = true

		case "config":

			X.Config = true

		case "redis":

			X.Redis = true

		case "debug":

			X.Debug = true

		case "reload":

			X.Reload = true

		case "workers":

			X.Workers = true

		case "health":

			X.Health = true

		case "all":

			X.Opensearch = true
			X.Submit = true
			X.LLM = true
			X.Config = true
			X.Redis = true
			X.Workers = true
			X.Debug = true
			X.Reload = true
			X.Health = true

		default:
			l.Logger(l.ERROR, "Unrecognized debug option: '%s'.", i)
			os.Exit(1)
		}

	}

}


func GetDebugLevel( bearerToken string ) (error, string) {

	var err error
	var debug_level string

	var D *Debug_Monitor_Struct

	debug_url := fmt.Sprintf("%s/api/%s/jsonair/debug", models.Env.JSONAIR_URL, define.JSONAIR_VERSION)
	json_query := fmt.Sprintf(`{"type":"%s","name":"%s"}`, models.Env.JSONAIR_TYPE, models.Env.JSONAIR_NAME)

	debug_level, _, err = http_req.HTTP(json_query, debug_url, "GET", bearerToken)

	if X.HTTP == true {

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
