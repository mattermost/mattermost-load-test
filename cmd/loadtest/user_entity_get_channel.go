// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package main

import (
	"time"

	"github.com/mattermost/mattermost-load-test/loadtestconfig"
)

type UserEntityGetChannel struct {
	UserEntityConfig
	Config UserEntityGetChannelConfiguration
}

type UserEntityGetChannelConfiguration struct {
	GetFrequencySeconds int
}

func NewUserEntityGetChannelConfig() UserEntityGetChannelConfiguration {
	var userEntityGetChannelConfig UserEntityGetChannelConfiguration
	loadtestconfig.UnmarshalConfigSubStruct(&userEntityGetChannelConfig)

	if userEntityGetChannelConfig.GetFrequencySeconds == 0 {
		userEntityGetChannelConfig.GetFrequencySeconds = 10
	}

	return userEntityGetChannelConfig
}

func NewUserEntityGetChannel(cfg UserEntityConfig) UserEntity {
	return &UserEntityGetChannel{
		UserEntityConfig: cfg,
		Config:           NewUserEntityGetChannelConfig(),
	}
}

func (me *UserEntityGetChannel) Start() {
	me.SendStatusLaunching()
	defer me.StopEntityWaitGroup.Done()

	getTicker := time.NewTicker(time.Second * time.Duration(me.Config.GetFrequencySeconds))
	defer getTicker.Stop()

	var getChannelCount int64 = 0

	me.SendStatusActive("Getting channels")
	for {
		select {
		case <-me.StopEntityChannel:
			me.SendStatusStopped("")
			return
		case <-getTicker.C:
			channel := me.GetChannelBasedOnActionCount(getChannelCount)
			me.Client.SetTeamId(channel.TeamId)
			if _, err := me.Client.GetChannel(channel.Id, ""); err != nil {
				me.SendStatusError(err, "Failed to get channel: "+channel.Id)
			} else {
				me.SendStatusActionSend("Got channel")
			}

			if _, err := me.Client.GetChannels(""); err != nil {
				me.SendStatusError(err, "Failed to get channels")
			} else {
				me.SendStatusActionSend("Got Channels")
			}

			if _, err := me.Client.GetPosts(channel.Id, 0, 60, ""); err != nil {
				me.SendStatusError(err, "Failed to get posts: "+channel.Id)
			} else {
				me.SendStatusActionSend("Got Posts")
			}

			if _, err := me.Client.GetChannelStats(channel.Id, ""); err != nil {
				me.SendStatusError(err, "Failed to get channel stats: "+channel.Id)
			} else {
				me.SendStatusActionSend("Got channel stats")
			}

			if _, err := me.Client.UpdateLastViewedAt(channel.Id, true); err != nil {
				me.SendStatusError(err, "Failed to update last viewed at: "+channel.Id)
			} else {
				me.SendStatusActionSend("Updated last viewed at")
			}
		}
	}
}
