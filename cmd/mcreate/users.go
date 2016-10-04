// Copyright (c) 2016 Spinpunch, Inc. All Rights Reserved.
// See License.txt for license information.

package main

import (
	"github.com/mattermost/mattermost-load-test/autocreation"
	"github.com/mattermost/mattermost-load-test/cmd/cmdlib"
	"github.com/spf13/cobra"
)

func createUsersCmd(cmd *cobra.Command, args []string) {
	context := cmdlib.MakeCommandContext()

	createUsers(context)
}

func createUsers(c *cmdlib.CommandContext) {
	c.PrettyPrintln("Creating Users")

	client := cmdlib.GetClient(&c.LoadTestConfig.ConnectionConfiguration)

	results := autocreation.CreateUsers(client, &c.LoadTestConfig.UsersConfiguration)

	c.PrintResultsHeader()
	c.PrettyPrintln("Users: ")
	printUsersResults(c, results)
}

func printUsersResults(c *cmdlib.CommandContext, results *autocreation.UsersCreationResult) {
	for _, result := range results.Users {
		if result != nil {
			c.Println(result.ToJson())
		}
	}
	c.PrintErrors(results.Errors)
}
