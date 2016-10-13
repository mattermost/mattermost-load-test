// Copyright (c) 2016 Spinpunch, Inc. All Rights Reserved.
// See License.txt for license information.

package main

import (
	"github.com/mattermost/mattermost-load-test/autocreation"
	"github.com/mattermost/mattermost-load-test/cmd/cmdlib"
	"github.com/mattermost/mattermost-load-test/loadtestconfig"
	"github.com/spf13/cobra"
)

func createTeamsCmd(cmd *cobra.Command, args []string) {
	context := cmdlib.MakeCommandContext()
	createTeams(context)
}

func createTeams(c *cmdlib.CommandContext) {
	c.PrettyPrintln("Creating Teams")

	inputState := loadtestconfig.ServerStateFromStdin()

	client, err := cmdlib.GetClient(&c.LoadTestConfig.ConnectionConfiguration)
	if err != nil {
		c.PrintError("Failed to get client: ", err)
		return
	}

	results := autocreation.CreateTeams(client, &c.LoadTestConfig.TeamsConfiguration)

	for _, result := range results.Teams {
		if result != nil {
			inputState.Teams = append(inputState.Teams, loadtestconfig.ServerStateTeam{Id: result.Id})
		}
	}

	c.PrintResultsHeader()
	c.PrettyPrintln("Teams: ")
	c.Print(inputState.ToJson())
	c.PrintErrors(results.Errors)
}
