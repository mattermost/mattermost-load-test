// Copyright (c) 2017 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package loadtest

import (
	"fmt"
	"html/template"
	"regexp"
	"strings"
	"sync"
	"time"

	"bytes"

	"github.com/mattermost/mattermost-load-test/cmdlog"
	"github.com/mattermost/mattermost-server/model"
	"github.com/montanaflynn/stats"
	"github.com/paulbellamy/ratecounter"
)

type RouteStatResults struct {
	Max                float64
	Min                float64
	Mean               float64
	Median             float64
	InterQuartileRange float64
}

type RouteStats struct {
	NumHits            int64
	NumErrors          int64
	Duration           []float64
	DurationLastMinute *ratecounter.AvgRateCounter
	Max                float64
	Min                float64
	Mean               float64
	Median             float64
	InterQuartileRange float64
}

type ClientTimingStats struct {
	Routes     map[string]*RouteStats
	RouteNames []string
}

func NewRouteStats() *RouteStats {
	return &RouteStats{
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

func (s *RouteStats) CalcResults() {
	s.Max, _ = stats.Max(s.Duration)
	s.Min, _ = stats.Min(s.Duration)
	s.Mean, _ = stats.Mean(s.Duration)
	s.Median, _ = stats.Median(s.Duration)
	s.InterQuartileRange, _ = stats.InterQuartileRange(s.Duration)
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
		newroutestats := NewRouteStats()
		newroutestats.AddSample(duration, status)
		ts.Routes[route] = newroutestats
		ts.RouteNames = append(ts.RouteNames, route)
	}
}

var getChannelMembersForUserPathRegex = regexp.MustCompile("/users/[a-z0-9]{26}/teams/[a-z0-9]{26}/channels/members")
var teamPathRegex = regexp.MustCompile("/teams/[a-z0-9]{26}/")
var removeChannelMemberPathRegex = regexp.MustCompile("/channels/[a-z0-9]{26}/members/[a-z0-9]{26}")
var channelPathRegex = regexp.MustCompile("/channels/[a-z0-9]{26}/")
var postPathRegex = regexp.MustCompile("/posts/[a-z0-9]{26}/")
var filePathRegex = regexp.MustCompile("/files/[a-z0-9]{26}/")
var getUserByEmailPathRegex = regexp.MustCompile("/users/email/.*@.*")

func processCommonPaths(path string) string {
	result := strings.TrimPrefix(path, model.API_URL_SUFFIX)
	result = getChannelMembersForUserPathRegex.ReplaceAllString(result, "/users/UID/teams/TID/channels/members/")
	result = teamPathRegex.ReplaceAllString(result, "/teams/TID/")
	result = removeChannelMemberPathRegex.ReplaceAllString(result, "/channels/CID/members/UID/")
	result = channelPathRegex.ReplaceAllString(result, "/channels/CID/")
	result = postPathRegex.ReplaceAllString(result, "/posts/PID/")
	result = filePathRegex.ReplaceAllString(result, "/files/PID/")
	result = getUserByEmailPathRegex.ReplaceAllString(result, "/users/email/EMAIL")
	return result
}

func (ts *ClientTimingStats) AddTimingReport(timingReport TimedRoundTripperReport) {
	path := processCommonPaths(timingReport.Path)
	ts.AddRouteSample(path, int64(timingReport.RequestDuration/time.Millisecond), timingReport.StatusCode)
}

// Score is currently the average mean of all the routes
func (ts *ClientTimingStats) GetScore() float64 {
	total := 0.0
	num := 0.0
	for _, route := range ts.RouteNames {
		stats := ts.Routes[route]
		total += stats.Mean
		total += stats.Median
		total += stats.InterQuartileRange
		num += 1.0
	}

	return total / num
}

func (ts *ClientTimingStats) PrintReport() string {
	const rates = `Total Hits: {{.NumHits}}
Error Rate: {{percent .NumErrors .NumHits}}%
Max Response Time: {{.Max}}ms
Min Response Time: {{.Min}}ms
Mean Response Time: {{printf "%.2f" .Mean}}ms
Median Response Time: {{printf "%.2f" .Median}}ms
Inter Quartile Range: {{.InterQuartileRange}}

`
	for _, route := range ts.Routes {
		route.CalcResults()
	}

	funcMap := template.FuncMap{
		"percent": func(x, y int64) string {
			return fmt.Sprintf("%.2f", float64(x)/float64(y)*100.0)
		},
	}
	rateTemplate := template.Must(template.New("rates").Funcs(funcMap).Parse(rates))

	var buf bytes.Buffer
	fmt.Fprintln(&buf, "")
	fmt.Fprintln(&buf, "--------- Timings Report ------------")

	for _, route := range ts.RouteNames {
		fmt.Fprintln(&buf, "Route: "+route)
		if err := rateTemplate.Execute(&buf, ts.Routes[route]); err != nil {
			cmdlog.Error("Error executing template: " + err.Error())
		}
	}

	fmt.Fprintf(&buf, "Score: %.2f", ts.GetScore())
	fmt.Fprintln(&buf, "")

	return buf.String()
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
