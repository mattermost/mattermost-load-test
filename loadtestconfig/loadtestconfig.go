// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package loadtestconfig

import (
	"encoding/json"
	"io/ioutil"
	"reflect"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type LoadTestConfig struct {
	ConnectionConfig
	LoadConfig
	AdvancedLoadConfig
	MultiServerConfig
	EnviromentSettings
}

type ConnectionConfig struct {
	ServerURL     string
	WebsocketURL  string
	AdminEmail    string
	AdminPassword string
}

type LoadConfig struct {
	NumUsers        int
	UtilizationRate float64
}

type AdvancedLoadConfig struct {
	PostsPerWorkDay         float64
	NumChannelsViewedPerDay float64
}

type MultiServerConfig struct {
	NumLoadtestServers   int
	LoadtestServerNumber int
}

type EnviromentSettings struct {
	NumTeams           int
	NumChannelsPerTeam int
	NumTeamsPerUser    int
	NumChannelsPerUser int
}

var defaultSettings = LoadTestConfig{
	ConnectionConfig: ConnectionConfig{
		ServerURL:     "http://localhost:8065",
		WebsocketURL:  "ws://localhost:8065",
		AdminEmail:    "test@test.com",
		AdminPassword: "passwd",
	},
	LoadConfig: LoadConfig{
		NumUsers:        10000,
		UtilizationRate: 0.35,
	},
	AdvancedLoadConfig: AdvancedLoadConfig{
		PostsPerWorkDay:         40.0,
		NumChannelsViewedPerDay: 40.0,
	},
	MultiServerConfig: MultiServerConfig{
		NumLoadtestServers:   1,
		LoadtestServerNumber: 1,
	},
	EnviromentSettings: EnviromentSettings{
		NumTeams:           1,
		NumChannelsPerTeam: 1000,
		NumTeamsPerUser:    1,
		NumChannelsPerUser: 10,
	},
}

func (config *LoadTestConfig) setDefaultsIfRequired() {
	if config.ServerURL == "" {
		config.ServerURL = defaultSettings.ServerURL
	}

	if config.WebsocketURL == "" {
		config.WebsocketURL = defaultSettings.WebsocketURL
	}

	if config.AdminEmail == "" {
		config.AdminEmail = defaultSettings.AdminEmail
	}

	if config.AdminPassword == "" {
		config.AdminPassword = defaultSettings.AdminPassword
	}

	if config.NumUsers == 0 {
		config.NumUsers = defaultSettings.NumUsers
	}

	if config.UtilizationRate == 0 {
		config.UtilizationRate = defaultSettings.UtilizationRate
	}

	if config.PostsPerWorkDay == 0 {
		config.PostsPerWorkDay = defaultSettings.PostsPerWorkDay
	}

	if config.NumChannelsViewedPerDay == 0 {
		config.NumChannelsViewedPerDay = defaultSettings.NumChannelsViewedPerDay
	}

	if config.NumLoadtestServers == 0 {
		config.NumLoadtestServers = defaultSettings.NumLoadtestServers
	}

	if config.LoadtestServerNumber == 0 {
		config.LoadtestServerNumber = defaultSettings.LoadtestServerNumber
	}

	if config.NumTeams == 0 {
		config.NumTeams = defaultSettings.NumTeams
	}

	if config.NumChannelsPerTeam == 0 {
		config.NumChannelsPerTeam = defaultSettings.NumChannelsPerTeam
	}

	if config.NumTeamsPerUser == 0 {
		config.NumTeamsPerUser = defaultSettings.NumTeamsPerUser
	}

	if config.NumChannelsPerUser == 0 {
		config.NumChannelsPerUser = defaultSettings.NumChannelsPerUser
	}
}

func SetupConfig() error {
	viper.SetConfigName("loadtestconfig")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config/")
	if err := viper.ReadInConfig(); err != nil {
		defaultConfig := LoadTestConfig{}
		defaultConfig.setDefaultsIfRequired()
		if marshaled, err := json.MarshalIndent(defaultConfig, "", "    "); err != nil {
			return err
		} else {
			if err := ioutil.WriteFile("loadtestconfig.json", marshaled, 0644); err != nil {
				return err
			}
		}
		if err := viper.ReadInConfig(); err != nil {
			return err
		}
	}

	return nil
}

func GetUsedConfigFile() string {
	return viper.ConfigFileUsed()
}

func UnmarshalConfigStruct(configStruct interface{}) error {
	return viper.Unmarshal(configStruct)
}

func UnmarshalConfigSubStruct(configStruct interface{}) error {
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

func GetConfig() *LoadTestConfig {
	var config LoadTestConfig
	UnmarshalConfigStruct(&config)

	config.setDefaultsIfRequired()

	return &config
}
