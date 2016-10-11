// Copyright (c) 2016 Spinpunch, Inc. All Rights Reserved.
// See License.txt for license information.

package loadtestconfig

type UsersConfiguration struct {
	UserEmailPrefix   string
	UserEmailDomain   string
	UserFirstName     string
	UserLastName      string
	UserUsername      string
	UserPassword      string
	NumUsers          int
	UseRandomId       bool
	NumChannelsToJoin int
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
		config.ChannelNamePrefix = "autoc"
	}
	if config.ChannelDisplayName == "" {
		config.ChannelDisplayName = "AutoC "
	}
	if config.NumChannelsPerTeam == 0 {
		config.NumChannelsPerTeam = 1
	}
}

func (config *TeamsConfiguration) SetDefaultsIfRequired() {
	if config.TeamNamePrefix == "" {
		config.TeamNamePrefix = "autot"
	}
	if config.TeamDisplayName == "" {
		config.TeamDisplayName = "AutoT "
	}
	if config.NumTeams == 0 {
		config.NumTeams = 1
	}
}

func (config *UsersConfiguration) SetDefaultsIfRequired() {
	if config.UserEmailPrefix == "" {
		config.UserEmailPrefix = "autou_"
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
	if config.UserUsername == "" {
		config.UserUsername = "autou"
	}
	if config.UserPassword == "" {
		config.UserPassword = "passwd"
	}
	if config.NumUsers == 0 {
		config.NumUsers = 1
	}
}
