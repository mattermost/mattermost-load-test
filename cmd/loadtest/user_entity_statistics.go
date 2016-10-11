// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package main

import (
	"strconv"
	"time"

	"github.com/paulbellamy/ratecounter"
)

type UserEntityStatistics struct {
	TotalErrors int64

	ErrorRate         *ratecounter.RateCounter
	ActionSendRate    *ratecounter.RateCounter
	ActionRecieveRate *ratecounter.RateCounter
}

func NewUserEntityStatistics(interval time.Duration) *UserEntityStatistics {
	return &UserEntityStatistics{
		TotalErrors:       0,
		ErrorRate:         ratecounter.NewRateCounter(interval),
		ActionSendRate:    ratecounter.NewRateCounter(interval),
		ActionRecieveRate: ratecounter.NewRateCounter(interval),
	}
}

func (stats *UserEntityStatistics) updateEntityStatistics(report UserEntityStatusReport) {
	if report.Status == STATUS_ACTION_SEND {
		stats.ActionSendRate.Incr(1)
	} else if report.Status == STATUS_ACTION_RECIEVE {
		stats.ActionRecieveRate.Incr(1)
	} else if report.Status == STATUS_ERROR {
		stats.ErrorRate.Incr(1)
		stats.TotalErrors += 1
	}
}

func statToString(stat int64) string {
	return strconv.Itoa(int(stat))
}

func doPrintStats(out UserEntityLogger, stats *UserEntityStatistics, stopChan <-chan bool) {
	// Print statistics on timer
	statsTicker := time.NewTicker(time.Second * 3)
	defer statsTicker.Stop()

	for {
		select {
		case <-stopChan:
			return
		case <-statsTicker.C:
			out.Println("Total Errors: " + statToString(stats.TotalErrors))
			out.Println("Send Actions per second: " + statToString(stats.ActionSendRate.Rate()))
			out.Println("Recieve Actions per second: " + statToString(stats.ActionRecieveRate.Rate()))
			out.Println("Errors per second: " + statToString(stats.ErrorRate.Rate()))
		}
	}
}
