// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package main

import (
	"fmt"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/mattermost/platform/model"
)

const (
	ADMIN_FOR_TESTING_EMAIL    = "admin_for_testing+success@simulator.amazonses.com"
	ADMIN_FOR_TESTING_USERNAME = "admin_for_testing"
	ADMIN_FOR_TESTING_PASSWORD = "password"

	TOTAL_CHANNELS_PER_TEAM    = 100
	TOTAL_USERS                = 10000
	NUMBER_OF_CHANNELS_TO_JOIN = 10

	CHANNEL_NAME_PREFIX = "test-channel-"
)

var usersByUsername = make(map[string]*model.User)
var channelsByName = make(map[string]*model.Channel)
var channelMembers = make(map[string]map[string]bool)
var config *Config

func AppError(err *model.AppError) {
	Error(fmt.Sprintf("eid: %v\n\tmsg: %v\n\tdtl: %v", err.Id, err.Message, err.DetailedError))
}

func Error(msg string) {
	println("[ERROR] " + msg)
}

func Info(msg string) {
	println("[INFO]  " + msg)
}

func main() {

	time := model.GetMillis()

	SetupConfig()
	config = GetConfig()

	adminClient := model.NewClient(config.Connection.ServerURL)
	var adminUser *model.User

	if result, err := adminClient.GetPing(); err != nil {
		Error(fmt.Sprintf("Failed to ping server at %v", config.Connection.ServerURL))
		Error("Did you follow the setup guide and modify setup.json?")
		AppError(err)
		return
	} else {
		Info(fmt.Sprintf("Successfully pinged server at %v running version %v", config.Connection.ServerURL, result["version"]))
	}

	if result, err := adminClient.Login(config.Connection.AdminEmail, config.Connection.AdminPassword); err != nil {
		Error(fmt.Sprintf("failed to login with '%v' and '%v'", config.Connection.AdminEmail, config.Connection.AdminPassword))
		Error("Did you follow the setup guide and create an administrator?")
		Error("Please run the command")
		Error(fmt.Sprintf("'./bin/platform user create --email %v --username loadtest_admin --password %v'", config.Connection.AdminEmail, config.Connection.AdminPassword))
		AppError(err)
		return
	} else {
		adminUser = result.Data.(*model.User)
		Info(fmt.Sprintf("Successfully logged in with user %v and roles of %v", adminUser.Email, adminUser.Roles))

		if !adminUser.IsInRole(model.PERMISSIONS_SYSTEM_ADMIN) {
			Error(fmt.Sprintf("%v is not a system admin, please run the command", adminUser.Email))
			Error(fmt.Sprintf("'./bin/platform roles system_admin %v", adminUser.Username))
			return
		}
	}

	if result, err := adminClient.GetConfig(); err != nil {
		Error("Failed to get the server config")
		AppError(err)
		return
	} else {
		serverConfig := result.Data.(*model.Config)

		if !*serverConfig.TeamSettings.EnableOpenServer {
			Info("EnableOpenServer is false, attempt to set to true for the load test...")
			*serverConfig.TeamSettings.EnableOpenServer = true
			if _, err := adminClient.SaveConfig(serverConfig); err != nil {
				Error("Failed to set EnableOpenServer")
				AppError(err)
				return
			}
		}

		Info("EnableOpenServer is true")

		if serverConfig.TeamSettings.MaxUsersPerTeam < 50000 {
			Info(fmt.Sprintf("MaxUsersPerTeam is %v, attempt to set to 50000 for the load test...", serverConfig.TeamSettings.MaxUsersPerTeam))
			serverConfig.TeamSettings.MaxUsersPerTeam = 50000
			if _, err := adminClient.SaveConfig(serverConfig); err != nil {
				Error("Failed to set MaxUsersPerTeam")
				AppError(err)
				return
			}
		}

		Info(fmt.Sprintf("MaxUsersPerTeam is %v", serverConfig.TeamSettings.MaxUsersPerTeam))
	}

	if _, err := adminClient.GetByUsername(ADMIN_FOR_TESTING_USERNAME, ""); err != nil {
		Info(fmt.Sprintf("%v account appears to be missing attempt to create as system admin for running the tests...", ADMIN_FOR_TESTING_EMAIL))

		user := &model.User{
			Email:    ADMIN_FOR_TESTING_EMAIL,
			Username: ADMIN_FOR_TESTING_USERNAME,
			Password: ADMIN_FOR_TESTING_PASSWORD,
		}

		if result, err := adminClient.CreateUser(user, ""); err != nil {
			Error(fmt.Sprintln("Failed to create system admin %v for testing", ADMIN_FOR_TESTING_EMAIL))
			AppError(err)
			return
		} else {
			adminForTestingUser := result.Data.(*model.User)
			Info(fmt.Sprintf("Successfully created system admin %v for testing", adminForTestingUser.Email))

			if _, err := adminClient.UpdateUserRoles(adminForTestingUser.Id, model.PERMISSIONS_SYSTEM_ADMIN+" system_user"); err != nil {
				Error(fmt.Sprintln("Failed to assign system admin role to %v for testing", ADMIN_FOR_TESTING_EMAIL))
				AppError(err)
				return
			}
		}
	}

	adminForTestingClient := model.NewClient(config.Connection.ServerURL)
	var adminForTestingUser *model.User

	if result, err := adminForTestingClient.Login(ADMIN_FOR_TESTING_EMAIL, ADMIN_FOR_TESTING_PASSWORD); err != nil {
		Error(fmt.Sprintf("failed to login to admin testing account with '%v' and '%v'", ADMIN_FOR_TESTING_EMAIL, ADMIN_FOR_TESTING_PASSWORD))
		AppError(err)
		return
	} else {
		adminForTestingUser = result.Data.(*model.User)
		Info(fmt.Sprintf("Successfully logged in to admin testing account with user %v and roles of %v", adminForTestingUser.Email, adminForTestingUser.Roles))

		if !adminForTestingUser.IsInRole(model.PERMISSIONS_SYSTEM_ADMIN) {
			Error(fmt.Sprintf("%v is not a system admin, this shouldn't happen.", adminForTestingUser.Email))
			return
		}
	}

	var team *model.Team

	if result, err := adminForTestingClient.GetInitialLoad(); err != nil {
		AppError(err)
		return
	} else {
		initialLoad := result.Data.(*model.InitialLoad)

		if len(initialLoad.Teams) == 0 {
			t := &model.Team{
				AllowOpenInvite: true,
				DisplayName:     "Team For Load Testing 01",
				Name:            "team-for-load-testing-01",
				Email:           adminForTestingUser.Email,
				Type:            model.TEAM_OPEN,
			}

			if result, err := adminForTestingClient.CreateTeam(t); err != nil {
				Error("Failed to create team for testing")
				AppError(err)
				return
			} else {
				team = result.Data.(*model.Team)
			}
		} else if len(initialLoad.Teams) == 1 {
			team = initialLoad.Teams[0]
		} else {
			Error("Invalid number of teams")
			return
		}
	}

	Info(fmt.Sprintf("Using team %v for testing", team.Name))
	adminForTestingClient.SetTeamId(team.Id)

	if result, err := adminForTestingClient.GetChannels(""); err != nil {
		Error("Someting happened while fetching channels for a team")
		AppError(err)
		return
	} else {
		channels := result.Data.(*model.ChannelList)

		for _, channel := range *channels {
			channelsByName[channel.Name] = channel
		}
	}

	Info(fmt.Sprintf("Starting to create or verify channels..."))

	for i := 0; i < TOTAL_CHANNELS_PER_TEAM; i++ {

		newChan := &model.Channel{
			Name:        CHANNEL_NAME_PREFIX + strconv.Itoa(i),
			DisplayName: fmt.Sprintf("Channel %v", i),
			Type:        model.CHANNEL_OPEN,
		}

		if channel, ok := channelsByName[newChan.Name]; !ok {
			if result, err := adminForTestingClient.CreateChannel(newChan); err != nil {
				Error("Someting happened while creating channels for a team")
				AppError(err)
				return
			} else {
				c := result.Data.(*model.Channel)
				channelsByName[c.Name] = c
				channelMembers[c.Id] = make(map[string]bool)
			}
		} else {
			m := make(map[string]bool)
			offset := 0
			limit := 1000

			for true {
				if result, err := adminForTestingClient.GetProfilesInChannel(channel.Id, offset, limit, ""); err != nil {
					Error("Failed to get profiles for channel")
					AppError(err)
					return
				} else {
					rm := result.Data.(map[string]*model.User)

					if len(rm) == 0 {
						break
					}

					for _, user := range rm {
						m[user.Id] = true
					}
				}

				offset = offset + limit
			}

			channelMembers[channel.Id] = m
		}

		print(fmt.Sprintf("\r %v/%v", i+1, TOTAL_CHANNELS_PER_TEAM))
	}

	channelSortedByName := make([]string, 0, len(channelsByName))
	for _, channel := range channelsByName {
		if strings.Index(channel.Name, CHANNEL_NAME_PREFIX) == 0 {
			channelSortedByName = append(channelSortedByName, channel.Name)
		}
	}

	sort.Strings(channelSortedByName)

	println()
	Info(fmt.Sprintf("Finished creating or verifying channels"))

	moreUsers := 0
	fetchLimit := 1000

	Info(fmt.Sprintf("Loading existing users"))

	for true {
		if result, err := adminForTestingClient.GetProfilesInTeam(team.Id, moreUsers, fetchLimit, ""); err != nil {
			Info("Failed loading users for the team")
			AppError(err)
			return
		} else {
			users := result.Data.(map[string]*model.User)

			if len(users) == 0 {
				break
			}

			for _, user := range users {
				usersByUsername[user.Username] = user
			}

			print(".")
		}

		moreUsers = moreUsers + fetchLimit
	}

	println()

	workerThreads := runtime.GOMAXPROCS(0) * 2
	Info(fmt.Sprintf("Finished loading existing users and found %v", len(usersByUsername)))
	Info(fmt.Sprintf("Setting number of worker threads to %v", workerThreads))
	Info(fmt.Sprintf("Starting to create or verify users..."))

	offset := TOTAL_USERS / workerThreads
	counter := make(chan int)

	for i := 0; i < workerThreads; i++ {

		end := i*offset + offset

		if i == workerThreads-1 {
			end = TOTAL_USERS
		}

		Info(fmt.Sprintf("Starting threads for users %v to %v", i*offset, end))
		go createUsers(counter, i*offset, end, team, adminForTestingClient.AuthToken)
	}

	printCounter(counter)

	println()

	Info(fmt.Sprintf("Finished creating or verifying users"))

	inSeconds := (model.GetMillis() - time) / 1000
	inMinutes := inSeconds / 60
	rSeconds := inSeconds - (inMinutes * 60)

	Info(fmt.Sprintf("Finished in %v min %v sec", inMinutes, rSeconds))
}

func printCounter(counter chan int) {
	i := 0

	for {
		select {
		case _, ok := <-counter:

			if !ok {
				// An error occurred and we shutting down
				return
			}

			i = i + 1
			print(fmt.Sprintf("\r %v/%v", i, TOTAL_USERS))
			if i == TOTAL_USERS {
				return
			}

		}

	}
}

func createUsers(counter chan int, from int, to int, team *model.Team, authToken string) {

	adminForTestingClient := model.NewClient(config.Connection.ServerURL)
	adminForTestingClient.AuthToken = authToken
	adminForTestingClient.AuthType = model.HEADER_BEARER
	adminForTestingClient.SetTeamId(team.Id)

	for i := from; i < to; i++ {
		user := &model.User{
			Email:    fmt.Sprintf("autou_%v+success@simulator.amazonses.com", i),
			Username: fmt.Sprintf("autou%v", i),
			Password: "passwd",
		}

		if ruser, ok := usersByUsername[user.Username]; !ok {
			if result, err := adminForTestingClient.CreateUser(user, ""); err != nil {
				if err.Id == "store.sql_user.save.username_exists.app_error" {
					if uresult, uerr := adminForTestingClient.GetByUsername(user.Username, ""); uerr != nil {
						Error("Failed to add to team (speical case) because user was already created")
						AppError(uerr)
						close(counter)
						return
					} else {
						user = uresult.Data.(*model.User)
					}

				} else {
					Error(fmt.Sprintf("Failed to create users %v", user.Email))
					AppError(err)
					close(counter)
					return
				}
			} else {
				user = result.Data.(*model.User)
			}

			if _, err := adminForTestingClient.AddUserToTeam(team.Id, user.Id); err != nil {
				Error("Failed to add user to team")
				AppError(err)
				close(counter)
				return
			}
		} else {
			user = ruser
		}

		channelNumber := i % TOTAL_CHANNELS_PER_TEAM

		for i := 0; i < NUMBER_OF_CHANNELS_TO_JOIN; i++ {

			channelNumber = channelNumber + 1

			if channelNumber >= TOTAL_CHANNELS_PER_TEAM {
				channelNumber = 0
			}

			channel := channelsByName[CHANNEL_NAME_PREFIX+strconv.Itoa(channelNumber)]

			if !channelMembers[channel.Id][user.Id] {
				adminForTestingClient.AddChannelMember(channel.Id, user.Id)
			}
		}

		counter <- 1
	}
}
