// Copyright (c) 2016 Spinpunch, Inc. All Rights Reserved.
// See License.txt for license information.

package main

import (
	"github.com/mattermost/mattermost-load-test/autocreation"
	"github.com/mattermost/mattermost-load-test/cmd/cmdlib"
	"github.com/mattermost/mattermost-load-test/loadtestconfig"
	"github.com/spf13/cobra"
)

func createUsersCmd(cmd *cobra.Command, args []string) {
	context := cmdlib.MakeCommandContext()

	createUsers(context)
}

func createUsers(c *cmdlib.CommandContext) {
	c.PrettyPrintln("Creating Users")

	inputState := loadtestconfig.ServerStateFromStdin()

	client := cmdlib.GetClient(&c.LoadTestConfig.ConnectionConfiguration)

	results := autocreation.CreateUsers(client, &c.LoadTestConfig.UsersConfiguration)

	for _, result := range results.Users {
		if result != nil {
			inputState.Users = append(inputState.Users, loadtestconfig.ServerStateUser{Id: result.Id})
		}
	}

	c.PrintResultsHeader()
	c.PrettyPrintln("Users: ")
	c.Print(inputState.ToJson())
	c.PrintErrors(results.Errors)
}
