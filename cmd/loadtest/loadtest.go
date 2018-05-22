// Copyright (c) 2017 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information

package main

import (
	"io/ioutil"

	"github.com/mattermost/mattermost-load-test/loadtest"
	"github.com/mattermost/mattermost-server/mlog"
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
	{
		Name:      "user-deactivation",
		ShortDesc: "Test deactivating and reactivating users while under load",
		Test:      &loadtest.TestDeactivation,
	},
}

func main() {
	// Initalize logging
	log := mlog.NewLogger(&mlog.LoggerConfiguration{
		EnableConsole: true,
		ConsoleJson:   true,
		ConsoleLevel:  mlog.LevelDebug,
	})

	// Redirect default golang logger to this logger
	mlog.RedirectStdLog(log)

	// Use this app logger as the global logger
	mlog.InitGlobalLogger(log)

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
		currentTest := test
		commands = append(commands, &cobra.Command{
			Use:   currentTest.Name,
			Short: currentTest.ShortDesc,
			Run: func(cmd *cobra.Command, args []string) {
				mlog.Info("Running test", mlog.String("test", currentTest.Name))
				if err := loadtest.RunTest(currentTest.Test); err != nil {
					mlog.Error("Run Test Failed", mlog.String("test", currentTest.Name), mlog.Err(err))
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
		mlog.Error("Unable to find configuration file", mlog.Err(err))
	}
	loadtest.RunProfile(cfg.ConnectionConfiguration.PProfURL, cfg.ResultsConfiguration.PProfLength)
}

func loadCmd(cmd *cobra.Command, args []string) {
	cfg, err := loadtest.GetConfig()
	if err != nil {
		mlog.Error("Unable to find configuration file", mlog.Err(err))
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
		mlog.Error("Unable to find configuration file", mlog.Err(err))
	}
	results := loadtest.GenerateBulkloadFile(&cfg.LoadtestEnviromentConfig)
	ioutil.WriteFile("loadtestbulkload.json", results.File.Bytes(), 0644)
}
