package main

import (
	"github.com/mattermost/mattermost-load-test/cmd/cmdlib"
	"github.com/spf13/cobra"
)

func createAllCmd(cmd *cobra.Command, args []string) {
	context := cmdlib.MakeCommandContext()

	createAll(context)
}

func createAll(c *cmdlib.CommandContext) {
	/*
		c.PrettyPrintln("Creating Teams, Channels, and Users")

		client := cmdlib.GetClient(&c.LoadTestConfig.ConnectionConfiguration)

		usersResult := autocreation.CreateUsers(client, &c.LoadTestConfig.UsersConfiguration)
		teamsResults := autocreation.CreateTeams(client, &c.LoadTestConfig.TeamsConfiguration)
		c.LoadTestConfig.ChannelsConfiguration.TeamIds = teamsResults.GetTeamIds()
		channelsResult := autocreation.CreateChannels(client, &c.LoadTestConfig.ChannelsConfiguration)

			printUsersResults(c, usersResult)
			printTeamsResults(c, teamsResults)
			printChannelsResults(c, channelsResult)*/
}
