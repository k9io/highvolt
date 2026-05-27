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

package util

import (
	"strings"

	"github.com/gabriel-vasile/mimetype"

	l "github.com/k9io/highvolt/internal/logger"
)

func CalculateBase64DecodedSize(b64 string) int {

	l := len(b64)

	if l == 0 {
		return 0
	}

	// 1. Calculate the raw size based on the 3/4 ratio

	size := (l * 3) / 4

	// 2. Adjust for padding at the end of the string

	if strings.HasSuffix(b64, "==") {
		size -= 2
	} else if strings.HasSuffix(b64, "=") {
		size -= 1
	}

	return size
}

func GetFileMagic(file string) string {

	mtype, err := mimetype.DetectFile(file)

	if err != nil {

		l.Logger(l.WARN, "Error getting MIME type [%v].  Skipping analysis.", err)
		return ""
	}

	/* Sometimes, thinks like 'text/plain; charset=UTF-8'.  We don't care about the charset */

	pureMime := strings.Split(mtype.String(), ";")[0]

	return pureMime

}


