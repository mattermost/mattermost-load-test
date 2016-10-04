// Copyright (c) 2016 Spinpunch, Inc. All Rights Reserved.
// See License.txt for license information.

package main

import (
	"github.com/mattermost/mattermost-load-test/autocreation"
	"github.com/mattermost/mattermost-load-test/cmd/cmdlib"
	"github.com/spf13/cobra"
)

func createChannelsCmd(cmd *cobra.Command, args []string) {
	context := cmdlib.MakeCommandContext()

	context.LoadTestConfig.ChannelsConfiguration.TeamIds = args

	createChannels(context)
}

func createChannels(c *cmdlib.CommandContext) {
	c.PrettyPrintln("Creating Channels")
	client := cmdlib.GetClient(&c.LoadTestConfig.ConnectionConfiguration)
	channelResults := autocreation.CreateChannels(client, &c.LoadTestConfig.ChannelsConfiguration)

	c.PrintResultsHeader()
	c.PrettyPrintln("Channels: ")

	printChannelsResults(c, channelResults)
}

func printChannelsResults(c *cmdlib.CommandContext, results *autocreation.ChannelsCreationResult) {
	for _, result := range results.Channels {
		if result != nil {
			c.Println(result.ToJson())
		}
	}
	c.PrintErrors(results.Errors)
}
