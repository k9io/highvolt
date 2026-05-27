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
//	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/k9io/highvolt/cmd/highvolt-server/models"

	"github.com/facette/natsort"

	l "github.com/k9io/highvolt/internal/logger"
)

type Job struct {
	ID        int
	QueueData []byte
}

var jobs chan Job
var jobID atomic.Int32
var queue_size atomic.Int32

var CancelQueue context.CancelFunc

/*****************************************************************************/
/* Init_Queue - Start "worker" go routines.  Set these in the background for */
/* future use.                                                               */
/*****************************************************************************/

func Init_Queue() {

	var ctx context.Context

	jobs = make(chan Job, models.C.Core.Max_Workers)

	ctx, CancelQueue = context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(models.C.Core.Max_Workers)

	l.Logger(l.INFO, "Starting up %d Worker threads.", models.C.Core.Max_Workers)

	for i := 0; i < models.C.Core.Max_Workers; i++ {
		go Worker(ctx, i, jobs, &wg)
	}

}

/*************************************************************************/
/* WriteQueue - This writes the incoming JSON from Highvolt clients to a */
/* queue directory.  This way, if the LLM is overloaded,  we don't drop  */
/* data.  We store with the epoch time to keep some sort of order to the */
/* the submissions                                                       */
/*************************************************************************/

func WriteQueue(jsondata string) (string, error) {

	/* We use the epoch to keep some sort of "order" to submissions */

	queue_size.Add(1)

	epoch := time.Now().Unix()

	tmp := fmt.Sprintf("highvolt-queue-%d-*.tmp", epoch)

	Q, err := os.CreateTemp(models.C.Core.Queue_Directory, tmp)

	if err != nil {

		l.Logger(l.ERROR, "Error create temp Queue File: %v. Analysis faile", err)
		//return errors.New(`{"stanus":"failed","reason":"Cannot create temp directory","code":500}`),nil
		return `{"status":"failed","reason":"Cannot create temp directory","code":500}`, nil


	}

	if _, err = Q.WriteString(jsondata); err != nil {
		l.Logger(l.ERROR, "Error writing to queue file: %v", err)
		Q.Close()
		os.Remove(Q.Name())
		queue_size.Add(-1)
		return `{"status":"failed","reason":"Cannot write to queue file","code":500}`, nil
	}
	Q.Close()

	F := strings.TrimSuffix(Q.Name(), ".tmp") + ".data"

	/* Atomic rename, this way we don't get half submission issues */

	if err = os.Rename(Q.Name(), F); err != nil {
		l.Logger(l.ERROR, "Failed to rename queue file %s: %v", Q.Name(), err)
		os.Remove(Q.Name())
		queue_size.Add(-1)
		return `{"status":"failed","reason":"Cannot commit queue file","code":500}`, nil
	}

	return `{"status":"success","code":202}`, nil

}

/***************************************************************************/
/* Monitor_Queue - Watches the queue directory for new entries.  This is a */
/* go routine in the background.  When a new entry is detected,  we submit */
/* it to the Workers.  If no workers are avaliable, it will block until    */
/* one becomes free.                                                       */
/***************************************************************************/

func Monitor_Queue() {

	/* Why not use iNotify?  Because not all operating systems support it. */

	for {

		pattern := models.C.Core.Queue_Directory + "/*.data" // We always convert PDF -> PNG

		matches, err := filepath.Glob(pattern)

		if err != nil {
			l.Logger(l.ERROR, "Error with pattern: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		natsort.Sort(matches) // Try and retain the order of the queue submissions.

		for _, i := range matches {

			l.Logger(l.NOTICE, "Got %s from queue.", i)

			jsondata, err := os.ReadFile(i)

			if err != nil {
				l.Logger(l.ERROR, "Error reading queue file %s: %v. Removing.", i, err)
				os.Remove(i)
				queue_size.Add(-1)
				continue
			}

			/* Send job data to a free worker */

			jobs <- Job{ID: int(jobID.Add(1)), QueueData: jsondata}

			err = os.Remove(i)

			queue_size.Add(-1)

			if err != nil {
				l.Logger(l.ERROR, "Error deleting queue file %s: %v", i, err)
			}

		}

		time.Sleep(1 * time.Second) /* Prevent 100% CPU usage */
	}

}

/**************************************************/
/* Shutdown_Queue - Called when Highvolt shutdown */
/**************************************************/

func Shutdown_Queue() {

	CancelQueue()

}

/****************/
/* GetQueueSize */
/****************/

func GetQueueSize() int {

	return int(queue_size.Load())

}
