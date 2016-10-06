// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package main

import (
	"github.com/mattermost/mattermost-load-test/cmd/loadtest/oldloadtest"
	"github.com/mattermost/mattermost-load-test/loadtestconfig"
	"github.com/spf13/cobra"
)

func main() {
	loadtestconfig.SetupConfig()

	cmdListenTest := &cobra.Command{
		Use:   "old",
		Short: "Run the old loadtests",
		Run: func(cmd *cobra.Command, args []string) {
			oldloadtest.RunOldLoadTests()
		},
	}

	cmdActiveUsers := &cobra.Command{
		Use:   "active",
		Short: "A number of active users posting and listening to websocket.",
		Run:   testActiveCmd,
	}

	var rootCmd = &cobra.Command{Use: "mloadtest"}
	rootCmd.AddCommand(cmdListenTest, cmdActiveUsers)
	rootCmd.Execute()
}
