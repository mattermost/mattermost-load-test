// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package main

import (
	"time"

	"github.com/mattermost/mattermost-load-test/loadtestconfig"
	"github.com/mattermost/platform/model"
)

type UserEntityPoster struct {
	UserEntityConfig
}

type UserEntityPosterConfiguration struct {
	PostingFrequencySeconds int
}

func NewUserEntityPosterConfig() UserEntityPosterConfiguration {
	var userEntityPosterConfig UserEntityPosterConfiguration
	var cfg struct {
		UserEntityPosterConfiguration UserEntityPosterConfiguration
	}
	loadtestconfig.UnmarshalConfigStruct(&cfg)

	userEntityPosterConfig = cfg.UserEntityPosterConfiguration

	if userEntityPosterConfig.PostingFrequencySeconds == 0 {
		userEntityPosterConfig.PostingFrequencySeconds = 1
	}

	return userEntityPosterConfig
}

func NewUserEntityPoster(cfg UserEntityConfig) UserEntity {
	return &UserEntityPoster{
		UserEntityConfig: cfg,
	}
}

func (me *UserEntityPoster) Start() {
	me.SendStatusLaunching()
	defer me.StopEntityWaitGroup.Done()
	posterConfig := NewUserEntityPosterConfig()

	// Allows us to perform our action every x seconds
	postTicker := time.NewTicker(time.Second * time.Duration(posterConfig.PostingFrequencySeconds))
	defer postTicker.Stop()

	var postCount int64 = 0

	me.SendStatusActive("Posting")
	for {
		select {
		case <-me.StopEntityChannel:
			me.SendStatusStopped("")
			return
		case <-postTicker.C:
			channel := me.State.Channels[me.EntityUser.ChannelsJoined[postCount%int64(len(me.EntityUser.ChannelsJoined))]]
			me.Client.SetTeamId(channel.TeamId)
			post := &model.Post{
				ChannelId: channel.Id,
				Message:   "Test message",
			}
			_, err := me.Client.CreatePost(post)
			if err != nil {
				me.SendStatusError(err, "Failed to post message")
			} else {
				me.SendStatusActionSend("Posted Message")
			}
			postCount++
		}
	}
}
