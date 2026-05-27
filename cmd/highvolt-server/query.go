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
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"

	"github.com/k9io/highvolt/cmd/highvolt-server/db"
	"github.com/k9io/highvolt/cmd/highvolt-server/helpers"

	l "github.com/k9io/highvolt/internal/logger"
)

/******************************************************************/
/* Query - Lookup if files have already been analyzed by Highvolt */
/******************************************************************/

func Query(c *gin.Context) {

	jsondata, _ := c.GetRawData()
	jsondata_s := string(jsondata)

	err := helpers.SanityCheck_Hash_Only(jsondata_s)

	if err != nil {

		l.Logger(l.ERROR, "%v", err)

		c.JSON(http.StatusBadRequest, gin.H{"status": "failed", "code": 400, "reason": err.Error()})
		c.Abort()

		return
	}

	sha256 := gjson.Get(jsondata_s, "sha256").String()

	l.Logger(l.INFO, "Query for %s from %s.", sha256, c.ClientIP())

	output, err := db.SearchBySHA256(sha256)

	if err != nil {

		l.Logger(l.ERROR, "OpenSearch query failed for %s: %v", sha256, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Query failed"})
		return

	}

	hits := gjson.Get(output, "hits.total.value").Int()

	if hits == 0 {

		l.Logger(l.INFO, "Sample %s has never been analyzed. [%s].", sha256, c.ClientIP())

		status := `{"status":"not found","code":404}`
		c.String(http.StatusOK, status)
		c.Abort()
		return

	} else {

		l.Logger(l.INFO, "%s been analyzed before [%s].", sha256, c.ClientIP())

		status := `{"status":"found","code":200}`
		c.String(http.StatusOK, status)
		c.Abort()
		return

	}

}
