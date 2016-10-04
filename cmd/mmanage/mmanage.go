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
		Short: "Login the specified users.",
		Run:   loginCmd,
	}

	var rootCmd = &cobra.Command{Use: "mmanage"}
	rootCmd.AddCommand(cmdTeams)
	rootCmd.Execute()
}

func loginCmd(cmd *cobra.Command, args []string) {
	context := cmdlib.MakeCommandContext()

	loginUsers(context, args)
}

func loginUsers(c *cmdlib.CommandContext, users []string) {
	client := cmdlib.GetClient(&c.LoadTestConfig.ConnectionConfiguration)

	loginResults := autocreation.LoginUsers(client, &c.LoadTestConfig.UsersConfiguration, users)

	for _, token := range loginResults.SessionTokens {
		c.Println(token)
	}
	c.PrintErrors(loginResults.Errors)
}
