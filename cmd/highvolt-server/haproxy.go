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
	"net"
	"strings"

	"github.com/k9io/highvolt/cmd/highvolt-server/debug"
	"github.com/k9io/highvolt/cmd/highvolt-server/models"
	"github.com/k9io/highvolt/cmd/highvolt-server/queue"

	l "github.com/k9io/highvolt/internal/logger"
)

/***************************************************************************/
/* Highvolt_HAProxy_Agent - This is an option feature that runs as a       */
/* "go routine". If you are not using haproxy (https://www.haproxy.org/),  */
/* you don't need this option.  HA proxy is useful to Highvolt when you    */
/* are running a cluster of servers running your LLMs.  This "tells"       */
/* haproxy how busy the node is. This is based off the work in the queue.  */
/***************************************************************************/

func Highvolt_HAProxy_Agent() {

	addr := fmt.Sprintf(":%d", models.C.HA_Proxy.Port)

	listener, err := net.Listen("tcp", addr)

	if err != nil {
		l.Logger(l.ERROR, "HA Proxy Agent-Check Error: %v", err)
		return
	}

	defer listener.Close()

	l.Logger(l.INFO, "Highvolt HAProxy Agent listening on %s (Max Queue size: %d)", addr, models.C.Core.Max_Queue_Size)

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}

		/* Grab current load.  */

		currentJobs := queue.GetQueueSize()

		var status string

		/* If the currentJobs >= the max number for the queue,  this
		   node is full.  Stop accepting jobs */

		if currentJobs >= models.C.Core.Max_Queue_Size {
			status = "drain\n"

			/* If we are idle, let haproxy know we can accept work! */

		} else if currentJobs == 0 {
			status = "100%\n"

		} else {

			/* Calculate how "busy" we are and send that to haproxy.  For
			  			   example:  if the max queue is 20 and we are currently processing
						   5 files,  health will be %75 */

			healthPct := 100 - int((float64(currentJobs)/float64(models.C.Core.Max_Queue_Size))*100)

			/* Ensure we don't send 0% unless we mean 'down' */

			if healthPct < 5 {
				healthPct = 5
			}

			status = fmt.Sprintf("%d%%\n", healthPct)
		}

		if debug.X.Health == true {

			c_status := strings.ReplaceAll(status, "\n", "")

			//conn.RemoteAddr().String()
			l.Logger(l.DEBUG, "haproxy agent-check from %s.  Sent %s", conn.RemoteAddr().String(), c_status)

		}

		if _, err := conn.Write([]byte(status)); err != nil {
			l.Logger(l.ERROR, "HAProxy agent-check write failed: %v", err)
		}
		conn.Close()
	}
}
