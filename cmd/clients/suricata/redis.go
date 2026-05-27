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
	"context"
	"fmt"
	"os"

	l "github.com/k9io/highvolt/internal/logger"

	"github.com/redis/go-redis/v9"
)

/**************************************************************************/
/* Redis_Init() - Initialize Redis connection.  This connection is to the */
/* Suricata Sub/Pub instance.  It is not related to Highvolts connection  */
/**************************************************************************/

func Redis_Init() {

	ctx := context.Background()

	/* Connect to Clients Redis DB */

	connect_string := fmt.Sprintf("%s:%d", Config.Redis.Host, Config.Redis.Port)

	if Debug.Redis == true {
		l.Logger(l.DEBUG, "[REDIS] Connection string: %s", connect_string)
	}

	Config.Redis.Client = redis.NewClient(&redis.Options{

		/* 2026/02/26 - While a password can be supplied,  Suricata doesn't currently
		   support authentication. The best place for this to connect is on localhost.
		   Because of that,  we don't bother with TLS (yet) */

		// TLSConfig: &tls.Config{
		//       MinVersion: tls.VersionTLS12,
		// },

		Addr:     connect_string,
		Username: Config.Redis.Username,
		Password: Config.Redis.Password,
		DB:       Config.Redis.Database,
	})

	/* Check the Redis Connection */

	_, err := Config.Redis.Client.Ping(ctx).Result()

	if err != nil {

		l.Logger(l.ERROR, "Error connecting to Suricata Redis: %v", err)
		os.Exit(1)

	}

}
