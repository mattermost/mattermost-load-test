// Copyright (c) 2016 Spinpunch, Inc. All Rights Reserved.
// See License.txt for license information.

package main

import (
	"github.com/mattermost/mattermost-load-test/loadtestconfig"
	"github.com/spf13/cobra"
)

func main() {
	loadtestconfig.SetupConfig()

	cmdTeams := &cobra.Command{
		Use:   "teams",
		Short: "Create some teams",
		Run:   createTeamsCmd,
	}
	loadtestconfig.SetIntFlag(cmdTeams.Flags(), "num", "n", "Numer of teams to create", "TeamsConfiguration.NumTeams", 1)
	loadtestconfig.SetBoolFlag(cmdTeams.Flags(), "rand", "r", "Use a random id to make team unique", "TeamsConfiguration.UseRandomId", false)

	cmdUsers := &cobra.Command{
		Use:   "users",
		Short: "Create some users",
		Run:   createUsersCmd,
	}
	loadtestconfig.SetIntFlag(cmdUsers.Flags(), "num", "n", "Numer of users to create", "UsersConfiguration.NumUsers", 1)
	loadtestconfig.SetBoolFlag(cmdUsers.Flags(), "rand", "r", "Use a random id to make user unique", "UsersConfiguration.UseRandomId", false)

	cmdChannels := &cobra.Command{
		Use:   "channels [teamIds]",
		Short: "Create some channels for teams",
		Run:   createChannelsCmd,
	}
	loadtestconfig.SetIntFlag(cmdChannels.Flags(), "num", "n", "Number of channels to create per team", "ChannelsConfiguration.NumChannelsPerTeam", 1)

	cmdAll := &cobra.Command{
		Use:   "all",
		Short: "Create teams, channels, and users as specified in configuration file",
		Run:   createAllCmd,
	}

	var rootCmd = &cobra.Command{Use: "mcreate"}

	loadtestconfig.SetBoolFlag(rootCmd.PersistentFlags(), "pretty", "p", "Pretty print the output of the command", "GlobalCommandConfig.PrettyPrint", false)

	rootCmd.AddCommand(cmdTeams, cmdUsers, cmdChannels, cmdAll)
	rootCmd.Execute()
}
