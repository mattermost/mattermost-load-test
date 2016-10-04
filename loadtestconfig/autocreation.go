// Copyright (c) 2016 Spinpunch, Inc. All Rights Reserved.
// See License.txt for license information.

package loadtestconfig

type UsersConfiguration struct {
	UserEmailPrefix string
	UserEmailDomain string
	UserFirstName   string
	UserLastName    string
	UserPassword    string
	NumUsers        int
	UseRandomId     bool
}

type TeamsConfiguration struct {
	TeamNamePrefix  string
	TeamDisplayName string
	NumTeams        int
	UseRandomId     bool
}

type ChannelsConfiguration struct {
	ChannelNamePrefix  string
	ChannelDisplayName string
	NumChannelsPerTeam int
	TeamIds            []string
	UseRandomId        bool
}

func (config *ChannelsConfiguration) SetDefaultsIfRequired() {
	if config.ChannelNamePrefix == "" {
		config.ChannelNamePrefix = "autocreatedchannel"
	}
	if config.ChannelDisplayName == "" {
		config.ChannelDisplayName = "Autocreated Channel"
	}
	if config.NumChannelsPerTeam == 0 {
		config.NumChannelsPerTeam = 1
	}
}

func (config *TeamsConfiguration) SetDefaultsIfRequired() {
	if config.TeamNamePrefix == "" {
		config.TeamNamePrefix = "autocreatedteam"
	}
	if config.TeamDisplayName == "" {
		config.TeamDisplayName = "Autocreated Team"
	}
	if config.NumTeams == 0 {
		config.NumTeams = 1
	}
}

func (config *UsersConfiguration) SetDefaultsIfRequired() {
	if config.UserEmailPrefix == "" {
		config.UserEmailPrefix = "autocreated_test_user_"
	}
	if config.UserEmailDomain == "" {
		config.UserEmailDomain = "+success@simulator.amazonses.com"
	}
	if config.UserFirstName == "" {
		config.UserFirstName = "TestFirst"
	}
	if config.UserLastName == "" {
		config.UserLastName = "TestLast"
	}
	if config.UserPassword == "" {
		config.UserPassword = "passwd"
	}
	if config.NumUsers == 0 {
		config.NumUsers = 1
	}
}
