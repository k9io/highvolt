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

package processors

import (
	"github.com/k9io/highvolt/cmd/highvolt-server/llm"
	"github.com/k9io/highvolt/cmd/highvolt-server/models"

	l "github.com/k9io/highvolt/internal/logger"
	"github.com/k9io/highvolt/internal/util"
)

func Submit_Data(file_data string, mimetype string, mime_ret string) string {

	var ai_data string

	switch mime_ret {

	case "IMAGE":

		file_size := util.CalculateBase64DecodedSize(file_data)

		models.ConfigMu.RLock()
		minImageSize := models.C.Core.Minimum_Image_Size
		models.ConfigMu.RUnlock()

		if file_size <= minImageSize {

			l.Logger(l.NOTICE, "Image file size is below minimum of %d [size: %d]. Rejecting sample.", minImageSize, file_size)

		} else {

			ai_data = llm.Submit_AI(file_data, mimetype, "IMAGE")

		}

	case "TEXT":

		ai_data = llm.Submit_AI(file_data, mimetype, "TEXT" )

	case "PDF":

		ai_data = Submit_PDF(file_data)

	case "OFFICE":

		ai_data = Submit_Office(file_data)

	case "ARCHIVE":

		ai_data = Submit_Archive(file_data)

	default:

		l.Logger(l.ERROR, "Unknown submission type. [%s]", mimetype)

	}

	return ai_data
}
