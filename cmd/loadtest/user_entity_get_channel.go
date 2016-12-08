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
	UseEtags            bool
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

	getChannelEtag := ""
	getPostsEtag := ""
	getChannelStatsEtag := ""

	me.SendStatusActive("Getting channels")
	for {
		select {
		case <-me.StopEntityChannel:
			me.SendStatusStopped("")
			return
		case <-getTicker.C:
			channel := me.GetChannelBasedOnActionCount(getChannelCount)
			me.Client.SetTeamId(channel.TeamId)
			if result, err := me.Client.GetChannel(channel.Id, getChannelEtag); err != nil {
				me.SendStatusError(err, "Failed to get channel: "+channel.Id)
			} else {
				if me.Config.UseEtags && result.Etag != "" {
					getChannelEtag = result.Etag
				}
				me.SendStatusActionSend("Got channel. Etag: " + getChannelEtag)
			}

			if result, err := me.Client.GetPosts(channel.Id, 0, 60, getPostsEtag); err != nil {
				me.SendStatusError(err, "Failed to get posts: "+channel.Id)
			} else {
				if me.Config.UseEtags && result.Etag != "" {
					getPostsEtag = result.Etag
				}
				me.SendStatusActionSend("Got Posts. Etag: " + getPostsEtag)
			}

			if result, err := me.Client.GetChannelStats(channel.Id, getChannelStatsEtag); err != nil {
				me.SendStatusError(err, "Failed to get channel stats: "+channel.Id)
			} else {
				if me.Config.UseEtags {
					getChannelStatsEtag = result.Etag
				}
				me.SendStatusActionSend("Got channel stats. Etag: " + getChannelStatsEtag)
			}

			if _, err := me.Client.UpdateLastViewedAt(channel.Id, true); err != nil {
				me.SendStatusError(err, "Failed to update last viewed at: "+channel.Id)
			} else {
				me.SendStatusActionSend("Updated last viewed at")
			}

			getChannelCount += 1
		}
	}
}
