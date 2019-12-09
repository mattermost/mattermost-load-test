// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package loadtest

import (
	"fmt"
	"math/rand"
	"runtime/debug"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-load-test/randutil"
	"github.com/mattermost/mattermost-server/v5/mlog"
	"github.com/mattermost/mattermost-server/v5/model"
)

type EntityConfig struct {
	EntityNumber        int
	EntityName          string
	EntityActions       []randutil.Choice
	UserData            UserImportData
	Users               []UserImportData
	ChannelMap          map[string]map[string]string
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

	r *rand.Rand
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
			action, err := randutil.WeightedChoice(ec.r, ec.EntityActions)
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

	ticker := time.NewTicker(45 * time.Second)
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
	actionWakeup(ec)

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
					actionWakeup(ec)
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

// channelMapLock makes it safe to update the ChannelMap below when multiple entities are in play.
// This won't be necessary if we just enumerate all the private channels like we do public ones.
var channelMapLock sync.Mutex

func (config *EntityConfig) GetTeamChannelId(teamName, channelName string) (string, error) {
	channelId := config.ChannelMap[teamName][channelName]
	if channelId != "" {
		return channelId, nil
	}

	teamId := config.TeamMap[teamName]
	if teamId == "" {
		return "", fmt.Errorf("unable to find team %s", teamName)
	}

	// Private channels won't have been fetched, so try to look it up on demand instead.
	// Ideally, we expose a way to list all the channels on a team, not just the public ones.
	channel, resp := config.AdminClient.GetChannelByName(channelName, teamId, "")
	if resp.Error != nil {
		return "", errors.Wrapf(resp.Error, "failed to get channel %s by name for team %s", channelName, teamId)
	}

	channelMapLock.Lock()
	defer channelMapLock.Unlock()
	config.ChannelMap[teamName][channelName] = channel.Id

	return channel.Id, nil
}
