// Copyright (c) 2016 Spinpunch, Inc. All Rights Reserved.
// See License.txt for license information.

package loadtestconfig

type UserCreationConfiguration struct {
	EmailPrefix       string
	EmailDomain       string
	FirstName         string
	LastName          string
	Username          string
	Password          string
	Num               int
	UseRandomId       bool
	NumChannelsToJoin int
	CreateThreads     int
	LoginThreads      int
}

type TeamCreationConfiguration struct {
	Name        string
	DisplayName string
	Num         int
	UseRandomId bool
	JoinThreads int
}

type ChannelCreationConfiguration struct {
	Name          string
	DisplayName   string
	NumPerTeam    int
	TeamIds       []string
	UseRandomId   bool
	CreateThreads int
	JoinThreads   int
}

func (config *ChannelCreationConfiguration) SetDefaultsIfRequired() {
	if config.Name == "" {
		config.Name = "autoc"
	}
	if config.DisplayName == "" {
		config.DisplayName = "AutoC "
	}
	if config.NumPerTeam == 0 {
		config.NumPerTeam = 1
	}
	if config.CreateThreads == 0 {
		config.CreateThreads = 8
	}
	if config.JoinThreads == 0 {
		config.JoinThreads = 8
	}
}

func (config *TeamCreationConfiguration) SetDefaultsIfRequired() {
	if config.Name == "" {
		config.Name = "autot"
	}
	if config.DisplayName == "" {
		config.DisplayName = "AutoT "
	}
	if config.Num == 0 {
		config.Num = 1
	}
	if config.JoinThreads == 0 {
		config.JoinThreads = 8
	}
}

func (config *UserCreationConfiguration) SetDefaultsIfRequired() {
	if config.EmailPrefix == "" {
		config.EmailPrefix = "autou_"
	}
	if config.EmailDomain == "" {
		config.EmailDomain = "+success@simulator.amazonses.com"
	}
	if config.FirstName == "" {
		config.FirstName = "TestFirst"
	}
	if config.LastName == "" {
		config.LastName = "TestLast"
	}
	if config.Username == "" {
		config.Username = "autou"
	}
	if config.Password == "" {
		config.Password = "passwd"
	}
	if config.NumChannelsToJoin == 0 {
		config.NumChannelsToJoin = 10
	}
	if config.Num == 0 {
		config.Num = 1
	}
	if config.CreateThreads == 0 {
		config.CreateThreads = 8
	}
	if config.LoginThreads == 0 {
		config.LoginThreads = 8
	}
}
