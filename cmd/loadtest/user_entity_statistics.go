// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package main

import (
	"strconv"
	"time"

	"github.com/mattermost/mattermost-load-test/cmd/cmdlib"
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

func doPrintStats(c *cmdlib.CommandContext, stats *UserEntityStatistics, stopChan <-chan bool) {
	// Print statistics on timer
	statsTicker := time.NewTicker(time.Second * 3)
	defer statsTicker.Stop()

	for {
		select {
		case <-stopChan:
			return
		case <-statsTicker.C:
			c.Println("Total Errors: " + statToString(stats.TotalErrors))
			c.Println("Send Actions per second: " + statToString(stats.ActionSendRate.Rate()))
			c.Println("Recieve Actions per second: " + statToString(stats.ActionRecieveRate.Rate()))
			c.Println("Errors per second: " + statToString(stats.ErrorRate.Rate()))
		}
	}
}
