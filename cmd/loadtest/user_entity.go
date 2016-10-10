// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package main

import (
	"sync"

	"github.com/mattermost/mattermost-load-test/loadtestconfig"
	"github.com/mattermost/platform/model"
)

type UserEntityConfig struct {
	Id                  int
	EntityUser          *loadtestconfig.ServerStateUser
	Client              *model.Client
	WebSocketClient     *model.WebSocketClient
	LoadTestConfig      *loadtestconfig.LoadTestConfig
	StatusReportChannel chan<- UserEntityStatusReport
	StopEntityChannel   <-chan bool
	StopEntityWaitGroup *sync.WaitGroup
}

func (config *UserEntityConfig) SendStatus(status int, err error, details string) {
	config.StatusReportChannel <- UserEntityStatusReport{
		Status:  status,
		Err:     err,
		Config:  config,
		Details: details,
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
