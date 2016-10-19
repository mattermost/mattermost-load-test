// Copyright (c) 2016 Spinpunch, Inc. All Rights Reserved.
// See License.txt for license information.

package main

import (
	"github.com/mattermost/mattermost-load-test/autocreation"
	"github.com/mattermost/mattermost-load-test/cmd/cmdlib"
	"github.com/mattermost/mattermost-load-test/loadtestconfig"
	"github.com/spf13/cobra"
)

func createChannelsCmd(cmd *cobra.Command, args []string) {
	context := cmdlib.MakeCommandContext()

	createChannels(context)
}

func createChannels(c *cmdlib.CommandContext) {
	c.PrettyPrintln("Creating Channels")
	inputState := loadtestconfig.ServerStateFromStdin()
	client, err := cmdlib.GetClient(&c.LoadTestConfig.ConnectionConfiguration)
	if err != nil {
		c.PrintError("Failed to get client: ", err)
		return
	}

	teamIds := inputState.GetTeamIds()

	c.LoadTestConfig.ChannelCreationConfiguration.TeamIds = teamIds

	channelResults := autocreation.CreateChannels(client, &c.LoadTestConfig.ChannelCreationConfiguration)

	for _, result := range channelResults.Channels {
		if result != nil {
			inputState.Channels = append(inputState.Channels, loadtestconfig.ServerStateChannel{Id: result.Id, TeamId: result.TeamId})
		}
	}

	c.PrintResultsHeader()
	c.PrettyPrintln("Channels: ")
	c.Print(inputState.ToJson())
	c.PrintErrors(channelResults.Errors)
}
