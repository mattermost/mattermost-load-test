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
	CreateThreads     int
	LoginThreads      int
}

type TeamsConfiguration struct {
	TeamNamePrefix  string
	TeamDisplayName string
	NumTeams        int
	UseRandomId     bool
	JoinThreads     int
}

type ChannelsConfiguration struct {
	ChannelNamePrefix  string
	ChannelDisplayName string
	NumChannelsPerTeam int
	TeamIds            []string
	UseRandomId        bool
	CreateThreads      int
	JoinThreads        int
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
	if config.CreateThreads == 0 {
		config.CreateThreads = 8
	}
	if config.JoinThreads == 0 {
		config.JoinThreads = 8
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
	if config.JoinThreads == 0 {
		config.JoinThreads = 8
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
	if config.NumChannelsToJoin == 0 {
		config.NumChannelsToJoin = 10
	}
	if config.NumUsers == 0 {
		config.NumUsers = 1
	}
	if config.CreateThreads == 0 {
		config.CreateThreads = 8
	}
	if config.LoginThreads == 0 {
		config.LoginThreads = 8
	}
}
