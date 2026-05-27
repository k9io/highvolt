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

	"github.com/k9io/highvolt/cmd/highvolt-server/helpers"
	"github.com/k9io/highvolt/cmd/highvolt-server/queue"

	l "github.com/k9io/highvolt/internal/logger"

	"github.com/gin-gonic/gin"
)

/***************************************************************************/
/* Submit - Gets data from the HTTP interface (gin) and passes the request */
/* to the Worker queue                                                     */
/***************************************************************************/

func Submit(c *gin.Context) {

	jsondata, _ := c.GetRawData()
	jsondata_s := string(jsondata)

	err := helpers.SanityCheck(jsondata_s)

	if err != nil {

		l.Logger(l.ERROR, "%v", err)

		c.JSON(http.StatusBadRequest, gin.H{"status": "failed", "code": 400, "reason": err.Error()})
		c.Abort()

		return
	}

	status, err := queue.WriteQueue(jsondata_s)

	if err != nil {
		l.Logger(l.ERROR, "WriteQueue failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"status": "failed", "code": 500})
		return
	}

	c.String(http.StatusOK, status)

}
