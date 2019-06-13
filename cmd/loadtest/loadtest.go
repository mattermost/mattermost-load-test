// Copyright (c) 2017 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mattermost/mattermost-load-test/loadtest"
	"github.com/mattermost/mattermost-server/mlog"
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
		Name:      "search-users",
		ShortDesc: "Test search users",
		Test:      &loadtest.TestSearchUsers,
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
	{
		Name:      "more-channels-browser",
		ShortDesc: "Test browsing more channels while under load",
		Test:      &loadtest.TestMoreChannelsBrowser,
	},
	{
		Name:      "autocomplete",
		ShortDesc: "Test autocomplete",
		Test:      &loadtest.TestAutocomplete,
	},
	{
		Name:      "reactions",
		ShortDesc: "Test reactions",
		Test:      &loadtest.TestReactions,
	},
}

func main() {
	cobra.OnInitialize(initConfig)

	cmdLoad := &cobra.Command{
		Use:   "loadposts",
		Short: "Load posts onto server",
		RunE:  loadCmd,
	}

	cmdGenerate := &cobra.Command{
		Use:   "genbulkload",
		Short: "Generate a bulkload file to be manually loaded onto a Mattermost server.",
		RunE:  genBulkLoadCmd,
	}

	cmdPprof := &cobra.Command{
		Use:   "pprof",
		Short: "Run a pprof",
		RunE:  pprofCmd,
	}

	var rootCmd = &cobra.Command{Use: "loadtest"}

	commands := make([]*cobra.Command, 0, len(tests))
	for _, test := range tests {
		currentTest := test
		commands = append(commands, &cobra.Command{
			Use:   currentTest.Name,
			Short: currentTest.ShortDesc,
			RunE: func(cmd *cobra.Command, args []string) error {
				mlog.Info("Running test", mlog.String("test", currentTest.Name))
				if err := loadtest.RunTest(currentTest.Test); err != nil {
					return errors.Wrap(err, "run test failed")
				}

				return nil
			},
		})
	}
	rootCmd.AddCommand(commands...)
	rootCmd.AddCommand(cmdPprof, cmdLoad, cmdGenerate)
	rootCmd.Execute()
}

func initConfig() {
	if err := loadtest.ReadConfig(); err != nil {
		fmt.Printf("Failed to initialize config: %s\n", err.Error())
		os.Exit(1)
	}

	cfg, err := loadtest.GetConfig()
	if err != nil {
		fmt.Printf("Failed to get logging config: %s\n", err.Error())
		os.Exit(1)
	}

	// Initalize logging
	log := mlog.NewLogger(&mlog.LoggerConfiguration{
		EnableConsole: cfg.LogSettings.EnableConsole,
		ConsoleJson:   cfg.LogSettings.ConsoleJson,
		ConsoleLevel:  strings.ToLower(cfg.LogSettings.ConsoleLevel),
		EnableFile:    cfg.LogSettings.EnableFile,
		FileJson:      cfg.LogSettings.FileJson,
		FileLevel:     strings.ToLower(cfg.LogSettings.FileLevel),
		FileLocation:  cfg.LogSettings.FileLocation,
	})

	// Redirect default golang logger to this logger
	mlog.RedirectStdLog(log)

	// Use this app logger as the global logger
	mlog.InitGlobalLogger(log)
}

func pprofCmd(cmd *cobra.Command, args []string) error {
	cfg := &loadtest.LoadTestConfig{}
	if err := viper.Unmarshal(cfg); err != nil {
		return errors.Wrap(err, "failed to read loadtest configuration")
	}

	loadtest.RunProfile(cfg.ConnectionConfiguration.PProfURL, cfg.ResultsConfiguration.PProfLength)

	return nil
}

func loadCmd(cmd *cobra.Command, args []string) error {
	cfg := &loadtest.LoadTestConfig{}
	if err := viper.Unmarshal(cfg); err != nil {
		return errors.Wrap(err, "failed to read loadtest configuration")
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

	return nil
}

func genBulkLoadCmd(cmd *cobra.Command, args []string) error {
	cfg := &loadtest.LoadTestConfig{}
	if err := viper.Unmarshal(cfg); err != nil {
		return errors.Wrap(err, "failed to read loadtest configuration")
	}

	results := loadtest.GenerateBulkloadFile(&cfg.LoadtestEnviromentConfig)
	ioutil.WriteFile("loadtestbulkload.json", results.File.Bytes(), 0644)

	return nil
}
