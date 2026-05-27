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

package queue

import (
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/k9io/highvolt/cmd/highvolt-server/helpers"
	"github.com/k9io/highvolt/cmd/highvolt-server/processors"
	"github.com/k9io/highvolt/cmd/highvolt-server/db"


	l "github.com/k9io/highvolt/internal/logger"
)

/***************************************************************************/
/* DoWork - This is part of the "work load" from pre-spawned "go routines" */
/***************************************************************************/

func DoWork(jdata []byte) error {

	var err error

	jsondata := string(jdata)

	ai_data := "{}"

	file_data := gjson.Get(jsondata, "file_data").String()

	if file_data == "" {

		l.Logger(l.DEBUG, "'file_data' JSON data not found.")
		l.Logger(l.DEBUG, "DEBUG: %s", jsondata)
		return nil

	}

	mimetype := gjson.Get(jsondata, "mimetype").String()

	if mimetype == "" {

		l.Logger(l.DEBUG, "'mimetype' JSON data not found.")
		return nil

	}

	sha256 := gjson.Get(jsondata, "sha256").String()

	if sha256 == "" {

		l.Logger(l.DEBUG, "'sha256' JSON data not found.")
		return nil

	}

	/* Is the MIME something we want to analyze? */

	check_mime, mime_ret := helpers.Check_MIME(mimetype)

	if check_mime == true {

		ai_data = processors.Submit_Data(file_data, mimetype, mime_ret)

	} else {

		l.Logger(l.WARN, "Magic type: '%s' is not supported.  Skipping sample.", mimetype)
		return nil
	}

	if ai_data == "" {
		ai_data = "{}"
	}

	jsondata, err = sjson.SetRaw(jsondata, "llm", ai_data)

	if err != nil {

		l.Logger(l.WARN, "Could not add file data for LLM analysis. [%s]", err)

	}

	if err = db.Index_Document(sha256, jsondata); err != nil {

		l.Logger(l.ERROR, "Failed to index document %s: %v", sha256, err)
		return err

	}

	return nil
}


