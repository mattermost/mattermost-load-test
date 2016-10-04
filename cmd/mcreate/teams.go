// Copyright (c) 2016 Spinpunch, Inc. All Rights Reserved.
// See License.txt for license information.

package main

import (
	"github.com/mattermost/mattermost-load-test/autocreation"
	"github.com/mattermost/mattermost-load-test/cmd/cmdlib"
	"github.com/spf13/cobra"
)

func createTeamsCmd(cmd *cobra.Command, args []string) {
	context := cmdlib.MakeCommandContext()
	createTeams(context)
}

func createTeams(c *cmdlib.CommandContext) {
	client := cmdlib.GetClient(&c.LoadTestConfig.ConnectionConfiguration)

	results := autocreation.CreateTeams(client, &c.LoadTestConfig.TeamsConfiguration)

	c.PrintResultsHeader()
	c.PrettyPrintln("Teams: ")
	printTeamsResults(c, results)
}

func printTeamsResults(c *cmdlib.CommandContext, results *autocreation.TeamsCreationResult) {
	for _, result := range results.Teams {
		if result != nil {
			c.Println(result.ToJson())
		}
	}
	c.PrintErrors(results.Errors)
}
