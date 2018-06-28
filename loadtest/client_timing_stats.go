// Copyright (c) 2017 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package loadtest

import (
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/mattermost/mattermost-server/model"
	"github.com/montanaflynn/stats"
	"github.com/paulbellamy/ratecounter"
)

type RouteStatResults struct {
	Max                float64
	Min                float64
	Mean               float64
	Median             float64
	Percentile90       float64
	Percentile95       float64
	InterQuartileRange float64
}

type RouteStats struct {
	Name               string
	NumHits            int64
	NumErrors          int64
	ErrorRate          float64
	Duration           []float64
	DurationLastMinute *ratecounter.AvgRateCounter `json:"-"`
	Max                float64
	Min                float64
	Mean               float64
	Median             float64
	Percentile90       float64
	Percentile95       float64
	InterQuartileRange float64
}

type ClientTimingStats struct {
	Routes map[string]*RouteStats
}

func NewRouteStats(name string) *RouteStats {
	return &RouteStats{
		Name:               name,
		NumErrors:          0,
		Duration:           make([]float64, 0, 100000),
		DurationLastMinute: ratecounter.NewAvgRateCounter(time.Minute),
	}
}

func (s *RouteStats) AddSample(duration int64, status int) {
	s.NumHits += 1
	// Don't count non-ok status in statistics
	if status >= 200 && status < 300 {
		s.Duration = append(s.Duration, float64(duration))
	} else {
		s.NumErrors += 1
	}
}

func (s *RouteStats) Merge(other *RouteStats) *RouteStats {
	newRouteStats := &RouteStats{}
	if s != nil {
		newRouteStats.Name = s.Name
		newRouteStats.NumHits = newRouteStats.NumHits + s.NumHits
		newRouteStats.NumErrors = newRouteStats.NumErrors + s.NumErrors
		newRouteStats.Duration = append(newRouteStats.Duration, s.Duration...)
	}
	if other != nil {
		newRouteStats.Name = other.Name
		newRouteStats.NumHits = newRouteStats.NumHits + other.NumHits
		newRouteStats.NumErrors = newRouteStats.NumErrors + other.NumErrors
		newRouteStats.Duration = append(newRouteStats.Duration, other.Duration...)
	}

	newRouteStats.CalcResults()

	return newRouteStats
}

func (s *RouteStats) CalcResults() {
	if s.NumHits > 0 {
		s.ErrorRate = float64(s.NumErrors) / float64(s.NumHits)
	} else {
		s.ErrorRate = 0
	}
	if len(s.Duration) > 0 {
		s.Max, _ = stats.Max(s.Duration)
		s.Min, _ = stats.Min(s.Duration)
		s.Mean, _ = stats.Mean(s.Duration)
		s.Median, _ = stats.Median(s.Duration)
		s.Percentile90, _ = stats.Percentile(s.Duration, 90)
		s.Percentile95, _ = stats.Percentile(s.Duration, 95)
		s.InterQuartileRange, _ = stats.InterQuartileRange(s.Duration)
	}
}

func NewClientTimingStats() *ClientTimingStats {
	return &ClientTimingStats{
		Routes: make(map[string]*RouteStats),
	}
}

func (ts *ClientTimingStats) AddRouteSample(route string, duration int64, status int) {
	if routestats, ok := ts.Routes[route]; ok {
		routestats.AddSample(duration, status)
	} else {
		newroutestats := NewRouteStats(route)
		newroutestats.AddSample(duration, status)
		ts.Routes[route] = newroutestats
	}
}

func (ts *ClientTimingStats) Merge(timings *ClientTimingStats) *ClientTimingStats {
	newStats := NewClientTimingStats()

	if ts != nil {
		for routeName, route := range ts.Routes {
			newStats.Routes[routeName] = newStats.Routes[routeName].Merge(route)
		}
	}
	if timings != nil {
		for routeName, route := range timings.Routes {
			newStats.Routes[routeName] = newStats.Routes[routeName].Merge(route)
		}
	}

	return newStats
}

var teamPathRegex *regexp.Regexp = regexp.MustCompile("/teams/[a-z0-9]{26}/")
var channelPathRegex *regexp.Regexp = regexp.MustCompile("/channels/[a-z0-9]{26}/")
var postPathRegex *regexp.Regexp = regexp.MustCompile("/posts/[a-z0-9]{26}/")
var filePathRegex *regexp.Regexp = regexp.MustCompile("/files/[a-z0-9]{26}/")
var userPathRegex *regexp.Regexp = regexp.MustCompile("/users/[a-z0-9]{26}/")
var userEmailPathRegex *regexp.Regexp = regexp.MustCompile("/users/email/[^/]+")
var teamMembersForUserPathRegex *regexp.Regexp = regexp.MustCompile("/teams/[a-z0-9]{26}/members/[a-z0-9]{26}")

func processCommonPaths(path string) string {
	result := strings.TrimPrefix(path, model.API_URL_SUFFIX)
	result = teamMembersForUserPathRegex.ReplaceAllString(result, "/teams/TID/members/UID")
	result = teamPathRegex.ReplaceAllString(result, "/teams/TID/")
	result = channelPathRegex.ReplaceAllString(result, "/channels/CID/")
	result = postPathRegex.ReplaceAllString(result, "/posts/PID/")
	result = filePathRegex.ReplaceAllString(result, "/files/PID/")
	result = userPathRegex.ReplaceAllString(result, "/users/UID/")
	result = userEmailPathRegex.ReplaceAllString(result, "/users/email/UID")
	return result
}

func (ts *ClientTimingStats) AddTimingReport(timingReport TimedRoundTripperReport) {
	path := processCommonPaths(timingReport.Path)
	ts.AddRouteSample(path, int64(timingReport.RequestDuration/time.Millisecond), timingReport.StatusCode)
}

// Score is the average of the 95th percentile, median and interquartile range of all routes.
func (ts *ClientTimingStats) GetScore() float64 {
	total := 0.0
	num := 0.0
	for _, stats := range ts.Routes {
		total += stats.Percentile95
		total += stats.Median
		total += stats.InterQuartileRange
		num += 1.0
	}

	return total / num
}

func (ts *ClientTimingStats) CalcResults() {
	for _, route := range ts.Routes {
		route.CalcResults()
	}
}

func ProcessClientRoundTripReports(stats *ClientTimingStats, v3chan <-chan TimedRoundTripperReport, v4chan <-chan TimedRoundTripperReport, stopChan <-chan bool, stopWait *sync.WaitGroup) {
	defer stopWait.Done()

	// This strange thing makes sure that the statusChan is drained before it will listen to the stopChan
	for {
		select {
		case timingReport := <-v3chan:
			stats.AddTimingReport(timingReport)
		case timingReport := <-v4chan:
			stats.AddTimingReport(timingReport)
		default:
			select {
			case timingReport := <-v3chan:
				stats.AddTimingReport(timingReport)
			case timingReport := <-v4chan:
				stats.AddTimingReport(timingReport)
			case <-stopChan:
				return
			}
		}
	}
}
