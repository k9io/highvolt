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
	"encoding/base64"
	"os"
	"path/filepath"
	"time"

	"github.com/k9io/highvolt/internal/define"
	"github.com/k9io/highvolt/internal/device"
	"github.com/k9io/highvolt/internal/http_req"
	"github.com/k9io/highvolt/internal/jwt"
	l "github.com/k9io/highvolt/internal/logger"


	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func FileScan(full_path string, mimetype string, md5_hex string, sha1_hex string, sha256_hex string, bearerToken *string) bool {

	now := time.Now()
	highvolt_timestamp := now.Format(time.RFC3339)

	file_data, err := os.ReadFile(full_path)

	if err != nil {

		l.Logger(l.ERROR, "Cannot read %s [%v].", full_path, err)
		return false

	}

	encodedStr := base64.StdEncoding.EncodeToString(file_data)

	filename := filepath.Base(full_path)

	highvolt_json, _ := sjson.Set("", "mimetype", mimetype)
	highvolt_json, _ = sjson.Set(highvolt_json, "timestamp", highvolt_timestamp)
	highvolt_json, _ = sjson.Set(highvolt_json, "file_data", encodedStr)
	highvolt_json, _ = sjson.Set(highvolt_json, "app", "voltscan")
	highvolt_json, _ = sjson.Set(highvolt_json, "md5", md5_hex)
	highvolt_json, _ = sjson.Set(highvolt_json, "sha1", sha1_hex)
	highvolt_json, _ = sjson.Set(highvolt_json, "sha256", sha256_hex)
	highvolt_json, _ = sjson.Set(highvolt_json, "full_path", full_path)
	highvolt_json, _ = sjson.Set(highvolt_json, "filename", filename)

	highvolt_json = device.Device_Info_JSON(highvolt_json)

	/* Query to see if the file has been submitted in the past */

	highvolt_query, _ := sjson.Set("", "sha256", sha256_hex)

	// json_data string, url string, http_type string, bearer_token string

	results, status_code, err := http_req.HTTP(highvolt_query, Config.Highvolt.Query, "POST", *bearerToken)

	if err != nil {
		l.Logger(l.ERROR, "Query to Highvolt failed: %v", err)
		return false
	}

	if status_code == 401 {
		l.Logger(l.NOTICE, "Highvolt bearer token expired. Getting a new one.")
		*bearerToken = jwt.PAT_Auth("highvolt", Config.Highvolt.URL, define.HIGHVOLT_VERSION, Config.Highvolt.Pat, false)
		results, _, err = http_req.HTTP(highvolt_query, Config.Highvolt.Query, "POST", *bearerToken)
		if err != nil {
			l.Logger(l.ERROR, "Query to Highvolt failed: %v", err)
			return false
		}
	}

	if results == "" {
		l.Logger(l.WARN, "No results from Highvolt server")
		return false
	}

	code := gjson.Get(results, "code").Int()

	if code == 200 {
		l.Logger(l.NOTICE, "%s has already been processed.  Skipping", sha256_hex)
		return true
	}

	l.Logger(l.NOTICE, "Submitting %s [%s] for analysis", full_path, mimetype)

	_, status_code, err = http_req.HTTP(highvolt_json, Config.Highvolt.Submit, "POST", *bearerToken)

	if err != nil {
		l.Logger(l.ERROR, "Submit to Highvolt failed: %v", err)
		return false
	}

	if status_code == 401 {
		l.Logger(l.NOTICE, "Highvolt bearer token expired. Getting a new one.")
		*bearerToken = jwt.PAT_Auth("highvolt", Config.Highvolt.URL, define.HIGHVOLT_VERSION, Config.Highvolt.Pat, false)
		_, _, err = http_req.HTTP(highvolt_json, Config.Highvolt.Submit, "POST", *bearerToken)
		if err != nil {
			l.Logger(l.ERROR, "Submit to Highvolt failed: %v", err)
			return false
		}
	}

	return true

}
