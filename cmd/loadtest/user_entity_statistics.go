// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package main

import (
	"strconv"
	"time"

	"github.com/paulbellamy/ratecounter"
)

type UserEntityStat map[string]int64

func (stat UserEntityStat) Modify(entity *UserEntityConfig, ammount int64) {
	if val, ok := stat[entity.SubEntityName]; ok {
		stat[entity.SubEntityName] = val + ammount
	} else {
		stat[entity.SubEntityName] = ammount
	}
}

func (stat UserEntityStat) ToString(represents string) string {
	out := represents + ": "
	if len(stat) > 0 {
		out += "\n"
		for subentity, value := range stat {
			out += "    " + subentity + ": " + statToString(value) + "\n"
		}
	} else {
		out += "0\n"
	}
	return out
}

type UserEntityStatistics struct {
	TotalErrors               UserEntityStat
	TotalEntitiesActive       UserEntityStat
	TotalEntitiesLaunching    UserEntityStat
	TotalEntitiesFailedLaunch UserEntityStat
	TotalEntitiesFailedActive UserEntityStat
	TotalEntitiesStopped      UserEntityStat

	ErrorRate         *ratecounter.RateCounter
	ActionSendRate    *ratecounter.RateCounter
	ActionRecieveRate *ratecounter.RateCounter
}

func NewUserEntityStatistics(interval time.Duration) *UserEntityStatistics {
	return &UserEntityStatistics{
		TotalErrors:               make(UserEntityStat),
		TotalEntitiesActive:       make(UserEntityStat),
		TotalEntitiesLaunching:    make(UserEntityStat),
		TotalEntitiesFailedLaunch: make(UserEntityStat),
		TotalEntitiesFailedActive: make(UserEntityStat),
		TotalEntitiesStopped:      make(UserEntityStat),
		ErrorRate:                 ratecounter.NewRateCounter(interval),
		ActionSendRate:            ratecounter.NewRateCounter(interval),
		ActionRecieveRate:         ratecounter.NewRateCounter(interval),
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
		stats.TotalErrors.Modify(report.Config, 1)
	case STATUS_ACTIVE:
		stats.TotalEntitiesActive.Modify(report.Config, 1)
		stats.TotalEntitiesLaunching.Modify(report.Config, -1)
	case STATUS_LAUNCHING:
		stats.TotalEntitiesLaunching.Modify(report.Config, 1)
	case STATUS_FAILED_LAUNCH:
		stats.TotalEntitiesLaunching.Modify(report.Config, -1)
		stats.TotalEntitiesFailedLaunch.Modify(report.Config, 1)
	case STATUS_FAILED_ACTIVE:
		stats.TotalEntitiesActive.Modify(report.Config, -1)
		stats.TotalEntitiesFailedActive.Modify(report.Config, 1)
	case STATUS_STOPPED:
		stats.TotalEntitiesStopped.Modify(report.Config, 1)
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
			out.Println(stats.TotalEntitiesActive.ToString("Total Entities Active"))
			out.Println(stats.TotalEntitiesLaunching.ToString("Total Entities Launching"))
			out.Println(stats.TotalErrors.ToString("Total Errors"))
			out.Println(stats.TotalEntitiesFailedLaunch.ToString("Total Entities Failed to Launch"))
			out.Println(stats.TotalEntitiesFailedActive.ToString("Total Entities Failed While Active"))
			out.Println(stats.TotalEntitiesStopped.ToString("Total Entities Stopped"))
			out.Println("Send Actions per second: " + statToString(stats.ActionSendRate.Rate()))
			out.Println("")
			out.Println("Recieve Actions per second: " + statToString(stats.ActionRecieveRate.Rate()))
			out.Println("")
			out.Println("Errors per second: " + statToString(stats.ErrorRate.Rate()))
			out.Println("-----------------------------")
		}
	}
}
