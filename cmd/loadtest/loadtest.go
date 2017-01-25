// Copyright (c) 2017 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information

package main

import (
	"fmt"

	"github.com/icrowley/fake"
	"github.com/mattermost/mattermost-load-test/loadtest"
	"github.com/spf13/cobra"
)

type TestItem struct {
	Name      string
	ShortDesc string
	Test      *loadtest.TestRun
}

//
// ADD YOUR NEW TEST HERE!
//
var tests []TestItem = []TestItem{
	{
		Name:      "basic",
		ShortDesc: "Basic test of posting",
		Test:      &loadtest.TestBasicPosting,
	},
	{
		Name:      "search",
		ShortDesc: "Test search",
		Test:      &loadtest.TestSearch,
	},
	{
		Name:      "getchannel",
		ShortDesc: "Test get channel",
		Test:      &loadtest.TestGetChannel,
	},
	{
		Name:      "all",
		ShortDesc: "Test Everything",
		Test:      &loadtest.TestAll,
	},
}

func main() {
	cmdPing := &cobra.Command{
		Use:   "ping",
		Short: "Check that our connection information to the server is correct.",
		Run:   pingCmd,
	}

	var rootCmd = &cobra.Command{Use: "loadtest"}

	commands := make([]*cobra.Command, 0, len(tests))
	for _, test := range tests {
		commands = append(commands, &cobra.Command{
			Use:   test.Name,
			Short: test.ShortDesc,
			Run: func(cmd *cobra.Command, args []string) {
				if err := loadtest.RunTest(test.Test); err != nil {
					fmt.Println("Run Test Failed: " + err.Error())
				}
			},
		})
	}
	rootCmd.AddCommand(commands...)
	rootCmd.AddCommand(cmdPing)
	rootCmd.Execute()
}

func pingCmd(cmd *cobra.Command, args []string) {
	// Print a paragraph
	fmt.Println(fake.Paragraph())
}
