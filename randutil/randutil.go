// Copyright (c) 2017 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information

package randutil

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"reflect"
)

type Choice struct {
	Weight int
	Item   interface{}
}

func IntRange(min, max int) (int, error) {
	var result int
	switch {
	case min > max:
		// Fail with error
		return result, fmt.Errorf("bad params")
	case max == min:
		result = max
	case max > min:
		maxRand := max - min
		b, err := rand.Int(rand.Reader, big.NewInt(int64(maxRand)))
		if err != nil {
			return result, err
		}
		result = min + int(b.Int64())
	}
	return result, nil
}

// Shuffle funtion from: https://stackoverflow.com/questions/12264789/shuffle-array-in-go
func Shuffle(slice interface{}) {
	rv := reflect.ValueOf(slice)
	swap := reflect.Swapper(slice)
	length := rv.Len()
	for i := length - 1; i > 0; i-- {
		j, _ := IntRange(0, i+1)
		swap(i, j)
	}
}

// Modified version of weighted choice from https://github.com/jmcvetta/randutil
func WeightedChoice(choices []Choice) (Choice, error) {
	// Based on this algorithm:
	//     http://eli.thegreenplace.net/2010/01/22/weighted-random-generation-in-python/
	var ret Choice

	if len(choices) == 0 {
		return ret, fmt.Errorf("Was given no choices! %v", choices)
	}
	if len(choices) == 1 {
		return choices[0], nil
	}

	sum := 0
	for _, c := range choices {
		sum += c.Weight
	}
	r, err := IntRange(0, sum)
	if err != nil {
		return ret, err
	}
	for _, c := range choices {
		r -= c.Weight
		if r < 0 {
			return c, nil
		}
	}
	err = errors.New("Internal error - code should not reach this point")
	return ret, err
}
