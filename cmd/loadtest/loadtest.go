// Copyright (c) 2017 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information

package main

import (
	"fmt"
	"io/ioutil"

	"github.com/icrowley/fake"
	"github.com/mattermost/mattermost-load-test/cmdlog"
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
	{
		Name:      "townsquare-spam",
		ShortDesc: "Test town-square getting spammed",
		Test:      &loadtest.TestTownSquareSpam,
	},
	{
		Name:      "team-leave-join",
		ShortDesc: "Test leaving and joining a team while under load",
		Test:      &loadtest.TestLeaveJoinTeam,
	},
}

func main() {
	cmdPing := &cobra.Command{
		Use:   "ping",
		Short: "Check that our connection information to the server is correct.",
		Run:   pingCmd,
	}

	cmdLoad := &cobra.Command{
		Use:   "loadposts",
		Short: "Load posts onto server",
		Run:   loadCmd,
	}

	cmdGenerate := &cobra.Command{
		Use:   "genbulkload",
		Short: "Generate a bulkload file to be manually loaded onto a Mattermost server.",
		Run:   genBulkLoadCmd,
	}

	cmdPprof := &cobra.Command{
		Use:   "pprof",
		Short: "Run a pprof",
		Run:   pprofCmd,
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
	rootCmd.AddCommand(cmdPing, cmdPprof, cmdLoad, cmdGenerate)
	rootCmd.Execute()
}

func pingCmd(cmd *cobra.Command, args []string) {
	// Print a paragraph
	fmt.Println(fake.Paragraph())
}

func pprofCmd(cmd *cobra.Command, args []string) {
	cfg, err := loadtest.GetConfig()
	if err != nil {
		fmt.Println("Unable to find configuration file: " + err.Error())
	}
	loadtest.RunProfile(cfg.ConnectionConfiguration.PProfURL, cfg.ResultsConfiguration.PProfLength)
}

func loadCmd(cmd *cobra.Command, args []string) {
	cmdlog.SetConsoleLog()
	cfg, err := loadtest.GetConfig()
	if err != nil {
		fmt.Println("Unable to find configuration file: " + err.Error())
	}
	loadtest.LoadPosts(cfg, cfg.ConnectionConfiguration.DBEndpoint)
}

func genBulkLoadCmd(cmd *cobra.Command, args []string) {
	cfg, err := loadtest.GetConfig()
	if err != nil {
		fmt.Println("Unable to find configuration file: " + err.Error())
	}
	results := loadtest.GenerateBulkloadFile(&cfg.LoadtestEnviromentConfig)
	ioutil.WriteFile("loadtestbulkload.json", results.File.Bytes(), 0644)
}
