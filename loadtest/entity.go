// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package loadtest

import (
	"math/rand"
	"runtime/debug"
	"sync"
	"time"

	"github.com/mattermost/mattermost-load-test/randutil"
	"github.com/mattermost/mattermost-server/mlog"
	"github.com/mattermost/mattermost-server/model"
)

type EntityConfig struct {
	EntityNumber        int
	EntityName          string
	EntityActions       []randutil.Choice
	UserData            UserImportData
	ChannelMap          map[string]string
	TeamMap             map[string]string
	TownSquareMap       map[string]string
	Client              *model.Client4
	AdminClient         *model.Client4
	WebSocketClient     *model.WebSocketClient
	ActionRate          time.Duration
	LoadTestConfig      *LoadTestConfig
	StatusReportChannel chan<- UserEntityStatusReport
	StopChannel         <-chan bool
	StopWaitGroup       *sync.WaitGroup
	Info                map[string]interface{}
}

func runEntity(ec *EntityConfig) {
	defer func() {
		if r := recover(); r != nil {
			mlog.Error("Recovered", mlog.Any("recover", r), mlog.String("stack", string(debug.Stack())))
			ec.StopWaitGroup.Add(1)
			go runEntity(ec)
		}
	}()
	defer ec.StopWaitGroup.Done()

	actionRateMaxVarianceMilliseconds := ec.LoadTestConfig.UserEntitiesConfiguration.ActionRateMaxVarianceMilliseconds

	// Ensure that the entities act at uniformly distributed times.
	now := time.Now()
	intervalStart := time.Unix(0, now.UnixNano()-now.UnixNano()%int64(ec.ActionRate/time.Nanosecond))
	start := intervalStart.Add(time.Duration(rand.Int63n(int64(ec.ActionRate))))
	if start.Before(now) {
		start = start.Add(ec.ActionRate)
	}
	delay := start.Sub(now)

	timer := time.NewTimer(delay)
	for {
		select {
		case <-ec.StopChannel:
			return
		case <-timer.C:
			action, err := randutil.WeightedChoice(ec.EntityActions)
			if err != nil {
				mlog.Error("Failed to pick weighted choice", mlog.Err(err))
				return
			}
			action.Item.(func(*EntityConfig))(ec)
			halfVarianceDuration := time.Duration(actionRateMaxVarianceMilliseconds / 2.0)
			randomDurationWithinVariance := time.Duration(rand.Intn(actionRateMaxVarianceMilliseconds))
			timer.Reset(ec.ActionRate + randomDurationWithinVariance - halfVarianceDuration)
		}
	}
}

func doStatusPolling(ec *EntityConfig) {
	defer func() {
		if r := recover(); r != nil {
			mlog.Error("Recovered", mlog.Any("recover", r), mlog.String("stack", string(debug.Stack())))
			ec.StopWaitGroup.Add(1)
			go doStatusPolling(ec)
		}
	}()
	defer ec.StopWaitGroup.Done()

	ticker := time.NewTicker(60 * time.Second)
	for {
		select {
		case <-ec.StopChannel:
			return
		case <-ticker.C:
			actionGetStatuses(ec)
		}
	}
}

func websocketListen(ec *EntityConfig) {
	defer ec.StopWaitGroup.Done()

	if ec.WebSocketClient == nil {
		return
	}

	ec.WebSocketClient.Listen()

	websocketRetryCount := 0

	for {
		select {
		case <-ec.StopChannel:
			return
		case _, ok := <-ec.WebSocketClient.EventChannel:
			if !ok {
				// If we are set to retry connection, first retry immediately, then backoff until retry max is reached
				for {
					if websocketRetryCount > 5 {
						if ec.WebSocketClient.ListenError != nil {
							mlog.Error("Websocket Error", mlog.Err(ec.WebSocketClient.ListenError))
						} else {
							mlog.Error("Server closed websocket")
						}
						mlog.Error("Websocket disconneced. Max retries reached.")
						return
					}
					time.Sleep(time.Duration(websocketRetryCount) * time.Second)
					if err := ec.WebSocketClient.Connect(); err != nil {
						websocketRetryCount++
						continue
					}
					ec.WebSocketClient.Listen()
					break
				}
			}
		}
	}
}

func (config *EntityConfig) SendStatus(status int, err error, details string) {
	config.StatusReportChannel <- UserEntityStatusReport{
		Status:  status,
		Err:     err,
		Config:  config,
		Details: details,
	}
}

func (config *EntityConfig) SendStatusLaunching() {
	config.SendStatus(STATUS_LAUNCHING, nil, "")
}

func (config *EntityConfig) SendStatusActive(details string) {
	config.SendStatus(STATUS_ACTIVE, nil, details)
}

func (config *EntityConfig) SendStatusError(err error, details string) {
	config.SendStatus(STATUS_ERROR, err, details)
}

func (config *EntityConfig) SendStatusFailedLaunch(err error, details string) {
	config.SendStatus(STATUS_FAILED_LAUNCH, err, details)
}

func (config *EntityConfig) SendStatusFailedActive(err error, details string) {
	config.SendStatus(STATUS_FAILED_ACTIVE, err, details)
}

func (config *EntityConfig) SendStatusActionSend(details string) {
	config.SendStatus(STATUS_ACTION_SEND, nil, details)
}

func (config *EntityConfig) SendStatusActionRecieve(details string) {
	config.SendStatus(STATUS_ACTION_RECIEVE, nil, details)
}

func (config *EntityConfig) SendStatusStopped(details string) {
	config.SendStatus(STATUS_STOPPED, nil, details)
}
