// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package loadtest

import (
	"fmt"
	"sync"
)

func PrintCounter(counter chan int, total int) {
	i := 0
	for {
		select {
		case _, ok := <-counter:

			if !ok {
				// An error occurred and we shutting down
				return
			}

			i = i + 1
			print(fmt.Sprintf("\r %v/%v", i, total))
			if i == total {
				return
			}

		}

	}
}

func ThreadSplit(arrayLen int, numThreads int, statusPrinter func(chan int, int), action func(int)) {
	var wg sync.WaitGroup
	counter := make(chan int)
	go statusPrinter(counter, arrayLen)
	wg.Add(numThreads)
	for threadNum := 0; threadNum < numThreads; threadNum++ {
		go func(threadNum int) {
			var end int
			if threadNum == numThreads-1 {
				end = arrayLen
			} else {
				end = (arrayLen / numThreads) * (threadNum + 1)
			}
			start := (arrayLen / numThreads) * threadNum
			for i := start; i < end; i++ {
				action(i)
				counter <- 1
			}
			wg.Done()
		}(threadNum)
	}
	wg.Wait()
	close(counter)
}
