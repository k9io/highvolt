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
	"errors"

	"github.com/k9io/highvolt/cmd/highvolt-server/models"
)

/* Store is implemented by every storage backend (OpenSearch, OpenObserve).
   SearchBySHA256 returns whether a document with that hash has already
   been indexed -- callers never see a backend's raw response shape. */

type Store interface {
	IndexDocument(document_id string, json_data string) error
	SearchBySHA256(sha256 string) (bool, error)
}

/* Writes fan out to every backend listed in 'write_backends'. Queries
   always resolve to the single backend named by 'query_backend'. The two
   are independent so a migration can dual-write while still reading from
   the trusted backend, then cut reads over once the new backend is
   validated. */

var writeStores []Store
var queryStore Store

func newStore(name string) Store {

	if name == "openobserve" {
		return openObserveStore{}
	}

	return opensearchStore{}

}

func Init_Store() {

	initialized := map[string]bool{}

	initBackend := func(name string) {

		if initialized[name] {
			return
		}

		if name == "openobserve" {
			Init_OpenObserve()
		} else {
			Init_Opensearch()
		}

		initialized[name] = true

	}

	writeStores = nil
	seenWrite := map[string]bool{}

	for _, name := range models.C.Write_Backends {

		if seenWrite[name] {
			continue
		}

		seenWrite[name] = true

		initBackend(name)
		writeStores = append(writeStores, newStore(name))

	}

	initBackend(models.C.Query_Backend)
	queryStore = newStore(models.C.Query_Backend)

}

func Index_Document(document_id string, json_data string) error {

	var errs []error

	for _, s := range writeStores {

		if err := s.IndexDocument(document_id, json_data); err != nil {
			errs = append(errs, err)
		}

	}

	return errors.Join(errs...)

}

func SearchBySHA256(sha256 string) (bool, error) {
	return queryStore.SearchBySHA256(sha256)
}
