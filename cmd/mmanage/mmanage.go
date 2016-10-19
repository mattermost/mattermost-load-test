// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package main

import (
	"fmt"
	"strconv"

	"github.com/mattermost/mattermost-load-test/autocreation"
	"github.com/mattermost/mattermost-load-test/cmd/cmdlib"
	"github.com/mattermost/mattermost-load-test/loadtestconfig"
	"github.com/mattermost/platform/model"
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

	cmdPing := &cobra.Command{
		Use:   "ping",
		Short: "Check that our connection information to the server is correct.",
		Run:   pingCmd,
	}

	loadtestconfig.SetIntFlag(cmdJoinChannel.Flags(), "num", "n", "Numer of channels to join each user to", "UsersConfiguration.NumChannelsToJoin", 1)

	var rootCmd = &cobra.Command{Use: "mmanage"}
	rootCmd.AddCommand(cmdTeams, cmdJoinTeam, cmdJoinChannel, cmdPing)
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

	client, err := cmdlib.GetClient(&c.LoadTestConfig.ConnectionConfiguration)
	if err != nil {
		c.PrintError("Failed to get client: ", err)
		return
	}

	joinResult := autocreation.JoinUsersToTeams(client, userIds, teamIds, c.LoadTestConfig.TeamCreationConfiguration.JoinThreads)

	c.Print(inputState.ToJson())
	c.PrintErrors(joinResult.Errors)
}

func loginCmd(cmd *cobra.Command, args []string) {
	context := cmdlib.MakeCommandContext()

	loginUsers(context)
}

func loginUsers(c *cmdlib.CommandContext) {
	c.PrettyPrintln("Logging in users:")

	inputState := loadtestconfig.ServerStateFromStdin()

	users := inputState.GetUserIds()

	client, err := cmdlib.GetClient(&c.LoadTestConfig.ConnectionConfiguration)
	if err != nil {
		c.PrintError("Failed to get client: ", err)
		return
	}

	loginResults := autocreation.LoginUsers(client, &c.LoadTestConfig.UserCreationConfguration, users)

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

// Old method for distrubuting users. Keep for later.
/*func joinUsersToChannel(c *cmdlib.CommandContext) {
	c.PrettyPrintln("Joining users to channel")

	inputState := loadtestconfig.ServerStateFromStdin()

	client, err := cmdlib.GetClient(&c.LoadTestConfig.ConnectionConfiguration)
	if err != nil {
		c.PrintError("Failed to get client: ", err)
		return
	}

	numChannelsToJoin := c.LoadTestConfig.UsersConfiguration.NumChannelsToJoin
	if len(inputState.Channels) < numChannelsToJoin {
		numChannelsToJoin = len(inputState.Channels)
	}

	c.PrintError("Joining users to " + strconv.Itoa(numChannelsToJoin) + " channels each.")
	errors := make([]error, numChannelsToJoin*len(inputState.Users))
	autocreation.ThreadSplit(len(inputState.Users), c.LoadTestConfig.ChannelsConfiguration.JoinThreads, func(iUser int) {
		for channelOffset := 0; channelOffset < numChannelsToJoin; channelOffset++ {
			iChannel := (iUser + channelOffset) % len(inputState.Channels)
			channel := inputState.Channels[iChannel]
			data := make(map[string]string)
			data["user_id"] = inputState.Users[iUser].Id
			_, err := client.DoApiPost(fmt.Sprintf("/teams/%v/channels/%v/add", channel.TeamId, channel.Id), model.MapToJson(data))
			if err != nil {
				errors[iUser] = err
			} else {
				inputState.Users[iUser].ChannelsJoined = append(inputState.Users[iUser].ChannelsJoined, iChannel)
			}
		}
	})

	c.Print(inputState.ToJson())
	c.PrintErrors(errors)
}*/

func joinUsersToChannel(c *cmdlib.CommandContext) {
	c.PrettyPrintln("Joining users to channel")

	inputState := loadtestconfig.ServerStateFromStdin()

	client, err := cmdlib.GetClient(&c.LoadTestConfig.ConnectionConfiguration)
	if err != nil {
		c.PrintError("Failed to get client: ", err)
		return
	}

	numChannelsToJoin := c.LoadTestConfig.UserCreationConfguration.NumChannelsToJoin
	if len(inputState.Channels) < numChannelsToJoin {
		numChannelsToJoin = len(inputState.Channels)
	}

	numUsers := len(inputState.Users)
	numChannels := len(inputState.Channels)

	if (numUsers*numChannelsToJoin)%numChannels != 0 {
		c.PrintError("Invalid users and channels to join sizes. Num users must be divisable by the number of channels.")
	}
	numUsersPerChannel := (numUsers * numChannelsToJoin) / numChannels

	c.PrintError("Joining users to " + strconv.Itoa(numChannelsToJoin) + " channels each.")
	c.PrintError("This is " + strconv.Itoa(numUsersPerChannel) + " users per channel.")
	errors := make([]error, numChannelsToJoin*len(inputState.Users))
	autocreation.ThreadSplit(numUsers, c.LoadTestConfig.ChannelCreationConfiguration.JoinThreads, func(iUser int) {
		for iChannel := (iUser / numUsersPerChannel) * numChannelsToJoin; iChannel < ((iUser/numUsersPerChannel)+1)*(numChannelsToJoin); iChannel += 1 {
			channel := inputState.Channels[iChannel]
			user := inputState.Users[iUser]
			data := make(map[string]string)
			data["user_id"] = user.Id
			_, err := client.DoApiPost(fmt.Sprintf("/teams/%v/channels/%v/add", channel.TeamId, channel.Id), model.MapToJson(data))
			if err != nil {
				errors[iUser] = err
			} else {
				inputState.Users[iUser].ChannelsJoined = append(inputState.Users[iUser].ChannelsJoined, iChannel)
			}
		}
	})

	c.Print(inputState.ToJson())
	c.PrintErrors(errors)
}
