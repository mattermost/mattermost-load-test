// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package loadtest

import (
	"bytes"
	"fmt"
	"html/template"
	"reflect"

	"github.com/mattermost/mattermost-load-test/autocreation"
	"github.com/mattermost/mattermost-load-test/cmdlog"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type LoadTestConfig struct {
	LoadtestEnviromentConfig  autocreation.LoadtestEnviromentConfig
	ConnectionConfiguration   ConnectionConfiguration
	UserEntitiesConfiguration UserEntitiesConfiguration
	DisplayConfiguration      DisplayConfiguration
	ResultsConfiguration      ResultsConfiguration
}

type UserEntitiesConfiguration struct {
	TestLengthMinutes                 int
	NumActiveEntities                 int
	EntityStartNum                    int
	ActionRateMilliseconds            int
	ActionRateMaxVarianceMilliseconds int
	EnableRequestTiming               bool
	UploadImageChance                 float64
	DoStatusPolling                   bool
	RandomizeEntitySelection          bool
}

type ConnectionConfiguration struct {
	ServerURL            string
	WebsocketURL         string
	PProfURL             string
	LocalCommands        bool
	SSHHostnamePort      string
	SSHUsername          string
	SSHPassword          string
	SSHKey               string
	MattermostInstallDir string
	ConfigFileLoc        string
	AdminEmail           string
	AdminPassword        string
	SkipBulkload         bool
	WaitForServerStart   bool
}

type ResultsConfiguration struct {
	CustomReportText     string
	SendReportToMMServer bool
	ResultsServerURL     string
	ResultsChannelId     string
	ResultsUsername      string
	ResultsPassword      string
	PProfDelayMinutes    int
	PProfLength          int
}

type DisplayConfiguration struct {
	ShowUI       bool
	LogToConsole bool
}

func GetConfig() (*LoadTestConfig, error) {
	viper.SetConfigName("loadtestconfig")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config/")
	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	var cfg *LoadTestConfig

	if err := unmarshalConfigStruct(&cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (cfg *LoadTestConfig) PrintReport() string {
	const settingsTemplateString = `Test Length: {{.UserEntitiesConfiguration.TestLengthMinutes}} minutes
Number of Active Entities: {{.UserEntitiesConfiguration.NumActiveEntities}}
Action Rate: {{.UserEntitiesConfiguration.ActionRateMilliseconds}} ms
Action Rate Max Variance: {{.UserEntitiesConfiguration.ActionRateMaxVarianceMilliseconds}} ms
Server: {{.ConnectionConfiguration.ServerURL}}
{{.DisplayConfiguration.CustomReportText}}
`
	settingsTemplate := template.Must(template.New("settings").Parse(settingsTemplateString))

	var buf bytes.Buffer
	fmt.Fprintln(&buf, "")
	fmt.Fprintln(&buf, "--------- Settings Report ------------")

	if err := settingsTemplate.Execute(&buf, cfg); err != nil {
		cmdlog.Error("Error executing template: " + err.Error())
	}

	fmt.Fprintln(&buf, "")

	return buf.String()
}

func unmarshalConfigStruct(configStruct interface{}) error {
	return viper.Unmarshal(configStruct)
}

func unmarshalConfigSubStruct(configStruct interface{}) error {
	return viper.Sub(reflect.ValueOf(configStruct).Elem().Type().Name()).Unmarshal(configStruct)
}

func SetIntFlag(flags *pflag.FlagSet, full, short, helpText, configFileSetting string, defaultValue int) {
	flags.IntP(full, short, defaultValue, helpText)
	viper.SetDefault(configFileSetting, defaultValue)
	viper.BindPFlag(configFileSetting, flags.Lookup(full))
}

func SetBoolFlag(flags *pflag.FlagSet, full, short, helpText, configFileSetting string, defaultValue bool) {
	flags.BoolP(full, short, defaultValue, helpText)
	viper.SetDefault(configFileSetting, defaultValue)
	viper.BindPFlag(configFileSetting, flags.Lookup(full))
}
