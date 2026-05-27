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

package db

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"

	//	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/k9io/highvolt/cmd/highvolt-server/debug"
	"github.com/k9io/highvolt/cmd/highvolt-server/models"

	l "github.com/k9io/highvolt/internal/logger"
)

var Opensearch_Client *opensearch.Client

func Init_Opensearch() {

	var err error

	if models.C.Opensearch.TLS_Skip_Verify {
		l.Logger(l.WARN, "OpenSearch TLS certificate verification is DISABLED. Do not use in production.")
	}

	Opensearch_Client, err = opensearch.NewClient(opensearch.Config{
		Addresses: []string{models.C.Opensearch.URL},
		Username:  models.C.Opensearch.Username,
		Password:  models.C.Opensearch.Password,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: models.C.Opensearch.TLS_Skip_Verify,
			},
		},
	})

	if err != nil {

		l.Logger(l.ERROR, "Error creating OpenSearch client: %v", err)
		os.Exit(1)

	}

}

func Index_Document(document_id string, json_data string) error {

	var err error

	/* Remove the "file_data".  We don't want to store that,  only the
	   file information and AI data */

	json_data, _ = sjson.Delete(json_data, "file_data")

	document := strings.NewReader(json_data)

	req := opensearchapi.IndexRequest{
		Index:      models.C.Opensearch.Index,
		DocumentID: document_id,
		Body:       document,
	}

	insertResponse, err := req.Do(context.Background(), Opensearch_Client)

	if err != nil {

		l.Logger(l.ERROR, "Error in req.Do() : %v", err)
		return err

	}

	defer insertResponse.Body.Close()

	bodyBytes, err := io.ReadAll(insertResponse.Body)

	if err != nil {

		l.Logger(l.ERROR, "Error reading respond body from Opensearch: %v", err)
		return err

	}

	if insertResponse.IsError() {

		l.Logger(l.ERROR, "OpenSearch index request failed (status %d): %s", insertResponse.StatusCode, string(bodyBytes))
		return fmt.Errorf("opensearch index request failed with status %d", insertResponse.StatusCode)

	}

	if debug.X.Opensearch == true {

		l.Logger(l.DEBUG, "Response from Opensearch: %s", string(bodyBytes))

	}

	return nil

}

/* DEBUG: This needs to support MD5, SHA1 and SHA256 */

func SearchBySHA256(sha256 string) (string, error) {

	if debug.X.Opensearch == true {

		l.Logger(l.DEBUG, "Doing SHA256 lookup for %s", sha256)
	}

	queryDoc := map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]interface{}{
				"sha256": map[string]interface{}{
					"value": sha256,
				},
			},
		},
	}

	queryBytes, err := json.Marshal(queryDoc)
	if err != nil {
		l.Logger(l.ERROR, "Error marshaling search query: %v", err)
		return "", err
	}

	searchReq := opensearchapi.SearchRequest{
		Index: []string{models.C.Opensearch.Index},
		Body:  strings.NewReader(string(queryBytes)),
	}

	ctx := context.Background()

	res, err := searchReq.Do(ctx, Opensearch_Client)

	if err != nil {

		l.Logger(l.ERROR, "Error executing search: %v", err)
		return "", err

	}

	bodyBytes, err := io.ReadAll(res.Body)

	if err != nil {

		l.Logger(l.ERROR, "Cannot read response body from Opensearch: %v", err)
		return "", err

	}

	sb := string(bodyBytes)

	if res.IsError() {

		if res.StatusCode == 404 {

			l.Logger(l.WARN, "Index '%s' not found. No worries.....", models.C.Opensearch.Index)

		} else {

			l.Logger(l.WARN, "Opensearch returned status of %v", res.StatusCode)

		}
	}


	if debug.X.Opensearch == true {

	l.Logger(l.DEBUG, "Return from Opensearch for SHA256 %s: %s", sha256, sb)

	}

	return sb, nil
}
