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

// patAuth exchanges a PAT for a JWT bearer token from the Highvolt or JSONAIR
// auth endpoint. This is a Lambda-local version of internal/jwt.PAT_Auth that
// uses the standard log package instead of the syslog-backed internal logger
// (the local syslog socket is not available inside Lambda containers).

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/k9io/highvolt/internal/http_req"
)

type jwtResponse struct {
	Access_Token string `json:"access_token"`
}

func patAuth(ltype, baseURL, version, pat string) string {
	body, err := json.Marshal(map[string]string{"token": pat})
	if err != nil {
		log.Fatalf("[ERROR] Cannot marshal PAT request for %s: %v", ltype, err)
	}

	authURL := fmt.Sprintf("%s/api/%s/%s/auth/token", baseURL, version, ltype)

	result, status, err := http_req.HTTP(string(body), authURL, "POST", "")
	if err != nil {
		log.Fatalf("[ERROR] PAT auth request failed for %s: %v", ltype, err)
	}
	if status != 200 {
		log.Fatalf("[ERROR] PAT auth returned HTTP %d for %s at %s", status, ltype, authURL)
	}

	var jwt jwtResponse
	if err := json.Unmarshal([]byte(result), &jwt); err != nil {
		log.Fatalf("[ERROR] Cannot parse JWT response for %s: %v", ltype, err)
	}
	if jwt.Access_Token == "" {
		log.Fatalf("[ERROR] Empty access_token in JWT response for %s", ltype)
	}

	return jwt.Access_Token
}
