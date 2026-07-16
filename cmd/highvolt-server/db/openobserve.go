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
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/k9io/highvolt/cmd/highvolt-server/debug"
	"github.com/k9io/highvolt/cmd/highvolt-server/models"

	l "github.com/k9io/highvolt/internal/logger"
)

var OpenObserve_Client *http.Client

type openObserveStore struct{}

func Init_OpenObserve() {

	if models.C.OpenObserve.TLS_Skip_Verify {
		l.Logger(l.WARN, "OpenObserve TLS certificate verification is DISABLED. Do not use in production.")
	}

	OpenObserve_Client = &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: models.C.OpenObserve.TLS_Skip_Verify,
			},
		},
	}

}

/* IndexDocument appends a record to the configured OpenObserve stream via
   its native JSON ingestion API. Unlike OpenSearch, OpenObserve has no
   upsert-by-ID concept -- every submission is appended as a new record,
   so re-submitting the same SHA256 produces multiple records rather than
   overwriting one. 'document_id' is accepted only for interface parity
   with the OpenSearch backend and is not sent to OpenObserve. */

func (openObserveStore) IndexDocument(document_id string, json_data string) error {

	json_data, _ = sjson.Delete(json_data, "file_data")

	body := "[" + json_data + "]"

	url := fmt.Sprintf("%s/api/%s/%s/_json", models.C.OpenObserve.URL, models.C.OpenObserve.Organization, models.C.OpenObserve.Stream)

	req, err := http.NewRequest("POST", url, bytes.NewReader([]byte(body)))

	if err != nil {

		l.Logger(l.ERROR, "Error building OpenObserve ingest request: %v", err)
		return err

	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(models.C.OpenObserve.Username, models.C.OpenObserve.Password)

	resp, err := OpenObserve_Client.Do(req)

	if err != nil {

		l.Logger(l.ERROR, "Error indexing document into OpenObserve: %v", err)
		return err

	}

	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)

	if err != nil {

		l.Logger(l.ERROR, "Error reading response body from OpenObserve: %v", err)
		return err

	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {

		l.Logger(l.ERROR, "OpenObserve ingest request failed (status %d): %s", resp.StatusCode, string(bodyBytes))
		return fmt.Errorf("openobserve ingest request failed with status %d", resp.StatusCode)

	}

	failed := gjson.GetBytes(bodyBytes, "status.0.failed").Int()

	if failed > 0 {

		l.Logger(l.ERROR, "OpenObserve reported %d failed record(s) for document %s: %s", failed, document_id, string(bodyBytes))
		return fmt.Errorf("openobserve reported %d failed record(s)", failed)

	}

	if debug.X.OpenObserve == true {

		l.Logger(l.DEBUG, "Response from OpenObserve: %s", string(bodyBytes))

	}

	return nil

}

/* SearchBySHA256 issues a SQL search against the configured OpenObserve
   stream. OpenObserve is time-partitioned and every search requires a
   time range, unlike OpenSearch's timeless term query -- there is no
   equivalent of a GET-by-document-ID. We scan from the Unix epoch through
   now; at this deployment's scale (millions, not billions, of documents)
   OpenObserve's per-file min/max and bloom-filter stats let it prune
   partitions that can't contain a match rather than scanning every row. */

/* DEBUG: This needs to support MD5, SHA1 and SHA256 */

func (openObserveStore) SearchBySHA256(sha256 string) (bool, error) {

	if debug.X.OpenObserve == true {

		l.Logger(l.DEBUG, "Doing SHA256 lookup for %s", sha256)
	}

	sql := fmt.Sprintf(`SELECT sha256 FROM "%s" WHERE sha256='%s' LIMIT 1`, models.C.OpenObserve.Stream, sha256)

	queryDoc := map[string]interface{}{
		"query": map[string]interface{}{
			"sql":        sql,
			"start_time": int64(0),
			"end_time":   time.Now().UnixMicro(),
			"from":       0,
			"size":       1,
		},
	}

	queryBytes, err := json.Marshal(queryDoc)

	if err != nil {

		l.Logger(l.ERROR, "Error marshaling OpenObserve search query: %v", err)
		return false, err

	}

	url := fmt.Sprintf("%s/api/%s/_search", models.C.OpenObserve.URL, models.C.OpenObserve.Organization)

	req, err := http.NewRequest("POST", url, bytes.NewReader(queryBytes))

	if err != nil {

		l.Logger(l.ERROR, "Error building OpenObserve search request: %v", err)
		return false, err

	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(models.C.OpenObserve.Username, models.C.OpenObserve.Password)

	resp, err := OpenObserve_Client.Do(req)

	if err != nil {

		l.Logger(l.ERROR, "Error executing OpenObserve search: %v", err)
		return false, err

	}

	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)

	if err != nil {

		l.Logger(l.ERROR, "Cannot read response body from OpenObserve: %v", err)
		return false, err

	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {

		l.Logger(l.WARN, "OpenObserve search returned status %d: %s", resp.StatusCode, string(bodyBytes))
		return false, fmt.Errorf("openobserve search request failed with status %d", resp.StatusCode)

	}

	if debug.X.OpenObserve == true {

		l.Logger(l.DEBUG, "Return from OpenObserve for SHA256 %s: %s", sha256, string(bodyBytes))

	}

	total := gjson.GetBytes(bodyBytes, "total").Int()

	return total > 0, nil

}
