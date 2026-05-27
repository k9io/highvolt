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

package models

import (
	"time"
	)

/* Configuration from env variables */

type Environment_Struct struct {

	RUNAS	    string

        JSONAIR_PAT string

        JSONAIR_URL  string
        JSONAIR_TYPE string
        JSONAIR_NAME string

	CONFIG_SLEEP       time.Duration
	RELOAD_SLEEP	   time.Duration
	DEBUG_SLEEP	   time.Duration

        JWTTokenSecret  []byte
        JWTTokenExpire  int

	HIGHVOLT_PAT	string


}

var Env Environment_Struct

