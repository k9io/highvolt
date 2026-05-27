/**
 ** Copyright (C) 2026 Key9, Inc <k9.io>
 ** Copyright (C) 2026 Champ Clark III <cclark@k9.io>
 **
 ** This file is part of the JSONAir.
 **
 ** This source code is licensed under the MIT license found in the
 ** LICENSE file in the root directory of this source tree.
 **
 **/

package jwt

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/k9io/highvolt/internal/http_req"

	l "github.com/k9io/highvolt/internal/logger"
)

type JWT_Struct struct {
	Access_Token string
}

type patRequest struct {
	Token string `json:"token"`
}

func PAT_Auth(ltype string, url string, version string, pat string, debug bool) string {

	var err error
	var JWT *JWT_Struct

	patReq := patRequest{Token: pat}
	patBytes, err := json.Marshal(patReq)

	if err != nil {
		l.Logger(l.ERROR, "Cannot marshal PAT request: %v", err)
		os.Exit(1)
	}

	auth_url := fmt.Sprintf("%s/api/%s/%s/auth/token", url, version, ltype)

	results, status_code, err := http_req.HTTP(string(patBytes), auth_url, "POST", "")

	if err != nil {

		l.Logger(l.ERROR, "http_req.HTTP: %v", err)
		return ""

	}

	if debug == true { 

           l.Logger(l.DEBUG, "[HTTP] %s to %s", ltype, url)
           l.Logger(l.DEBUG, "[HTTP] Results: %s", results)
           l.Logger(l.DEBUG, "[HTTP] Status code: %d", status_code)

	}

	if status_code != http.StatusOK {

		l.Logger(l.ERROR, "Error getting Bearer Token.  HTTP Status: %v", status_code)
		os.Exit(1)

	}

	err = json.Unmarshal([]byte(results), &JWT)

	if err != nil {

		l.Logger(l.ERROR, "Cannot parse Bearer Token: %v", err)
		os.Exit(1)

	}

	if JWT.Access_Token == "" {

		l.Logger(l.ERROR, "Unable to find the 'access_token'.")
		os.Exit(1)

	}

	return JWT.Access_Token

}
