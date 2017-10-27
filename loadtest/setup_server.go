// Copyright (c) 2017 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information

package loadtest

import (
	"fmt"

	"time"

	"github.com/mattermost/mattermost-load-test/cmdlog"
	"github.com/mattermost/mattermost-server/model"
)

type ServerSetupData struct {
	TeamIdMap      map[string]string
	ChannelIdMap   map[string]string
	BulkloadResult GenerateBulkloadFileResult
}

func SetupServer(cfg *LoadTestConfig) (*ServerSetupData, error) {
	if cfg.ConnectionConfiguration.WaitForServerStart {
		WaitForServer(cfg.ConnectionConfiguration.ServerURL)
	}

	var cmdrun ServerCLICommandRunner
	var cmderr error
	if cfg.ConnectionConfiguration.LocalCommands {
		cmdlog.Info("Connecting to local app server")
		cmdrun, cmderr = NewLocalConnection(cfg.ConnectionConfiguration.MattermostInstallDir)
	} else {
		cmdlog.Info("Connecting to app server over SSH")
		cmdrun, cmderr = ConnectSSH(cfg.ConnectionConfiguration.SSHHostnamePort, cfg.ConnectionConfiguration.SSHKey, cfg.ConnectionConfiguration.SSHUsername, cfg.ConnectionConfiguration.SSHPassword, cfg.ConnectionConfiguration.MattermostInstallDir, cfg.ConnectionConfiguration.ConfigFileLoc)
	}
	if cmderr != nil {
		cmdlog.Error("Unable to connect issue platform commands. Continuing anyway... Got error: " + cmderr.Error())
		cmdrun = nil
	} else {
		cmdlog.Info("Testing ability to run commands.")
		if success, output := cmdrun.RunPlatformCommand("version"); !success {
			cmdlog.Error("Unable to connect issue platform commands. Continuing anyway... Got Output: " + output)
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

	cmdlog.Info("Checking configuration parameters.")
	if err := checkConfigForLoadtests(adminClient); err != nil {
		return nil, err
	}

	cmdlog.Info("Generating users for loadtest.")
	bulkloadResult := GenerateBulkloadFile(&cfg.LoadtestEnviromentConfig)

	if !cfg.ConnectionConfiguration.SkipBulkload {
		if cmdrun == nil {
			return nil, fmt.Errorf("Failed to bulk import users because was unable to connect to app server to issue platform CLI commands. Please fill in SSH info and see errors above. You can also use `loadtest genbulkload` to load the users manually without having to provide SSH info.")
		}
		cmdlog.Info("Aquiring bulkload lock")
		if getBulkloadLock(adminClient) {
			cmdlog.Info("Aquired bulkload lock")
			cmdlog.Info("Sending loadtest file.")
			if err := cmdrun.SendLoadtestFile(&bulkloadResult.File); err != nil {
				return nil, err
			}
			cmdlog.Info("Running bulk import.")
			if success, output := cmdrun.RunPlatformCommand("import bulk --workers 64 --apply loadtestusers.json"); !success {
				return nil, fmt.Errorf("Failed to bulk import users: " + output)
			} else {
				cmdlog.Info(output)
			}
			cmdlog.Info("Releasing bulkload lock")
			releaseBulkloadLock(adminClient)
		}
	}

	cmdlog.Info("Clearing caches")
	if success, _ := adminClient.InvalidateCaches(); !success {
		cmdlog.Error("Could not clear caches")
	}

	teamIdMap := make(map[string]string)
	channelIdMap := make(map[string]string)
	if teams, resp := adminClient.GetAllTeams("", 0, cfg.LoadtestEnviromentConfig.NumTeams+200); resp.Error != nil {
		return nil, resp.Error
	} else {
		for _, team := range teams {
			teamIdMap[team.Name] = team.Id
			numRecieved := 200
			for page := 0; numRecieved == 200; page++ {
				if channels, resp2 := adminClient.GetPublicChannelsForTeam(team.Id, page, 200, ""); resp2.Error != nil {
					cmdlog.Errorf("Could not get public channels for team %v. Error: %v", team.Id, resp2.Error.Error())
					return nil, resp2.Error
				} else {
					numRecieved = len(channels)
					for _, channel := range channels {
						channelIdMap[team.Name+channel.Name] = channel.Id
					}
				}
			}
		}
	}

	return &ServerSetupData{
		TeamIdMap:      teamIdMap,
		ChannelIdMap:   channelIdMap,
		BulkloadResult: bulkloadResult,
	}, nil
}

func checkConfigForLoadtests(adminClient *model.Client4) error {
	if serverConfig, resp := adminClient.GetConfig(); serverConfig == nil {
		cmdlog.Error("Failed to get the server config")
		cmdlog.AppError(resp.Error)
		return resp.Error
	} else {
		if !*serverConfig.TeamSettings.EnableOpenServer {
			cmdlog.Info("EnableOpenServer is false, attempt to set to true for the load test...")
			*serverConfig.TeamSettings.EnableOpenServer = true
			if _, resp := adminClient.UpdateConfig(serverConfig); resp.Error != nil {
				cmdlog.Error("Failed to set EnableOpenServer")
				cmdlog.AppError(resp.Error)
				return resp.Error
			}
		}

		cmdlog.Info("EnableOpenServer is true")

		if *serverConfig.TeamSettings.MaxUsersPerTeam < 50000 {
			cmdlog.Infof("MaxUsersPerTeam is %v, attempt to set to 50000 for the load test...", serverConfig.TeamSettings.MaxUsersPerTeam)
			*serverConfig.TeamSettings.MaxUsersPerTeam = 50000
			if _, resp := adminClient.UpdateConfig(serverConfig); resp.Error != nil {
				cmdlog.Error("Failed to set MaxUsersPerTeam")
				cmdlog.AppError(resp.Error)
				return resp.Error
			}
		}

		cmdlog.Infof("MaxUsersPerTeam is %v", serverConfig.TeamSettings.MaxUsersPerTeam)

		if *serverConfig.TeamSettings.MaxChannelsPerTeam < 50000 {
			cmdlog.Infof("MaxChannelsPerTeam is %v, attempt to set to 50000 for the load test...", *serverConfig.TeamSettings.MaxChannelsPerTeam)
			*serverConfig.TeamSettings.MaxChannelsPerTeam = 50000
			if _, resp := adminClient.UpdateConfig(serverConfig); resp.Error != nil {
				cmdlog.Error("Failed to set MaxChannelsPerTeam")
				cmdlog.AppError(resp.Error)
				return resp.Error
			}
		}

		cmdlog.Infof("MaxChannelsPerTeam is %v", *serverConfig.TeamSettings.MaxChannelsPerTeam)

		if !serverConfig.ServiceSettings.EnableIncomingWebhooks {
			cmdlog.Info("Enabing incoming webhooks for the load test...")
			serverConfig.ServiceSettings.EnableIncomingWebhooks = true
			if _, resp := adminClient.UpdateConfig(serverConfig); resp.Error != nil {
				cmdlog.Error("Failed to set EnableIncomingWebhooks")
				cmdlog.AppError(resp.Error)
				return resp.Error
			}
		}

		cmdlog.Info("EnableIncomingWebhooks is true")

		if *serverConfig.ServiceSettings.EnableOnlyAdminIntegrations {
			cmdlog.Info("Disabling only admin integrations for loadtest.")
			*serverConfig.ServiceSettings.EnableOnlyAdminIntegrations = false
			if _, resp := adminClient.UpdateConfig(serverConfig); resp.Error != nil {
				cmdlog.Error("Failed to set EnableOnlyAdminIntegrations")
				cmdlog.AppError(resp.Error)
				return resp.Error
			}
		}
		cmdlog.Info("EnableOnlyAdminIntegrations is false")

	}

	return nil
}

func WaitForServer(serverURL string) {
	numSuccess := 0
	waitClient := model.NewAPIv4Client(serverURL)
	for numSuccess < 5 {
		for success, resp := waitClient.GetPing(); resp.Error != nil || success != "OK"; success, resp = waitClient.GetPing() {
			numSuccess = 0
			cmdlog.Info("Waiting for server to be up")
			time.Sleep(5 * time.Second)
		}
		cmdlog.Infof("Success %v", numSuccess)
		numSuccess++
	}
}

func getBulkloadLock(adminClient *model.Client4) bool {
	if user, resp := adminClient.GetMe(""); resp.Error != nil {
		cmdlog.Errorf("Unable to get admin user while trying to get lock 1: %v", resp.Error.Error())
		return false
	} else if user.Nickname == "" {
		myId := model.NewId()
		user.Nickname = myId
		if _, resp := adminClient.UpdateUser(user); resp.Error != nil {
			cmdlog.Errorf("Unable to update admin user while trying to get lock 1: %v", resp.Error.Error())
			return false
		}
		time.Sleep(2 * time.Second)
		if updatedUser, resp := adminClient.GetMe(""); resp.Error != nil {
			cmdlog.Errorf("Unable to get admin user while trying to get lock 2: %v", resp.Error.Error())
			return false
		} else if updatedUser.Nickname == myId {
			// We got the lock!
			return true
		}
	}

	// If we didn't get the lock, wait for it to clear
	for {
		time.Sleep(2 * time.Second)
		cmdlog.Info("Polling for lock release: " + time.Now().Format(time.UnixDate))
		if updatedUser, resp := adminClient.GetMe(""); resp.Error != nil {
			cmdlog.Errorf("Unable to get admin user while trying to wait for lock 3: %v", resp.Error.Error())
			return false
		} else if updatedUser.Nickname == "" {
			// Lock has been released
			cmdlog.Info("Lock Released: " + time.Now().Format(time.UnixDate))
			return false
		}
	}
}

func releaseBulkloadLock(adminClient *model.Client4) {
	if user, resp := adminClient.GetMe(""); resp.Error != nil {
		cmdlog.Errorf("Unable to get admin user while trying to release lock. Note that system will be in a bad state. You need to change the system admin user's nickname to blank to fix things. Error: %v", resp.Error.Error())
	} else if user.Nickname == "" {
		cmdlog.Errorf("Unable to get admin user while trying to get lock 1: %v", resp.Error.Error())
	} else {
		user.Nickname = ""
		if _, resp := adminClient.UpdateUser(user); resp.Error != nil {
			cmdlog.Errorf("Unable to update admin user while trying to release lock 1: %v", resp.Error.Error())
		}
	}
}
