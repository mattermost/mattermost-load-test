// Copyright (c) 2017 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information

package loadtest

import (
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"

	"time"

	"github.com/gizak/termui"
	"github.com/mattermost/mattermost-load-test/cmdlog"
	"github.com/mattermost/platform/model"
)

func RunTest(test *TestRun) error {
	defer cmdlog.CloseLog()

	cfg, err := GetConfig()
	if err != nil {
		return fmt.Errorf("Unable to find configuration file: " + err.Error())
	}

	clientTimingStats := NewClientTimingStats()

	strchanbe := cmdlog.GetStringChannelBackend()
	logbuf := NewUIBuffer(10)
	go func() {
		for {
			select {
			case str := <-strchanbe:
				logbuf.Add(str)
			}
		}
	}()

	if cfg.DisplayConfiguration.ShowUI {
		go CreateLoadtestUI(clientTimingStats, logbuf)
	} else if cfg.DisplayConfiguration.LogToConsole {
		cmdlog.SetConsoleLog()
	}

	cmdlog.Info("Setting up server.")
	serverData, err := SetupServer(cfg)
	if err != nil {
		return err
	}

	cmdlog.Info("Logging in as users.")
	tokens := loginAsUsers(cfg)

	// Stop channels and wait groups, to stop and wait verious things
	// For entity monitoring routines
	stopMonitors := make(chan bool)
	var waitMonitors sync.WaitGroup
	// For entities
	stopEntity := make(chan bool)
	var waitEntity sync.WaitGroup

	// Data channels
	// Channel to recieve user entity status reports
	statusChannel := make(chan UserEntityStatusReport, 10000)
	// Channels to recieve timing information from the clients
	clientTimingChannel := make(chan TimedRoundTripperReport, 10000)
	clientTimingChannel3 := make(chan TimedRoundTripperReport, 10000)

	waitMonitors.Add(1)
	go ProcessClientRoundTripReports(clientTimingStats, clientTimingChannel3, clientTimingChannel, stopMonitors, &waitMonitors)

	numEntities := len(tokens)
	entityNum := 0
	for _, usertype := range test.UserEntities {
		numEntitesToCreateForType := int(math.Floor((float64(usertype.Freq) / 100.0) * float64(numEntities)))
		cmdlog.Info("Starting " + strconv.Itoa(numEntitesToCreateForType) + " entities")
		for i := 0; i < numEntitesToCreateForType; i++ {
			cmdlog.Infof("Starting entity %v", entityNum)
			// Get the user auth token for this entity.
			entityToken := tokens[entityNum]

			// Create some clients
			userClient := newClientFromToken(entityToken, cfg.ConnectionConfiguration.ServerURL)
			userClient3 := newV3ClientFromToken(entityToken, cfg.ConnectionConfiguration.ServerURL)
			if cfg.UserEntitiesConfiguration.EnableRequestTiming {
				userClient.HttpClient.Transport = NewTimedRoundTripper(clientTimingChannel)
			}
			if cfg.UserEntitiesConfiguration.EnableRequestTiming {
				userClient3.HttpClient.Transport = NewTimedRoundTripper(clientTimingChannel3)
			}

			// Websocket client
			websocketURL := cfg.ConnectionConfiguration.WebsocketURL
			userWebsocketClient, err := model.NewWebSocketClient(websocketURL, entityToken)
			if err != nil {
				cmdlog.Error("Unable to connect websocket: " + err.Error())
			}

			// How fast to spam the server
			actionRate := time.Duration(float64(cfg.UserEntitiesConfiguration.ActionRateMilliseconds)*usertype.RateMultiplier) * time.Millisecond

			entityConfig := &EntityConfig{
				EntityNumber:        entityNum,
				EntityName:          usertype.Entity.Name,
				EntityActions:       usertype.Entity.Actions,
				UserData:            serverData.BulkloadResult.Users[entityNum],
				ChannelMap:          serverData.ChannelIdMap,
				TeamMap:             serverData.TeamIdMap,
				Client:              userClient,
				Client3:             userClient3,
				WebSocketClient:     userWebsocketClient,
				ActionRate:          actionRate,
				LoadTestConfig:      cfg,
				StatusReportChannel: statusChannel,
				StopChannel:         stopEntity,
				StopWaitGroup:       &waitEntity,
				Info:                make(map[string]interface{}),
			}

			waitEntity.Add(1)
			go runEntity(entityConfig)

			waitEntity.Add(1)
			go websocketListen(entityConfig)

			if cfg.UserEntitiesConfiguration.DoStatusPolling {
				waitEntity.Add(1)
				go doStatusPolling(entityConfig)
			}

			// Spread out the entities to avoid everything happening at once
			time.Sleep(time.Millisecond * time.Duration(rand.Intn(20)))
			entityNum++
		}
	}

	cmdlog.Info("Done starting entities")

	interrupChannel := make(chan os.Signal)
	signal.Notify(interrupChannel, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	cmdlog.Infof("Test set to run for %v minutes", cfg.UserEntitiesConfiguration.TestLengthMinutes)
	timeoutchan := time.After(time.Duration(cfg.UserEntitiesConfiguration.TestLengthMinutes) * time.Minute)

	select {
	case <-interrupChannel:
		cmdlog.Info("Interupted!")
	case <-timeoutchan:
		cmdlog.Info("Test finished normally")
	}
	close(stopEntity)
	termui.StopLoop()

	cmdlog.Info("Waiting for user entities. Timout is 10 seconds.")
	waitWithTimeout(&waitEntity, 10*time.Second)

	cmdlog.Info("Stopping monitor routines. Timeout is 10 seconds.")
	close(stopMonitors)
	waitWithTimeout(&waitMonitors, 10*time.Second)

	report := clientTimingStats.PrintReport()
	cmdlog.Info(report)
	ioutil.WriteFile("results.txt", []byte(report), 0644)

	cmdlog.Info("DONE!")

	return nil
}
