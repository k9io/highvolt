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
	"github.com/gin-gonic/gin"
	l "github.com/k9io/highvolt/internal/logger"
)

/***********************************************************************/
/* HTTP_Logger() - Used for HTTP logging when "gin" is set to anything */
/* other than "production".  Otherwise,  we don't log it this way.     */
/***********************************************************************/

func HTTP_Logger() gin.HandlerFunc {
	return func(c *gin.Context) {

		clientIP := c.ClientIP()
		l.Logger(l.INFO, "%s %s %s", c.Request.Method, c.Request.URL.Path, clientIP)
		c.Next()
	}
}
