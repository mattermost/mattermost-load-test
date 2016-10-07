// Copyright (c) 2016 Spinpunch, Inc. All Rights Reserved.
// See License.txt for license information.

package main

import (
	"github.com/mattermost/mattermost-load-test/autocreation"
	"github.com/mattermost/mattermost-load-test/cmd/cmdlib"
	"github.com/mattermost/mattermost-load-test/loadtestconfig"
	"github.com/spf13/cobra"
)

func main() {
	loadtestconfig.SetupConfig()

	cmdTeams := &cobra.Command{
		Use:   "login",
		Short: "Login all users.",
		Run:   loginCmd,
	}

	cmdJoinTeam := &cobra.Command{
		Use:   "jointeam",
		Short: "Join users to teams",
		Run:   joinTeamCmd,
	}

	cmdJoinChannel := &cobra.Command{
		Use:   "joinchannel",
		Short: "Join users to channels",
		Run:   joinChannelCmd,
	}
	loadtestconfig.SetIntFlag(cmdJoinChannel.Flags(), "num", "n", "Numer of channels to join each user to", "UsersConfiguration.NumChannelsToJoin", 1)

	var rootCmd = &cobra.Command{Use: "mmanage"}
	rootCmd.AddCommand(cmdTeams, cmdJoinTeam, cmdJoinChannel)
	rootCmd.Execute()
}

func joinTeamCmd(cmd *cobra.Command, args []string) {
	context := cmdlib.MakeCommandContext()

	joinUsersToTeam(context)
}

func joinUsersToTeam(c *cmdlib.CommandContext) {
	c.PrettyPrintln("Logging in users:")

	inputState := loadtestconfig.ServerStateFromStdin()

	teamIds := inputState.GetTeamIds()
	userIds := inputState.GetUserIds()

	client := cmdlib.GetClient(&c.LoadTestConfig.ConnectionConfiguration)

	errors := make([]error, 0, len(teamIds)*len(userIds))
	for _, team := range teamIds {
		for _, user := range userIds {
			_, err := client.AddUserToTeam(team, user)
			if err != nil {
				errors = append(errors, err)
			}
		}
	}

	c.Print(inputState.ToJson())
	c.PrintErrors(errors)
}

func loginCmd(cmd *cobra.Command, args []string) {
	context := cmdlib.MakeCommandContext()

	loginUsers(context)
}

func loginUsers(c *cmdlib.CommandContext) {
	c.PrettyPrintln("Logging in users:")

	inputState := loadtestconfig.ServerStateFromStdin()

	users := inputState.GetUserIds()

	client := cmdlib.GetClient(&c.LoadTestConfig.ConnectionConfiguration)

	loginResults := autocreation.LoginUsers(client, &c.LoadTestConfig.UsersConfiguration, users)

	for i, token := range loginResults.SessionTokens {
		if token != "" {
			inputState.Users[i].SessionToken = token
		}
	}

	c.Print(inputState.ToJson())

	c.PrintErrors(loginResults.Errors)
}

func joinChannelCmd(cmd *cobra.Command, args []string) {
	context := cmdlib.MakeCommandContext()

	joinUsersToChannel(context)
}

func joinUsersToChannel(c *cmdlib.CommandContext) {
	c.PrettyPrintln("Joining users to channel")

	inputState := loadtestconfig.ServerStateFromStdin()

	client := cmdlib.GetClient(&c.LoadTestConfig.ConnectionConfiguration)

	numChannelsToJoin := c.LoadTestConfig.UsersConfiguration.NumChannelsToJoin
	if len(inputState.Channels) < numChannelsToJoin {
		numChannelsToJoin = len(inputState.Channels)
	}

	errors := make([]error, 0, numChannelsToJoin*len(inputState.Users))
	for iUser, user := range inputState.Users {
		for channelOffset := 0; channelOffset < numChannelsToJoin; channelOffset++ {
			iChannel := (iUser + channelOffset) % len(inputState.Channels)
			channel := inputState.Channels[iChannel]
			client.SetTeamId(channel.TeamId)
			_, err := client.AddChannelMember(channel.Id, user.Id)
			if err != nil {
				errors = append(errors, err)
			} else {
				inputState.Users[iUser].ChannelsJoined = append(inputState.Users[iUser].ChannelsJoined, iChannel)
			}
		}
	}

	c.Print(inputState.ToJson())
	c.PrintErrors(errors)
}
