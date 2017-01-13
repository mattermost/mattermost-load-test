// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package main

import (
	"fmt"
	"reflect"

	"github.com/spf13/viper"
)

type Connection struct {
	ServerURL     string
	WebsocketURL  string
	AdminEmail    string
	AdminPassword string
}

func (me *Connection) setDefaultsIfRequired() {
	if me.ServerURL == "" {
		me.ServerURL = "http://localhost:8065"
	}

	if me.WebsocketURL == "" {
		me.WebsocketURL = "ws://localhost:8065"
	}

	if me.AdminEmail == "" {
		me.AdminEmail = "test@test.com"
	}

	if me.AdminPassword == "" {
		me.AdminPassword = "passwd"
	}
}

type Config struct {
	Connection Connection
}

func (me *Config) setDefaultsIfRequired() {
	me.Connection.setDefaultsIfRequired()
}

func SetupConfig() {
	viper.SetConfigName("setup")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config/")
	err := viper.ReadInConfig()
	if err != nil {
		Error(fmt.Sprintf("Error while reading config file: %v", err.Error()))
	}

	Info(fmt.Sprintf("Loaded config file from %v", viper.ConfigFileUsed()))
}

func GetConfig() *Config {
	var config Config
	UnmarshalConfigStruct(&config)

	config.setDefaultsIfRequired()

	return &config
}

func UnmarshalConfigStruct(configStruct interface{}) error {
	return viper.Unmarshal(configStruct)
}

func UnmarshalConfigSubStruct(configStruct interface{}) error {
	return viper.Sub(reflect.ValueOf(configStruct).Elem().Type().Name()).Unmarshal(configStruct)
}
