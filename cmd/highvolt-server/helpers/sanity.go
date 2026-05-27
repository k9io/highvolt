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

package helpers

import (
	"errors"
	"regexp"

	"github.com/tidwall/gjson"
)

var (
	reMD5    = regexp.MustCompile(`^[0-9a-fA-F]{32}$`)
	reSHA1   = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)
	reSHA256 = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)
)

func validateHashes(jsondata string) error {
	if v := gjson.Get(jsondata, "md5"); v.Exists() && !reMD5.MatchString(v.String()) {
		return errors.New("invalid md5 format")
	}
	if v := gjson.Get(jsondata, "sha1"); v.Exists() && !reSHA1.MatchString(v.String()) {
		return errors.New("invalid sha1 format")
	}
	if v := gjson.Get(jsondata, "sha256"); v.Exists() && !reSHA256.MatchString(v.String()) {
		return errors.New("invalid sha256 format")
	}
	return nil
}

/* Validate that incoming request has the minimum number of JSON objects to analyze! */

func SanityCheck(jsondata string) error {

	res := gjson.Get(jsondata, "timestamp")

	if !res.Exists() {
		return errors.New("'timestamp' not found")
	}

	res1 := gjson.Get(jsondata, "md5")
	res2 := gjson.Get(jsondata, "sha1")
	res3 := gjson.Get(jsondata, "sha256")

	if !res1.Exists() && !res2.Exists() && !res3.Exists() {
		return errors.New("no hash found")
	}

	if err := validateHashes(jsondata); err != nil {
		return err
	}

	res = gjson.Get(jsondata, "file_data")

	if !res.Exists() {
		return errors.New("'file_data' not found")
	}

	/*
	res = gjson.Get(jsondata, "magic")

	if !res.Exists() {
		return errors.New("'magic' not found")
	}
	*/

	return nil

}

func SanityCheck_Hash_Only(jsondata string) error {

	res1 := gjson.Get(jsondata, "md5")
	res2 := gjson.Get(jsondata, "sha1")
	res3 := gjson.Get(jsondata, "sha256")

	if !res1.Exists() && !res2.Exists() && !res3.Exists() {
		return errors.New("no hash found")
	}

	return validateHashes(jsondata)
}
