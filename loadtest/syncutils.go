// Copyright (c) 2017 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information

package loadtest

import (
	"sync"
	"time"
)

func waitWithTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan bool)
	go func() {
		wg.Wait()
		close(c)
	}()
	select {
	// Everything is OK
	case <-c:
		return true
	// We timed out
	case <-time.After(timeout):
		return false
	}
}
