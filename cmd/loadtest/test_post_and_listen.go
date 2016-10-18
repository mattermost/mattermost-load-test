// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package main

import (
	"strconv"

	"github.com/mattermost/mattermost-load-test/cmd/cmdlib"
	"github.com/mattermost/mattermost-load-test/loadtestconfig"
	"github.com/spf13/cobra"
)

func testListenAndPostCmd(cmd *cobra.Command, args []string) {
	context := cmdlib.MakeCommandContext()

	testListenAndPost(context)
}

func testListenAndPost(c *cmdlib.CommandContext) {
	inputState := loadtestconfig.ServerStateFromStdin()

	c.Println("Starting listen and post load test")

	posterConfig := NewUserEntityPosterConfig()
	c.LoadTestConfig.UserEntitiesConfiguration.EntityRampupDistanceMilliseconds = (posterConfig.PostingFrequencySeconds * 1000) / c.LoadTestConfig.UserEntitiesConfiguration.NumClientEntities
	c.Println("Calculated EntityRampupDistance: " + strconv.Itoa(c.LoadTestConfig.UserEntitiesConfiguration.EntityRampupDistanceMilliseconds))

	StartUserEntities(c.LoadTestConfig, inputState, NewUserEntityWebsocketListener, NewUserEntityPoster)
}
