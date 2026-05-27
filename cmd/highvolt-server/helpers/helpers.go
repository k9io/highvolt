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
	"github.com/k9io/highvolt/cmd/highvolt-server/debug"
	"github.com/k9io/highvolt/cmd/highvolt-server/models"

	l "github.com/k9io/highvolt/internal/logger"
)

func Check_MIME(mimetype string) (bool, string) {

	models.ConfigMu.RLock()
	imageTypes := models.C.Core.MIME_Types.Image
	pdfTypes := models.C.Core.MIME_Types.PDF
	officeTypes := models.C.Core.MIME_Types.Office
	archiveTypes := models.C.Core.MIME_Types.Archive
	textTypes := models.C.Core.MIME_Types.Text
	models.ConfigMu.RUnlock()

	for _, m := range imageTypes {

		if m == mimetype {

			if debug.X.Submit == true {

				l.Logger(l.INFO, "Detected an IMAGE [%s]", m)

			}

			return true, "IMAGE"
		}
	}

	for _, m := range pdfTypes {

		if m == mimetype {

			if debug.X.Submit == true {

				l.Logger(l.INFO, "Detected a PDF [%s]", m)

			}

			return true, "PDF"
		}
	}

	for _, m := range officeTypes {

		if m == mimetype {

			if debug.X.Submit == true {

				l.Logger(l.INFO, "Detected an Office format [%s]", m)

			}

			return true, "OFFICE"
		}
	}

	for _, m := range archiveTypes {

		if m == mimetype {

			if debug.X.Submit == true {

				l.Logger(l.INFO, "Detected an Archive [%s]", m)

			}

			return true, "ARCHIVE"

		}
	}

	for _, m := range textTypes {

		if m == mimetype {

			if debug.X.Submit == true {

				l.Logger(l.INFO, "Detected an Text [%s]", m)

			}

			return true, "TEXT"

		}
	}

	if debug.X.Submit == true {

		l.Logger(l.DEBUG, "Unknown MIME type submitted.")

	}

	return false, ""
}
