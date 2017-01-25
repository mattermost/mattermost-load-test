// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package loadtest

type UIBuffer struct {
	nodes       []interface{}
	maxSize     int
	currentSize int
	head        int
	tail        int
}

func NewUIBuffer(size int) *UIBuffer {
	return &UIBuffer{
		nodes:       make([]interface{}, size, size),
		maxSize:     size,
		currentSize: 0,
		head:        0,
	}
}

func (b *UIBuffer) Add(item interface{}) {
	b.nodes[b.head] = item
	b.head += 1
	if b.currentSize != b.maxSize {
		b.currentSize += 1
	}
	if b.head == b.maxSize {
		b.head = 0
	}
}

func (b *UIBuffer) GetBufInt() []int {
	output := make([]int, 0, b.currentSize)

	index := 0
	if b.maxSize == b.currentSize {
		index = (b.head + 1) % b.maxSize
	}
	for size := 0; size < b.currentSize; size++ {
		if index == b.maxSize {
			index = 0
		}
		output = append(output, b.nodes[index].(int))
		index++
	}

	return output
}

func (b *UIBuffer) GetBufString() []string {
	output := make([]string, 0, b.currentSize)

	index := 0
	if b.maxSize == b.currentSize {
		index = (b.head + 1) % b.maxSize
	}
	for size := 0; size < b.currentSize; size++ {
		if index == b.maxSize {
			index = 0
		}
		output = append(output, b.nodes[index].(string))
		index++
	}

	return output
}
