// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package main

import (
	"time"

	"github.com/mattermost/platform/model"
)

type UserEntityWebsocketListener struct {
	UserEntityConfig
}

func NewUserEntityWebsocketListener(cfg UserEntityConfig) UserEntity {
	return &UserEntityWebsocketListener{
		UserEntityConfig: cfg,
	}
}

func (me *UserEntityWebsocketListener) Start() {
	me.SendStatusLaunching()
	defer me.StopEntityWaitGroup.Done()

	if err := me.WebSocketClient.Connect(); err != nil {
		me.SendStatusFailedLaunch(err, "Failed to connect websocket")
		return
	}

	me.WebSocketClient.Listen()

	websocketRetryCount := 0

	me.SendStatusActive("Listening")
	for {
		select {
		case <-me.StopEntityChannel:
			me.SendStatusStopped("")
			return
		case event, ok := <-me.WebSocketClient.EventChannel:
			if !ok {
				if me.WebSocketClient.ListenError != nil {
					me.SendStatusError(me.WebSocketClient.ListenError, "Websocket error")
				} else {
					me.SendStatusError(nil, "Server closed websocket")
				}

				// If we are set to retry connection, first retry immediately, then backoff until retry max is reached
				if me.LoadTestConfig.ConnectionConfiguration.RetryWebsockets {
					for {
						if websocketRetryCount > me.LoadTestConfig.ConnectionConfiguration.MaxRetryWebsocket {
							me.SendStatusFailedActive(nil, "Websocket disconneced. Max retries reached.")
							return
						}
						time.Sleep(time.Duration(websocketRetryCount) * time.Second)
						me.WebSocketClient.EventChannel = make(chan *model.WebSocketEvent, 100)
						me.WebSocketClient.ResponseChannel = make(chan *model.WebSocketResponse, 100)
						if err := me.WebSocketClient.Connect(); err != nil {
							websocketRetryCount++
							continue
						}
						me.WebSocketClient.Listen()
						break
					}
				} else {
					me.SendStatusFailedActive(nil, "Websocket disconneced. No Retry.")
					return
				}
			}
			if event != nil {
				me.SendStatusActionRecieve("Recieved websocket event: " + event.Event)
			}
		}
	}
}
