// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package main

import (
	"strconv"
	"time"

	"github.com/paulbellamy/ratecounter"
)

type UserEntityStatistics struct {
	TotalErrors               int64
	TotalEntitiesActive       int64
	TotalEntitiesLaunching    int64
	TotalEntitiesFailedLaunch int64
	TotalEntitiesFailedActive int64
	TotalEntitesStopped       int64

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
	switch report.Status {
	case STATUS_ACTION_SEND:
		stats.ActionSendRate.Incr(1)
	case STATUS_ACTION_RECIEVE:
		stats.ActionRecieveRate.Incr(1)
	case STATUS_ERROR:
		stats.ErrorRate.Incr(1)
		stats.TotalErrors += 1
	case STATUS_ACTIVE:
		stats.TotalEntitiesActive += 1
		stats.TotalEntitiesLaunching -= 1
	case STATUS_LAUNCHING:
		stats.TotalEntitiesLaunching += 1
	case STATUS_FAILED_LAUNCH:
		stats.TotalEntitiesLaunching -= 1
		stats.TotalEntitiesFailedLaunch += 1
	case STATUS_FAILED_ACTIVE:
		stats.TotalEntitiesActive -= 1
		stats.TotalEntitiesFailedActive += 1
	case STATUS_STOPPED:
		stats.TotalEntitesStopped += 1
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
			out.Println("------- Entity Stats --------")
			out.Println("Total Entities Active: " + statToString(stats.TotalEntitiesActive))
			out.Println("Total Entities Launching: " + statToString(stats.TotalEntitiesLaunching))
			out.Println("Total Errors: " + statToString(stats.TotalErrors))
			out.Println("Total Entities Failed to Launch: " + statToString(stats.TotalEntitiesFailedLaunch))
			out.Println("Total Entities Failed While Active: " + statToString(stats.TotalEntitiesFailedActive))
			out.Println("Send Actions per second: " + statToString(stats.ActionSendRate.Rate()))
			out.Println("Recieve Actions per second: " + statToString(stats.ActionRecieveRate.Rate()))
			out.Println("Errors per second: " + statToString(stats.ErrorRate.Rate()))
			out.Println("-----------------------------")
		}
	}
}
