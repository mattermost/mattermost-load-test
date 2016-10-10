// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package main

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/mattermost/mattermost-load-test/cmd/cmdlib"
	"github.com/mattermost/mattermost-load-test/loadtestconfig"
	"github.com/mattermost/platform/model"
	"github.com/spf13/cobra"
)

type UserEntityPosterConfiguration struct {
	PostingFrequencySeconds int
}

func NewUserEntityPosterConfig(config *UserEntityConfig) UserEntityPosterConfiguration {
	var userEntityPosterConfig UserEntityPosterConfiguration
	loadtestconfig.UnmarshalConfigStruct(&userEntityPosterConfig)

	if userEntityPosterConfig.PostingFrequencySeconds == 0 {
		userEntityPosterConfig.PostingFrequencySeconds = 1
	}

	return userEntityPosterConfig
}

func testActiveCmd(cmd *cobra.Command, args []string) {
	context := cmdlib.MakeCommandContext()

	testActive(context)
}

func testActive(c *cmdlib.CommandContext) {
	numEntities := c.LoadTestConfig.UserEntitiesConfiguration.NumClientEntities

	inputState := loadtestconfig.ServerStateFromStdin()

	c.PrettyPrintln("Starting active users load test")

	// Create a channel to signal a stop command
	stopChan := make(chan bool)

	// Create a wait group so we can wait for our entites to complete
	var stopWait sync.WaitGroup

	// Create Channel for users to report status
	statusPrinterStopChan := make(chan bool)

	// Waitgroup for waiting for status messages to finish printing
	var printerWait sync.WaitGroup

	// Channel to recieve user entity status reports
	statusChannel := make(chan UserEntityStatusReport, 1000)

	printerWait.Add(1)
	go UserEntityStatusPrinter(c, statusChannel, statusPrinterStopChan, &printerWait, inputState.Users)

	c.Println("Starting ramp-up")
	for entityNum := 0; entityNum < numEntities; entityNum++ {
		// Get the user for this entity. If there are not enough users
		// for the number of entities requested, wrap around.
		entityUser := &inputState.Users[entityNum%len(inputState.Users)]

		userClient := cmdlib.GetUserClient(&c.LoadTestConfig.ConnectionConfiguration, entityUser)

		userWebsocketClient, err := model.NewWebSocketClient(c.LoadTestConfig.ConnectionConfiguration.WebsocketURL, userClient.AuthToken)
		if err != nil {
			c.PrintErrorln("Unable to setup websocket client", err)
			continue
		}

		config := UserEntityConfig{
			Id:                  entityNum,
			EntityUser:          entityUser,
			Client:              userClient,
			WebSocketClient:     userWebsocketClient,
			LoadTestConfig:      c.LoadTestConfig,
			StatusReportChannel: statusChannel,
			StopEntityChannel:   stopChan,
			StopEntityWaitGroup: &stopWait,
		}
		stopWait.Add(2)
		go startWebsocketListenerUserEntity(config)
		go startPosterUserEntity(config, inputState.Channels, entityUser)
		time.Sleep(time.Duration(c.LoadTestConfig.UserEntitiesConfiguration.EntityRampupDistanceMilliseconds) * time.Millisecond)
	}
	c.Println("Ramp-up complete")
	interrupChannel := make(chan os.Signal)
	signal.Notify(interrupChannel, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-interrupChannel
	c.Println("Shutdown signal recieved.")
	close(stopChan)

	c.Println("Waiting for user entities")
	stopWait.Wait()
	c.Println("Flushing status reporting channel")
	close(statusPrinterStopChan)
	printerWait.Wait()
	c.Println("DONE!")
}

func startWebsocketListenerUserEntity(config UserEntityConfig) {
	config.SendStatusLaunching()
	defer config.StopEntityWaitGroup.Done()

	config.WebSocketClient.Listen()

	websocketRetryCount := 0

	config.SendStatusActive("Listening")
	for {
		select {
		case <-config.StopEntityChannel:
			config.SendStatusStopped("")
			return
		case event, ok := <-config.WebSocketClient.EventChannel:
			if !ok {
				if config.WebSocketClient.ListenError != nil {
					config.SendStatusError(config.WebSocketClient.ListenError, "Websocket error")
				} else {
					config.SendStatusError(nil, "Server closed websocket")
				}

				// If we are set to retry connection, first retry immediately, then backoff until retry max is reached
				if config.LoadTestConfig.ConnectionConfiguration.RetryWebsockets {
					if websocketRetryCount > config.LoadTestConfig.ConnectionConfiguration.MaxRetryWebsocket {
						config.SendStatusFailed(nil, "Websocket disconneced. Max retries reached.")
						return
					}
					time.Sleep(time.Duration(websocketRetryCount) * time.Second)
					config.WebSocketClient.Listen()
					websocketRetryCount++
					continue
				} else {
					config.SendStatusFailed(nil, "Websocket disconneced. No Retry.")
					return
				}
			}
			config.SendStatusActionRecieve("Recieved websocket event: " + event.Event)
		}
	}
}

func startPosterUserEntity(config UserEntityConfig, channels []loadtestconfig.ServerStateChannel, user *loadtestconfig.ServerStateUser) {
	config.SendStatusLaunching()
	defer config.StopEntityWaitGroup.Done()
	posterConfig := NewUserEntityPosterConfig(&config)

	// Allows us to perform our action every x seconds
	postTicker := time.NewTicker(time.Second * time.Duration(posterConfig.PostingFrequencySeconds))
	defer postTicker.Stop()

	var postCount int64 = 0

	config.SendStatusActive("Posting")
	for {
		select {
		case <-config.StopEntityChannel:
			config.SendStatusStopped("")
			return
		case <-postTicker.C:
			channel := channels[user.ChannelsJoined[postCount%int64(len(user.ChannelsJoined))]]
			config.Client.SetTeamId(channel.TeamId)
			post := &model.Post{
				ChannelId: channel.Id,
				Message:   "Test message",
			}
			_, err := config.Client.CreatePost(post)
			if err != nil {
				config.SendStatusError(err, "Failed to post message")
			} else {
				config.SendStatusActionSend("Posted Message")
			}
			postCount++
		}
	}
}
