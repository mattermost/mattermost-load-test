// Copyright (c) 2017 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information

package loadtest

import (
	"fmt"

	"time"

	"github.com/mattermost/mattermost-server/mlog"
	"github.com/mattermost/mattermost-server/model"
)

type ServerSetupData struct {
	TeamIdMap       map[string]string
	ChannelIdMap    map[string]string
	TownSquareIdMap map[string]string
	BulkloadResult  GenerateBulkloadFileResult
}

func SetupServer(cfg *LoadTestConfig) (*ServerSetupData, error) {
	if cfg.ConnectionConfiguration.WaitForServerStart {
		WaitForServer(cfg.ConnectionConfiguration.ServerURL)
	}

	var cmdrun ServerCLICommandRunner
	var cmderr error
	if cfg.ConnectionConfiguration.LocalCommands {
		mlog.Info("Connecting to local app server")
		cmdrun, cmderr = NewLocalConnection(cfg.ConnectionConfiguration.MattermostInstallDir)
	} else {
		mlog.Info("Connecting to app server over SSH")
		cmdrun, cmderr = ConnectSSH(cfg.ConnectionConfiguration.SSHHostnamePort, cfg.ConnectionConfiguration.SSHKey, cfg.ConnectionConfiguration.SSHUsername, cfg.ConnectionConfiguration.SSHPassword, cfg.ConnectionConfiguration.MattermostInstallDir, cfg.ConnectionConfiguration.ConfigFileLoc)
	}
	if cmderr != nil {
		mlog.Error("Unable to connect issue platform commands. Continuing anyway... Got error: " + cmderr.Error())
		cmdrun = nil
	} else {
		mlog.Info("Testing ability to run commands.")
		if success, output := cmdrun.RunPlatformCommand("version"); !success {
			mlog.Error("Unable to connect issue platform commands. Continuing anyway... Got Output: " + output)
			cmdrun.Close()
			cmdrun = nil
		}
	}

	if cmdrun != nil {
		defer cmdrun.Close()
	}

	adminClient := getAdminClient(cfg.ConnectionConfiguration.ServerURL, cfg.ConnectionConfiguration.AdminEmail, cfg.ConnectionConfiguration.AdminPassword, cmdrun)
	if adminClient == nil {
		return nil, fmt.Errorf("Unable create admin client.")
	}

	mlog.Info("Checking configuration parameters.")
	if err := checkConfigForLoadtests(adminClient); err != nil {
		return nil, err
	}

	mlog.Info("Generating users for loadtest.")
	bulkloadResult := GenerateBulkloadFile(&cfg.LoadtestEnviromentConfig)

	if !cfg.ConnectionConfiguration.SkipBulkload {
		if cmdrun == nil {
			return nil, fmt.Errorf("Failed to bulk import users because was unable to connect to app server to issue platform CLI commands. Please fill in SSH info and see errors above. You can also use `loadtest genbulkload` to load the users manually without having to provide SSH info.")
		}
		mlog.Info("Acquiring bulkload lock")
		if getBulkloadLock(adminClient) {
			mlog.Info("Sending loadtest file.")
			if err := cmdrun.SendLoadtestFile(&bulkloadResult.File); err != nil {
				releaseBulkloadLock(adminClient)
				return nil, err
			}
			mlog.Info("Running bulk import.")
			if success, output := cmdrun.RunPlatformCommand("import bulk --workers 64 --apply loadtestusers.json"); !success {
				releaseBulkloadLock(adminClient)
				return nil, fmt.Errorf("Failed to bulk import users: " + output)
			} else {
				mlog.Info(output)
			}
			releaseBulkloadLock(adminClient)
		}
	}

	mlog.Info("Clearing caches")
	if success, _ := adminClient.InvalidateCaches(); !success {
		mlog.Error("Could not clear caches")
	}

	teamIdMap := make(map[string]string)
	channelIdMap := make(map[string]string)
	townSquareIdMap := make(map[string]string)
	if teams, resp := adminClient.GetAllTeams("", 0, cfg.LoadtestEnviromentConfig.NumTeams+200); resp.Error != nil {
		return nil, resp.Error
	} else {
		for _, team := range teams {
			teamIdMap[team.Name] = team.Id
			numRecieved := 200
			for page := 0; numRecieved == 200; page++ {
				if channels, resp2 := adminClient.GetPublicChannelsForTeam(team.Id, page, 200, ""); resp2.Error != nil {
					mlog.Error("Could not get public channels for team", mlog.String("team_id", team.Id), mlog.Err(resp2.Error))
					return nil, resp2.Error
				} else {
					numRecieved = len(channels)
					for _, channel := range channels {
						channelIdMap[team.Name+channel.Name] = channel.Id
						if channel.Name == "town-square" {
							mlog.Info("Found town-square", mlog.String("team", team.Name))
							townSquareIdMap[team.Name] = channel.Id
						}
					}
				}
			}
		}
	}

	return &ServerSetupData{
		TeamIdMap:       teamIdMap,
		ChannelIdMap:    channelIdMap,
		TownSquareIdMap: townSquareIdMap,
		BulkloadResult:  bulkloadResult,
	}, nil
}

func checkConfigForLoadtests(adminClient *model.Client4) error {
	if serverConfig, resp := adminClient.GetConfig(); serverConfig == nil {
		mlog.Error("Failed to get the server config", mlog.Err(resp.Error))
		return resp.Error
	} else {
		if !*serverConfig.TeamSettings.EnableOpenServer {
			mlog.Info("EnableOpenServer is false, attempt to set to true for the load test...")
			*serverConfig.TeamSettings.EnableOpenServer = true
			if _, resp := adminClient.UpdateConfig(serverConfig); resp.Error != nil {
				mlog.Error("Failed to set EnableOpenServer", mlog.Err(resp.Error))
				return resp.Error
			}
		}

		mlog.Info("EnableOpenServer is true")

		if *serverConfig.TeamSettings.MaxUsersPerTeam < 50000 {
			mlog.Info("Attempting to set MaxUsersPerTeam to 50000 for the load test.", mlog.Int("old_max_users_per_team", *serverConfig.TeamSettings.MaxUsersPerTeam))
			*serverConfig.TeamSettings.MaxUsersPerTeam = 50000
			if _, resp := adminClient.UpdateConfig(serverConfig); resp.Error != nil {
				mlog.Error("Failed to set MaxUsersPerTeam", mlog.Err(resp.Error))
				return resp.Error
			}
		}

		mlog.Info(fmt.Sprintf("MaxUsersPerTeam is %v", *serverConfig.TeamSettings.MaxUsersPerTeam))

		if *serverConfig.TeamSettings.MaxChannelsPerTeam < 50000 {
			mlog.Info("Attempting to set MaxChannelsPerTeam to 50000 for the load test.", mlog.Int64("old_max_channels_per_team", *serverConfig.TeamSettings.MaxChannelsPerTeam))
			*serverConfig.TeamSettings.MaxChannelsPerTeam = 50000
			if _, resp := adminClient.UpdateConfig(serverConfig); resp.Error != nil {
				mlog.Error("Failed to set MaxChannelsPerTeam", mlog.Err(resp.Error))
				return resp.Error
			}
		}

		mlog.Info(fmt.Sprintf("MaxChannelsPerTeam is %v", *serverConfig.TeamSettings.MaxChannelsPerTeam))

		if !serverConfig.ServiceSettings.EnableIncomingWebhooks {
			mlog.Info("Enabing incoming webhooks for the load test...")
			serverConfig.ServiceSettings.EnableIncomingWebhooks = true
			if _, resp := adminClient.UpdateConfig(serverConfig); resp.Error != nil {
				mlog.Error("Failed to set EnableIncomingWebhooks", mlog.Err(resp.Error))
				return resp.Error
			}
		}

		mlog.Info("EnableIncomingWebhooks is true")

		if *serverConfig.ServiceSettings.EnableOnlyAdminIntegrations {
			mlog.Info("Disabling only admin integrations for loadtest.")
			*serverConfig.ServiceSettings.EnableOnlyAdminIntegrations = false
			if _, resp := adminClient.UpdateConfig(serverConfig); resp.Error != nil {
				mlog.Error("Failed to set EnableOnlyAdminIntegrations", mlog.Err(resp.Error))
				return resp.Error
			}
		}
		mlog.Info("EnableOnlyAdminIntegrations is false")

	}

	return nil
}

func WaitForServer(serverURL string) {
	numSuccess := 0
	waitClient := model.NewAPIv4Client(serverURL)
	for numSuccess < 5 {
		for success, resp := waitClient.GetPing(); resp.Error != nil || success != "OK"; success, resp = waitClient.GetPing() {
			numSuccess = 0
			mlog.Info("Waiting for server to be up")
			time.Sleep(5 * time.Second)
		}
		mlog.Info(fmt.Sprintf("Success %v", numSuccess))
		numSuccess++
	}
}

func getBulkloadLock(adminClient *model.Client4) bool {
	if user, resp := adminClient.GetMe(""); resp.Error != nil {
		mlog.Error("Unable to get admin user while trying to get lock 1", mlog.Err(resp.Error))
		return false
	} else if user.Nickname == "" {
		myId := model.NewId()
		user.Nickname = myId
		if _, resp := adminClient.UpdateUser(user); resp.Error != nil {
			mlog.Error("Unable to update admin user while trying to get lock 1", mlog.Err(resp.Error))
			return false
		}
		time.Sleep(2 * time.Second)
		if updatedUser, resp := adminClient.GetMe(""); resp.Error != nil {
			mlog.Error("Unable to get admin user while trying to get lock 2: %v", mlog.Err(resp.Error))
			return false
		} else if updatedUser.Nickname == myId {
			// We got the lock!
			mlog.Info("Acquired bulkload lock")
			return true
		}
	}

	// If we didn't get the lock, wait for it to clear
	for {
		time.Sleep(2 * time.Second)
		mlog.Info("Polling for lock release: " + time.Now().Format(time.UnixDate))
		if updatedUser, resp := adminClient.GetMe(""); resp.Error != nil {
			mlog.Error("Unable to get admin user while trying to wait for lock 3", mlog.Err(resp.Error))
			return false
		} else if updatedUser.Nickname == "" {
			// Lock has been released
			mlog.Info("Lock Released: " + time.Now().Format(time.UnixDate))
			return false
		}
	}
}

func releaseBulkloadLock(adminClient *model.Client4) {
	mlog.Info("Releasing bulkload lock")
	if user, resp := adminClient.GetMe(""); resp.Error != nil {
		mlog.Error("Unable to get admin user while trying to release lock. Note that system will be in a bad state. You need to change the system admin user's nickname to blank to fix things.", mlog.Err(resp.Error))
	} else if user.Nickname == "" {
		mlog.Error("Unable to get admin user while trying to get lock 1", mlog.Err(resp.Error))
	} else {
		user.Nickname = ""
		if _, resp := adminClient.UpdateUser(user); resp.Error != nil {
			mlog.Error("Unable to update admin user while trying to release lock 1", mlog.Err(resp.Error))
		}
	}
}
