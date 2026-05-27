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
	"context"
	"sync"

	l "github.com/k9io/highvolt/internal/logger"
)

/*******************************************************************/
/* Worker - This is where we wait for incoming work.  This becomes */
/* multiple go routines                                            */
/*******************************************************************/

func Worker(ctx context.Context, id int, jobs <-chan Job, wg *sync.WaitGroup) {

	defer wg.Done()

	for {

		select {
		case <-ctx.Done():
			return

		case j, ok := <-jobs:

			if !ok { // channel closed => pool draining done
				return
			}

			l.Logger(l.NOTICE, "*** Thread %d Active ***", id)

			err := DoWork(j.QueueData)

			if err != nil {
				l.Logger(l.ERROR, "Got error on submission: %v", err)
			}
		}
	}
}
