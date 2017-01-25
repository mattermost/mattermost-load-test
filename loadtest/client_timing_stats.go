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
	"github.com/mattermost/platform/model"
	"github.com/paulbellamy/ratecounter"
)

type RouteStats struct {
	NumHits            int64
	NumErrors          int64
	MaxDuration        int64
	MinDuration        int64
	Mean               float64
	Variance           float64
	DurationLastMinute *ratecounter.AvgRateCounter
}

type ClientTimingStats struct {
	Routes     map[string]*RouteStats
	RouteNames []string
}

func NewRouteStats() *RouteStats {
	return &RouteStats{
		NumHits:            0,
		MaxDuration:        0,
		MinDuration:        -1,
		Mean:               0.0,
		Variance:           0.0,
		DurationLastMinute: ratecounter.NewAvgRateCounter(time.Minute),
	}
}

func (s *RouteStats) AddSample(duration int64, status int) {
	s.NumHits += 1
	// Don't count non-ok status in statistics
	if status >= 200 && status < 300 {
		delta := float64(duration) - s.Mean
		s.Mean += delta / float64(s.NumHits)
		delta2 := float64(duration) - s.Mean
		s.Variance += delta * delta2

		if s.MinDuration == -1 || s.MinDuration > duration {
			s.MinDuration = duration
		}

		if s.MaxDuration < duration {
			s.MaxDuration = duration
		}

		s.DurationLastMinute.Incr(duration)
	} else {
		s.NumErrors += 1
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
		newroutestats := NewRouteStats()
		newroutestats.AddSample(duration, status)
		ts.Routes[route] = newroutestats
		ts.RouteNames = append(ts.RouteNames, route)
	}
}

var teamPathRegex *regexp.Regexp = regexp.MustCompile("/teams/[a-z0-9]{26}/")
var channelPathRegex *regexp.Regexp = regexp.MustCompile("/channels/[a-z0-9]{26}/")
var postPathRegex *regexp.Regexp = regexp.MustCompile("/posts/[a-z0-9]{26}/")
var filePathRegex *regexp.Regexp = regexp.MustCompile("/files/[a-z0-9]{26}/")

func processCommonPaths(path string) string {
	result := strings.TrimPrefix(path, model.API_URL_SUFFIX)
	result = teamPathRegex.ReplaceAllString(result, "/teams/TID/")
	result = channelPathRegex.ReplaceAllString(result, "/channels/CID/")
	result = postPathRegex.ReplaceAllString(result, "/posts/PID/")
	result = filePathRegex.ReplaceAllString(result, "/files/PID/")
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
		num += 1.0
	}

	return total / num
}

func (ts *ClientTimingStats) PrintReport() string {
	const rates = `Total Hits: {{.NumHits}}
Error Rate: {{percent .NumErrors .NumHits}}%
Max Response Time: {{.MaxDuration}}ms
Min Response Time: {{.MinDuration}}ms
Mean Response Time: {{printf "%.2f" .Mean}}ms
Variance of Response Time: {{variance .Variance .NumHits}}

`

	funcMap := template.FuncMap{
		"percent": func(x, y int64) string {
			return fmt.Sprintf("%.2f", float64(x)/float64(y)*100.0)
		},
		"variance": func(m2 float64, num int64) string {
			return fmt.Sprintf("%.2f", m2/float64(num-1))
		},
	}
	rateTemplate := template.Must(template.New("rates").Funcs(funcMap).Parse(rates))

	var buf bytes.Buffer
	fmt.Fprintln(&buf, "")
	fmt.Fprintln(&buf, "--------- Loadtest Report ------------")

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
