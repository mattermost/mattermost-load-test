// Copyright (c) 2016 Spinpunch, Inc. All Rights Reserved.
// See License.txt for license information.

package autocreation

import "sync"

func ThreadSplit(arrayLen int, numThreads int, action func(int)) {
	var wg sync.WaitGroup
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
			}
			wg.Done()
		}(threadNum)
	}
	wg.Wait()
}
