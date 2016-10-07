// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package main

import (
	"fmt"
	"sync"

	"github.com/mattermost/mattermost-load-test/loadtestconfig"
	"github.com/mattermost/platform/model"
)

const (
	STATUS_LAUNCHING      int = iota
	STATUS_ACTIVE         int = iota
	STATUS_STOPPED        int = iota
	STATUS_ERROR          int = iota
	STATUS_FAILED         int = iota
	STATUS_ACTION_SEND    int = iota
	STATUS_ACTION_RECIEVE int = iota
)

type UserEntityStatusReport struct {
	Status       int
	Err          error
	UserEntityId int
	Details      string
}

func statusString(status int) string {
	switch status {
	case STATUS_LAUNCHING:
		return "LAUNCHING"
	case STATUS_ACTIVE:
		return "ACTIVE"
	case STATUS_STOPPED:
		return "STOPPED"
	case STATUS_ERROR:
		return "ERROR"
	case STATUS_FAILED:
		return "FAILED"
	case STATUS_ACTION_SEND:
		return "ACTION_SEND"
	case STATUS_ACTION_RECIEVE:
		return "ACTION_RECIEVE"
	}
	return "SOMTHING BAD"
}

func (report UserEntityStatusReport) String() string {
	if report.Err == nil {
		return fmt.Sprintf("#%v [%v]: %v", report.UserEntityId, statusString(report.Status), report.Details)
	}
	return fmt.Sprintf("#%v [%v]: %v, %v", report.UserEntityId, statusString(report.Status), report.Details, report.Err)
}

type UserEntityConfig struct {
	Id                  int
	Client              *model.Client
	WebSocketClient     *model.WebSocketClient
	LoadTestConfig      *loadtestconfig.LoadTestConfig
	StatusReportChannel chan<- UserEntityStatusReport
	StopEntityChannel   <-chan bool
	StopEntityWaitGroup *sync.WaitGroup
}

func (config *UserEntityConfig) SendStatus(status int, err error, details string) {
	config.StatusReportChannel <- UserEntityStatusReport{
		Status:       status,
		Err:          err,
		UserEntityId: config.Id,
		Details:      details,
	}
}

func (config *UserEntityConfig) SendStatusLaunching() {
	config.SendStatus(STATUS_LAUNCHING, nil, "")
}

func (config *UserEntityConfig) SendStatusActive(details string) {
	config.SendStatus(STATUS_ACTIVE, nil, details)
}

func (config *UserEntityConfig) SendStatusError(err error, details string) {
	config.SendStatus(STATUS_ERROR, err, details)
}

func (config *UserEntityConfig) SendStatusFailed(err error, details string) {
	config.SendStatus(STATUS_FAILED, err, details)
}

func (config *UserEntityConfig) SendStatusActionSend(details string) {
	config.SendStatus(STATUS_ACTION_SEND, nil, details)
}

func (config *UserEntityConfig) SendStatusActionRecieve(details string) {
	config.SendStatus(STATUS_ACTION_RECIEVE, nil, details)
}

func (config *UserEntityConfig) SendStatusStopped(details string) {
	config.SendStatus(STATUS_STOPPED, nil, details)
}
