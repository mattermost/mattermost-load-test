// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package main

import (
	"github.com/mattermost/mattermost-load-test/cmd/cmdlib"
	"github.com/mattermost/mattermost-load-test/loadtestconfig"
	"github.com/spf13/cobra"
)

func testGetChannelCmd(cmd *cobra.Command, args []string) {
	context := cmdlib.MakeCommandContext()

	testGetChannel(context)
}

func testGetChannel(c *cmdlib.CommandContext) {
	inputState := loadtestconfig.ServerStateFromStdin()

	c.Println("Starting get channel load test")

	StartUserEntities(c.LoadTestConfig, inputState, NewUserEntityGetChannel)
}
