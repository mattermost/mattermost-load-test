// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package main

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/mattermost/mattermost-load-test/cmd/cmdlib"
	"github.com/mattermost/mattermost-load-test/loadtestconfig"
	"github.com/mattermost/platform/model"
)

type UserEntityLogger interface {
	Println(a ...interface{}) (int, error)
}

type TmpLogger struct {
	Writer io.Writer
}

func (logger *TmpLogger) Println(a ...interface{}) (int, error) {
	return fmt.Fprintln(logger.Writer, a...)
}

func waitWithTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan bool)
	go func() {
		wg.Wait()
		close(c)
	}()
	select {
	// Everything is OK
	case <-c:
		return true
	// We timed out
	case <-time.After(timeout):
		return false
	}
}

func StartUserEntities(config *loadtestconfig.LoadTestConfig, serverState *loadtestconfig.ServerState, entityCreationFunctions ...UserEntityCreator) {
	// Create a channel to signal a stop command
	stopChan := make(chan bool)

	// Create a wait group so we can wait for our entites to complete
	var stopWait sync.WaitGroup

	// Create Channel for users to report status
	statusPrinterStopChan := make(chan bool)

	// Waitgroup for waiting for status messages to finish printing
	var printerWait sync.WaitGroup

	// Channel to recieve user entity status reports
	statusChannel := make(chan UserEntityStatusReport, 10000)

	// Wait group so all entities start staggered properly
	var entityWaitChannel sync.WaitGroup
	entityWaitChannel.Add(1)

	// Create writer
	out := &TmpLogger{
		Writer: os.Stdout,
	}

	// Channel to recieve timing information from the client
	clientTimingChannel := make(chan TimedRoundTripperReport, 10000)

	printerWait.Add(1)
	go UserEntityStatusPrinter(out, statusChannel, clientTimingChannel, statusPrinterStopChan, &printerWait, serverState.Users)

	numEntities := config.UserEntitiesConfiguration.LastEntityNumber
	entityOffset := config.UserEntitiesConfiguration.FirstEntityNumber

	out.Println("------------------------- Starting " + strconv.Itoa(numEntities) + " entities")
	for entityNum := entityOffset; entityNum < numEntities; entityNum++ {
		out.Println("Starting Entity: " + strconv.Itoa(entityNum))
		// Get the user for this entity. If there are not enough users
		// for the number of entities requested, wrap around.
		entityUser := &serverState.Users[entityNum%len(serverState.Users)]

		userClient := cmdlib.GetUserClient(&config.ConnectionConfiguration, entityUser)
		if config.ConnectionConfiguration.EnableRequestTiming {
			userClient.HttpClient.Transport = NewTimedRoundTripper(clientTimingChannel)
		}

		websocketURL := config.ConnectionConfiguration.WebsocketURL
		userWebsocketClient := &model.WebSocketClient{
			websocketURL,
			websocketURL + model.API_URL_SUFFIX_V3,
			nil,
			userClient.AuthToken,
			1,
			make(chan *model.WebSocketEvent, 100),
			make(chan *model.WebSocketResponse, 100),
			nil,
		}

		entityConfig := UserEntityConfig{
			Id:                  entityNum,
			SubEntityName:       "TBD", //Filled in for each sub-entity below
			EntityUser:          entityUser,
			Client:              userClient,
			WebSocketClient:     userWebsocketClient,
			LoadTestConfig:      config,
			State:               serverState,
			StatusReportChannel: statusChannel,
			StopEntityChannel:   stopChan,
			StopEntityWaitGroup: &stopWait,
		}
		stopWait.Add(len(entityCreationFunctions))
		for _, createEntity := range entityCreationFunctions {
			entityConfig.SubEntityName = runtime.FuncForPC(reflect.ValueOf(createEntity).Pointer()).Name()[8:]
			entity := createEntity(entityConfig)
			go func(entityNum int) {
				afterChan := time.After(time.Duration(config.UserEntitiesConfiguration.EntityRampupDistanceMilliseconds) * time.Millisecond * time.Duration(entityNum))
				select {
				case <-stopChan:
					return
				case <-afterChan:
					entityWaitChannel.Wait()
					entity.Start()
				}
			}(entityNum)
		}
	}
	// Release the entities
	entityWaitChannel.Done()

	out.Println("------------------------- Done starting entities")
	interrupChannel := make(chan os.Signal)
	signal.Notify(interrupChannel, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-interrupChannel
	out.Println("Shutdown signal recieved.")
	close(stopChan)

	out.Println("Waiting for user entities, timout is 5 seconds")
	waitWithTimeout(&stopWait, 5*time.Second)

	out.Println("Flushing status reporting channel")
	close(statusPrinterStopChan)
	printerWait.Wait()
	out.Println("DONE!")
}
