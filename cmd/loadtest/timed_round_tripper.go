// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package main

import (
	"net/http"
	"time"
)

type TimedRoundTripperReport struct {
	Path            string
	RequestDuration time.Duration
}

type TimedRoundTripper struct {
	standardRoundTripper http.RoundTripper
	reportChan           chan<- TimedRoundTripperReport
}

func NewTimedRoundTripper(reportChan chan<- TimedRoundTripperReport) *TimedRoundTripper {
	rt := &TimedRoundTripper{
		standardRoundTripper: http.DefaultTransport,
		reportChan:           reportChan,
	}

	return rt
}

func (trt *TimedRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	requestStart := time.Now()
	resp, err := trt.standardRoundTripper.RoundTrip(r)
	requestEnd := time.Now()

	trt.reportChan <- TimedRoundTripperReport{
		Path:            r.URL.Path,
		RequestDuration: requestEnd.Sub(requestStart),
	}

	return resp, err
}
