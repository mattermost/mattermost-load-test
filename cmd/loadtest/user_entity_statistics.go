// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package main

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/VividCortex/ewma"
	"github.com/mattermost/platform/model"
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

type UserEntityRates struct {
	RateDescription   string
	ErrorRate         *ratecounter.RateCounter
	ActionSendRate    *ratecounter.RateCounter
	ActionRecieveRate *ratecounter.RateCounter
}

func NewUserEntityRates(interval time.Duration, intervalDescription string) UserEntityRates {
	return UserEntityRates{
		RateDescription:   intervalDescription,
		ErrorRate:         ratecounter.NewRateCounter(interval),
		ActionSendRate:    ratecounter.NewRateCounter(interval),
		ActionRecieveRate: ratecounter.NewRateCounter(interval),
	}
}

func (rates *UserEntityRates) String() string {
	out := ""
	out += "Send actions " + rates.RateDescription + " " + rates.ActionSendRate.String() + "\n\n"
	out += "Recieve actions " + rates.RateDescription + " " + rates.ActionRecieveRate.String() + "\n\n"
	out += "Errors " + rates.RateDescription + " " + rates.ErrorRate.String() + "\n"
	return out
}

type UserEntityStatistics struct {
	StartTime time.Time

	TotalErrors               UserEntityStat
	TotalEntitiesActive       UserEntityStat
	TotalEntitiesLaunching    UserEntityStat
	TotalEntitiesFailedLaunch UserEntityStat
	TotalEntitiesFailedActive UserEntityStat
	TotalEntitiesStopped      UserEntityStat

	UserEntityRatesPerSecond UserEntityRates
	UserEntityRatesPerMinute UserEntityRates
	UserEntityRatesPerHour   UserEntityRates

	RouteTimings map[string]ewma.MovingAverage
	Routes       []string
}

func NewUserEntityStatistics() *UserEntityStatistics {
	return &UserEntityStatistics{
		StartTime:                 time.Now(),
		TotalErrors:               make(UserEntityStat),
		TotalEntitiesActive:       make(UserEntityStat),
		TotalEntitiesLaunching:    make(UserEntityStat),
		TotalEntitiesFailedLaunch: make(UserEntityStat),
		TotalEntitiesFailedActive: make(UserEntityStat),
		TotalEntitiesStopped:      make(UserEntityStat),
		UserEntityRatesPerSecond:  NewUserEntityRates(time.Second, "per second"),
		UserEntityRatesPerMinute:  NewUserEntityRates(time.Minute, "per minute"),
		UserEntityRatesPerHour:    NewUserEntityRates(time.Hour, "per hour"),
		RouteTimings:              make(map[string]ewma.MovingAverage),
	}
}

func (stats *UserEntityStatistics) updateTotals(report UserEntityStatusReport) {
	switch report.Status {
	case STATUS_ERROR:
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

func (rates *UserEntityRates) updateRates(status int) {
	switch status {
	case STATUS_ACTION_SEND:
		rates.ActionSendRate.Incr(1)
	case STATUS_ACTION_RECIEVE:
		rates.ActionRecieveRate.Incr(1)
	case STATUS_ERROR:
		rates.ErrorRate.Incr(1)
	}
}

func (stats *UserEntityStatistics) updateEntityStatistics(report UserEntityStatusReport) {
	stats.updateTotals(report)
	stats.UserEntityRatesPerSecond.updateRates(report.Status)
	stats.UserEntityRatesPerMinute.updateRates(report.Status)
	stats.UserEntityRatesPerHour.updateRates(report.Status)
}

var teamPathRegex *regexp.Regexp = regexp.MustCompile("/teams/[a-z0-9]{26}/")
var channelPathRegex *regexp.Regexp = regexp.MustCompile("/channels/[a-z0-9]{26}/")

func processCommonPaths(path string) string {
	result := strings.TrimPrefix(path, model.API_URL_SUFFIX)
	result = teamPathRegex.ReplaceAllString(result, "/teams/TID/")
	result = channelPathRegex.ReplaceAllString(result, "/channels/CID/")
	return result
}

func (stats *UserEntityStatistics) updateClientTimingStats(timingReport TimedRoundTripperReport) {
	path := processCommonPaths(timingReport.Path)
	if timing, ok := stats.RouteTimings[path]; ok {
		timing.Add(float64(timingReport.RequestDuration.Nanoseconds()))
	} else {
		newTiming := ewma.NewMovingAverage()
		newTiming.Add(float64(timingReport.RequestDuration.Nanoseconds()))
		stats.RouteTimings[path] = newTiming
		stats.Routes = append(stats.Routes, path)
		sort.Strings(stats.Routes)
	}
}

func statToString(stat int64) string {
	return strconv.Itoa(int(stat))
}

func durationToString(d time.Duration) string {
	conv := func(f float64) string {
		return fmt.Sprintf("%02d", int(math.Floor(f))%60)
	}
	return conv(d.Hours()) + ":" + conv(d.Minutes()) + ":" + conv(d.Seconds())
}

func printTimingsStats(stats *UserEntityStatistics) string {
	conv := func(val float64) string {
		return fmt.Sprintf("%d", int64(math.Floor(val/1000000.0)))
	}
	out := ""
	for _, route := range stats.Routes {
		out += route + " : " + conv(stats.RouteTimings[route].Value()) + "ms\n"
	}
	return out
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
			timeSinceStart := time.Now().Sub(stats.StartTime)
			out.Println("------- Entity Stats --------")
			out.Println("Elapsed Time:  " + durationToString(timeSinceStart))
			out.Println(stats.TotalEntitiesActive.ToString("Total Entities Active"))
			out.Println(stats.TotalEntitiesLaunching.ToString("Total Entities Launching"))
			out.Println(stats.TotalErrors.ToString("Total Errors"))
			out.Println(stats.TotalEntitiesFailedLaunch.ToString("Total Entities Failed to Launch"))
			out.Println(stats.TotalEntitiesFailedActive.ToString("Total Entities Failed While Active"))
			out.Println(stats.TotalEntitiesStopped.ToString("Total Entities Stopped"))
			out.Println("----------- Rates -----------")
			out.Println("-------- Per Second ---------")
			out.Println(stats.UserEntityRatesPerSecond.String())
			out.Println("-------- Per Minute ---------")
			out.Println(stats.UserEntityRatesPerMinute.String())
			out.Println("-------- Per Hour -----------")
			out.Println(stats.UserEntityRatesPerHour.String())
			out.Println("-----------------------------")
			if len(stats.RouteTimings) > 0 {
				out.Println("")
				out.Println("---------- Timings ----------")
				out.Println(printTimingsStats(stats))
			}
		}
	}
}
