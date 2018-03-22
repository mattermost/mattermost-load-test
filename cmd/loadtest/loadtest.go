// Copyright (c) 2017 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information

package main

import (
	"fmt"
	"io/ioutil"

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
	rootCmd.AddCommand(cmdPprof, cmdLoad, cmdGenerate)
	rootCmd.Execute()
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

	driverName := cfg.ConnectionConfiguration.DriverName
	dataSource := cfg.ConnectionConfiguration.DataSource

	// Ensure backwards compatibility with old configuration files.
	if driverName == "" {
		driverName = "mysql"
	}
	if dataSource == "" {
		dataSource = cfg.ConnectionConfiguration.DBEndpoint
	}

	loadtest.LoadPosts(cfg, driverName, dataSource)
}

func genBulkLoadCmd(cmd *cobra.Command, args []string) {
	cfg, err := loadtest.GetConfig()
	if err != nil {
		fmt.Println("Unable to find configuration file: " + err.Error())
	}
	results := loadtest.GenerateBulkloadFile(&cfg.LoadtestEnviromentConfig)
	ioutil.WriteFile("loadtestbulkload.json", results.File.Bytes(), 0644)
}
