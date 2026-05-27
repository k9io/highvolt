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
	"os"
	"syscall"

	//	"github.com/k9io/highvolt/cmd/highvolt-server/config"

	//	"github.com/k9io/highvolt/cmd/highvolt-server/db"

	l "github.com/k9io/highvolt/internal/logger"
)

func SigHandler(c chan os.Signal) {

	for {
		sig := <-c

		l.Logger(l.NOTICE, "Caught signal %v", sig)

		switch sig {

		case syscall.SIGINT, syscall.SIGTERM, syscall.SIGABRT:

			/* Opensearch handles exit itself */

			//db.CloseRedis()

			os.Exit(0)

		case syscall.SIGHUP:

			//LoadConfig()

			// Reload here
			// os.Exit(0)

		}
	}
}
