// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package loadtest

import (
	"time"

	"github.com/VividCortex/ewma"
	"github.com/paulbellamy/ratecounter"
)

type UserEntityStat map[string]int64

/*func (stat UserEntityStat) Modify(entity *EntityConfig, ammount int64) {
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
}*/

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

/*func (stats *UserEntityStatistics) updateTotals(report UserEntityStatusReport) {
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
*/

/*
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

func doPrintStats(stats *UserEntityStatistics, stopChan <-chan bool) {
	// Print statistics on timer
	statsTicker := time.NewTicker(time.Second * 3)
	defer statsTicker.Stop()

	for {
		select {
		case <-stopChan:
			return
		case <-statsTicker.C:
			timeSinceStart := time.Now().Sub(stats.StartTime)
			cmdlog.Println("------- Entity Stats --------")
			cmdlog.Println("Elapsed Time:  " + durationToString(timeSinceStart))
			cmdlog.Println(stats.TotalEntitiesActive.ToString("Total Entities Active"))
			cmdlog.Println(stats.TotalEntitiesLaunching.ToString("Total Entities Launching"))
			cmdlog.Println(stats.TotalErrors.ToString("Total Errors"))
			cmdlog.Println(stats.TotalEntitiesFailedLaunch.ToString("Total Entities Failed to Launch"))
			cmdlog.Println(stats.TotalEntitiesFailedActive.ToString("Total Entities Failed While Active"))
			cmdlog.Println(stats.TotalEntitiesStopped.ToString("Total Entities Stopped"))
			cmdlog.Println("----------- Rates -----------")
			cmdlog.Println("-------- Per Second ---------")
			cmdlog.Println(stats.UserEntityRatesPerSecond.String())
			cmdlog.Println("-------- Per Minute ---------")
			cmdlog.Println(stats.UserEntityRatesPerMinute.String())
			cmdlog.Println("-------- Per Hour -----------")
			cmdlog.Println(stats.UserEntityRatesPerHour.String())
			cmdlog.Println("-----------------------------")
			if len(stats.RouteTimings) > 0 {
				cmdlog.Println("")
				cmdlog.Println("---------- Timings ----------")
				cmdlog.Println(printTimingsStats(stats))
			}
		}
	}
}
*/
