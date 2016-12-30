// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package main

import (
	"github.com/mattermost/mattermost-load-test/loadtestconfig"
	"github.com/spf13/cobra"
)

func main() {
	loadtestconfig.SetupConfig()

	cmdActiveUsers := &cobra.Command{
		Use:   "listenandpost",
		Short: "A number of active users posting and listening to websocket.",
		Run:   testListenAndPostCmd,
	}

	cmdGetChannel := &cobra.Command{
		Use:   "getchannel",
		Short: "A number of users getting channels as if they where the web client.",
		Run:   testGetChannelCmd,
	}

	var rootCmd = &cobra.Command{Use: "mloadtest"}
	rootCmd.AddCommand(cmdActiveUsers, cmdGetChannel)
	rootCmd.Execute()
}
