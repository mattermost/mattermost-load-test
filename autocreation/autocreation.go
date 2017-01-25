// Copyright (c) 2017 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package autocreation

import (
	"fmt"
	"runtime"
	"strconv"

	"sync"

	"github.com/mattermost/mattermost-load-test/cmd/cmdlog"
	"github.com/mattermost/mattermost-load-test/loadtestconfig"
	"github.com/mattermost/platform/model"
)

const (
	ADMIN_FOR_TESTING_EMAIL    = "admin_for_loadtests+success@simulator.amazonses.com"
	ADMIN_FOR_TESTING_USERNAME = "admin_for_loadtests"
	ADMIN_FOR_TESTING_PASSWORD = "password"

	CHANNEL_NAME_PREFIX = "lt-channel-"
)

func DoAutocreation(config loadtestconfig.LoadTestConfig) {
	// Login as the user supplied in config
	adminClient := getAdminClient(config.ServerURL, config.AdminEmail, config.AdminPassword)
	if adminClient == nil {
		return
	}

	// Check configuration to make sure it is good for loadtesting
	checkConfigForLoadtests(adminClient)

	// Login as a user created specificly for loadtesting
	// This allows the user to login as their user without having it joined to every channel
	adminForTestingClient := getAdminForTestingClient(adminClient)
	if adminForTestingClient == nil {
		return
	}

	// Create or get testing teams
	teams := createOrGetTestingTeams(adminForTestingClient)
	team := teams[0]
	cmdlog.Infof("Using team %v for testing", team)
	adminForTestingClient.SetTeamId(team.Id)

	// Create or get channels
	cmdlog.Info("Starting to create or verify channels...")
	channels := getOrCreateChannels(adminForTestingClient, team, config.NumChannelsPerTeam)

	// Create or get users
	cmdlog.Infof("Starting to create or verify users...")
	users := getOrCreateUsers(adminForTestingClient, config.NumUsers)

	// Login as users to get tokens

	// Join users to team if nessisary
	if err := verifyOrCreateTeamMembers(adminForTestingClient, users, team); err != nil {
		return
	}

	// Join users to channels if nessiary
}

func printCounter(counter chan int, total int) {
	i := 0
	for {
		select {
		case _, ok := <-counter:

			if !ok {
				// An error occurred and we shutting down
				return
			}

			i = i + 1
			print(fmt.Sprintf("\r %v/%v", i, total))
			if i == total {
				return
			}

		}

	}
}

func fetchExistingTeamMembers(client *model.Client, team *model.Team) map[string]*model.User {
	usersByUsername := make(map[string]*model.User)
	moreUsers := 0
	fetchLimit := 1000

	cmdlog.Infof("Loading existing users")

	for {
		if result, err := client.GetTeamMembers(team.Id, moreUsers, fetchLimit, ""); err != nil {
			cmdlog.Info("Failed loading existing team members for team:", team.Name)
			cmdlog.AppError(err)
			return nil
		} else {
			users := result.Data.(map[string]*model.TeamMember)

			if len(users) == 0 {
				break
			}

			for _, user := range users {
				usersByUsername[user.Username] = user
			}
		}

		moreUsers = moreUsers + fetchLimit
	}

	return usersByUsername
}

func verifyOrCreateTeamMembers(client *model.Client, usersByUsername map[string]*model.User, team *model.Team) error {
	client.GetTeamMembers(team.Id, 

	for _, user := range usersByUsername {
		if _, err := client.AddUserToTeam(team.Id, user.Id); err != nil {
			cmdlog.Error("Failed to add user to team")
			cmdlog.AppError(err)
			return err
		}
	}

	return nil
}

func getOrCreateUsers(client *model.Client, numUsers int) map[string]*model.User {
	workerThreads := runtime.GOMAXPROCS(0) * 2
	cmdlog.Infof("Setting number of worker threads to %v", workerThreads)
	resultUsers := make(map[string]*model.User, numUsers)
	var resultUsersLock sync.Mutex

	usersByUsername := fetchExistingUsers(client)
	cmdlog.Infof("Finished loading existing users and found %v", len(usersByUsername))

	ThreadSplit(numUsers, workerThreads, printCounter, func(usernum int) {
		user := &model.User{
			Email:    fmt.Sprintf("autou_%v+success@simulator.amazonses.com", usernum),
			Username: fmt.Sprintf("autou%v", usernum),
			Password: "passwd",
		}

		if ruser, ok := usersByUsername[user.Username]; !ok {
			if result, err := client.CreateUser(user, ""); err != nil {
				if err.Id == "store.sql_user.save.username_exists.app_error" {
					if uresult, uerr := client.GetByUsername(user.Username, ""); uerr != nil {
						cmdlog.Error("Failed to create user (speical case) because user was already created")
						cmdlog.AppError(uerr)
						return
					} else {
						cmdlog.Debug("User was missed on first pass:", user.Username)
						user = uresult.Data.(*model.User)
					}
				} else {
					cmdlog.Errorf("Failed to create user %v", user.Email)
					cmdlog.AppError(err)
					return
				}
			} else {
				user = result.Data.(*model.User)
			}

		} else {
			user = ruser
		}

		resultUsersLock.Lock()
		resultUsers[user.Username] = user
		resultUsersLock.Unlock()
	})

	cmdlog.Infof("Finished creating or verifying users")

	return resultUsers
}

/*
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

*/

func fetchExistingUsers(client *model.Client) map[string]*model.User {
	usersByUsername := make(map[string]*model.User)
	moreUsers := 0
	fetchLimit := 1000

	cmdlog.Infof("Loading existing users")

	for {
		if result, err := client.GetProfiles(moreUsers, fetchLimit, ""); err != nil {
			cmdlog.Info("Failed loading existing users.")
			cmdlog.AppError(err)
			return nil
		} else {
			users := result.Data.(map[string]*model.User)

			if len(users) == 0 {
				break
			}

			for _, user := range users {
				usersByUsername[user.Username] = user
			}
		}

		moreUsers = moreUsers + fetchLimit
	}

	return usersByUsername
}

/*func GetOrCreateChannelMembers(client *model.Client) {
	channelMembers := make(map[string]map[string]bool, numChannels)
	channelMembers[c.Id] = make(map[string]bool)
	m := make(map[string]bool)
	offset := 0
	limit := 1000

	for {
		if result, err := client.GetProfilesInChannel(channel.Id, offset, limit, ""); err != nil {
			cmdlog.Error("Failed to get profiles for channel")
			cmdlog.AppError(err)
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
}*/

func getOrCreateChannels(client *model.Client, team *model.Team, numChannels int) map[string]*model.Channel {
	channelsByName := make(map[string]*model.Channel, numChannels)

	client.SetTeamId(team.Id)

	// Retrieve all existing channels
	if result, err := client.GetChannels(""); err != nil {
		cmdlog.Error("Someting happened while fetching channels for a team")
		cmdlog.AppError(err)
		return nil
	} else {
		channels := result.Data.(*model.ChannelList)

		for _, channel := range *channels {
			channelsByName[channel.Name] = channel
		}
	}

	// Verify every channel exists, if not, create it.
	for channelNumber := 0; channelNumber < numChannels; channelNumber++ {
		newChan := &model.Channel{
			Name:        CHANNEL_NAME_PREFIX + strconv.Itoa(channelNumber),
			DisplayName: fmt.Sprintf("LT Channel %v", channelNumber),
			Type:        model.CHANNEL_OPEN,
		}
		if _, ok := channelsByName[newChan.Name]; !ok {
			if result, err := client.CreateChannel(newChan); err != nil {
				cmdlog.Error("Someting happened while creating channels for a team")
				cmdlog.AppError(err)
				return nil
			} else {
				c := result.Data.(*model.Channel)
				channelsByName[c.Name] = c
			}
		}

		cmdlog.Infof("\rChannels Created: %v/%v", channelNumber+1, numChannels)
	}

	return channelsByName
}

func createOrGetTestingTeams(adminForTestingClient *model.Client) []*model.Team {
	var team *model.Team
	if result, err := adminForTestingClient.GetInitialLoad(); err != nil {
		cmdlog.AppError(err)
		return nil
	} else {
		initialLoad := result.Data.(*model.InitialLoad)

		if len(initialLoad.Teams) == 0 {
			t := &model.Team{
				AllowOpenInvite: true,
				DisplayName:     "Team For Load Testing 01",
				Name:            "team-for-load-testing-01",
				Email:           ADMIN_FOR_TESTING_EMAIL,
				Type:            model.TEAM_OPEN,
			}

			if result, err := adminForTestingClient.CreateTeam(t); err != nil {
				cmdlog.Error("Failed to create team for testing")
				cmdlog.AppError(err)
				return nil
			} else {
				team = result.Data.(*model.Team)
			}
		} else if len(initialLoad.Teams) == 1 {
			team = initialLoad.Teams[0]
		} else {
			cmdlog.Error("Invalid number of teams")
			return nil
		}
	}

	// Only one team for now
	teams := make([]*model.Team, 0, 1)
	teams = append(teams, team)

	return teams
}

func getAdminForTestingClient(adminClient *model.Client) *model.Client {
	// Make sure testing account exists
	if _, err := adminClient.GetByUsername(ADMIN_FOR_TESTING_USERNAME, ""); err != nil {
		cmdlog.Infof("%v account appears to be missing attempt to create as system admin for running the tests...", ADMIN_FOR_TESTING_EMAIL)

		user := &model.User{
			Email:    ADMIN_FOR_TESTING_EMAIL,
			Username: ADMIN_FOR_TESTING_USERNAME,
			Password: ADMIN_FOR_TESTING_PASSWORD,
		}

		if result, err := adminClient.CreateUser(user, ""); err != nil {
			cmdlog.Errorf("Failed to create system admin %v for testing", ADMIN_FOR_TESTING_EMAIL)
			cmdlog.AppError(err)
			return nil
		} else {
			adminForTestingUser := result.Data.(*model.User)
			cmdlog.Infof("Successfully created system admin %v for testing", adminForTestingUser.Email)

			if _, err := adminClient.UpdateUserRoles(adminForTestingUser.Id, model.PERMISSIONS_SYSTEM_ADMIN+" system_user"); err != nil {
				cmdlog.Errorf("Failed to assign system admin role to %v for testing", ADMIN_FOR_TESTING_EMAIL)
				cmdlog.AppError(err)
				return nil
			}
		}
	}

	adminForTestingClient := model.NewClient(adminClient.Url)

	if result, err := adminForTestingClient.Login(ADMIN_FOR_TESTING_EMAIL, ADMIN_FOR_TESTING_PASSWORD); err != nil {
		cmdlog.Errorf("failed to login to admin testing account with '%v' and '%v'", ADMIN_FOR_TESTING_EMAIL, ADMIN_FOR_TESTING_PASSWORD)
		cmdlog.AppError(err)
		return nil
	} else {
		adminForTestingUser := result.Data.(*model.User)
		cmdlog.Infof("Successfully logged in to admin testing account with user %v and roles of %v", adminForTestingUser.Email, adminForTestingUser.Roles)

		if !adminForTestingUser.IsInRole(model.PERMISSIONS_SYSTEM_ADMIN) {
			cmdlog.Errorf("%v is not a system admin, this shouldn't happen.", adminForTestingUser.Email)
			return nil
		}
	}

	return adminForTestingClient
}

func checkConfigForLoadtests(adminClient *model.Client) error {
	if result, err := adminClient.GetConfig(); err != nil {
		cmdlog.Error("Failed to get the server config")
		cmdlog.AppError(err)
		return err
	} else {
		serverConfig := result.Data.(*model.Config)

		if !*serverConfig.TeamSettings.EnableOpenServer {
			cmdlog.Info("EnableOpenServer is false, attempt to set to true for the load test...")
			*serverConfig.TeamSettings.EnableOpenServer = true
			if _, err := adminClient.SaveConfig(serverConfig); err != nil {
				cmdlog.Error("Failed to set EnableOpenServer")
				cmdlog.AppError(err)
				return err
			}
		}

		cmdlog.Info("EnableOpenServer is true")

		if serverConfig.TeamSettings.MaxUsersPerTeam < 50000 {
			cmdlog.Infof("MaxUsersPerTeam is %v, attempt to set to 50000 for the load test...", serverConfig.TeamSettings.MaxUsersPerTeam)
			serverConfig.TeamSettings.MaxUsersPerTeam = 50000
			if _, err := adminClient.SaveConfig(serverConfig); err != nil {
				cmdlog.Error("Failed to set MaxUsersPerTeam")
				cmdlog.AppError(err)
				return err
			}
		}

		cmdlog.Infof("MaxUsersPerTeam is %v", serverConfig.TeamSettings.MaxUsersPerTeam)
	}

	return nil
}

func getAdminClient(serverURL string, adminEmail string, adminPass string) *model.Client {
	client := model.NewClient(serverURL)

	if result, err := client.GetPing(); err != nil {
		cmdlog.Errorf("Failed to ping server at %v", serverURL)
		cmdlog.Error("Did you follow the setup guide and modify loadtestconfig.json?")
		cmdlog.AppError(err)
		return nil
	} else {
		cmdlog.Infof("Successfully pinged server at %v running version %v", serverURL, result["version"])
	}

	if result, err := client.Login(adminEmail, adminPass); err != nil {
		cmdlog.Errorf("failed to login with '%v' and '%v'", adminEmail, adminPass)
		cmdlog.Error("Did you follow the setup guide and create an administrator?")
		cmdlog.Error("Please run the command")
		cmdlog.Errorf("'./bin/platform user create --email %v --username loadtest_admin --password %v'", adminEmail, adminPass)
		cmdlog.AppError(err)
		return nil
	} else {
		adminUser := result.Data.(*model.User)
		cmdlog.Infof("Successfully logged in with user %v and roles of %v", adminUser.Email, adminUser.Roles)

		if !adminUser.IsInRole(model.PERMISSIONS_SYSTEM_ADMIN) {
			cmdlog.Errorf("%v is not a system admin, please run the command", adminUser.Email)
			cmdlog.Errorf("'./bin/platform roles system_admin %v", adminUser.Username)
			return nil
		}
	}

	return client
}
