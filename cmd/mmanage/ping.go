// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package main

import (
	"github.com/mattermost/mattermost-load-test/cmd/cmdlib"
	"github.com/spf13/cobra"
)

func pingCmd(cmd *cobra.Command, args []string) {
	context := cmdlib.MakeCommandContext()

	pingServer(context)
}

func pingServer(c *cmdlib.CommandContext) {
	client, err1 := cmdlib.GetClient(&c.LoadTestConfig.ConnectionConfiguration)
	if err1 != nil {
		c.PrintError("Failed to get client: ", err1)
		return
	}
	pinginfo, err2 := client.GetPing()
	if err2 != nil {
		c.PrintError("Ping to server failed: ", err2)
		return
	}
	c.Println(pinginfo)
}
