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
	"fmt"

	"github.com/gabriel-vasile/mimetype"
	l "github.com/k9io/highvolt/internal/logger"
)

func MIMEType(file string, sha256 string, mimeTypes []string) (error, bool, string) {

	mtype, err := mimetype.DetectFile(file)

	if err != nil {

		return fmt.Errorf("Error getting MIME type [%v].  Skipping %s.", err, sha256), false, ""

	}

	for _, m := range mimeTypes {

		if m == mtype.String() {

			l.Logger(l.NOTICE, "Got MIME type '%s' for %s.", m, sha256)
			return nil, true, m

		}

	}

	l.Logger(l.WARN, "MIME type '%s' not in our list for %s.  Skipping analysis.", mtype, sha256)

	return nil, false, ""
}
