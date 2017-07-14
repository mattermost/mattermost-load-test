// Copyright (c) 2017 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information

package loadtest

import (
	"fmt"

	"time"

	"github.com/mattermost/mattermost-load-test/autocreation"
	"github.com/mattermost/mattermost-load-test/cmdlog"
	"github.com/mattermost/platform/model"
)

type ServerSetupData struct {
	TeamIdMap      map[string]string
	ChannelIdMap   map[string]string
	BulkloadResult autocreation.GenerateBulkloadFileResult
}

func SetupServer(cfg *LoadTestConfig) (*ServerSetupData, error) {
	if cfg.ConnectionConfiguration.WaitForServerStart {
		waitClient := model.NewAPIv4Client(cfg.ConnectionConfiguration.ServerURL)
		for success, resp := waitClient.GetPing(); resp.Error != nil || success != "OK"; success, resp = waitClient.GetPing() {
			cmdlog.Info("Waiting for server to be up")
			time.Sleep(5 * time.Second)
		}
	}

	cmdlog.Info("Connecting to load server.")
	var cmdrun ServerCommandRunner
	var cmderr error
	if cfg.ConnectionConfiguration.LocalCommands {
		cmdrun, cmderr = NewLocalConnection(cfg.ConnectionConfiguration.MattermostInstallDir)
	} else {
		cmdrun, cmderr = ConnectSSH(cfg.ConnectionConfiguration.SSHHostnamePort, cfg.ConnectionConfiguration.SSHKey, cfg.ConnectionConfiguration.SSHUsername, cfg.ConnectionConfiguration.SSHPassword, cfg.ConnectionConfiguration.MattermostInstallDir, cfg.ConnectionConfiguration.ConfigFileLoc)
	}
	if cmderr != nil {
		return nil, cmderr
	}
	defer cmdrun.Close()

	cmdlog.Info("Testing ability to run commands.")
	if success, output := cmdrun.RunPlatformCommand("version"); !success {
		return nil, fmt.Errorf("Unable to issue platform commands. Got: " + output)
	}

	cmdlog.Info("Checking configuration parameters.")
	adminClient := getAdminClient(cfg.ConnectionConfiguration.ServerURL, cfg.ConnectionConfiguration.AdminEmail, cfg.ConnectionConfiguration.AdminPassword, cmdrun)
	if adminClient == nil {
		return nil, fmt.Errorf("Unable create admin client.")
	}
	if err := checkConfigForLoadtests(adminClient); err != nil {
		return nil, err
	}

	cmdlog.Info("Generating bulkload file.")
	bulkloadResult := autocreation.GenerateBulkloadFile(&cfg.LoadtestEnviromentConfig)
	cmdlog.Info("Sending loadtest file.")
	if err := cmdrun.SendLoadtestFile(&bulkloadResult.File); err != nil {
		return nil, err
	}

	if !cfg.ConnectionConfiguration.SkipBulkload {
		cmdlog.Info("Running bulk import.")
		if success, output := cmdrun.RunPlatformCommand("import bulk --workers 64 --apply loadtestusers.json"); !success {
			return nil, fmt.Errorf("Failed to bulk import users: " + output)
		} else {
			cmdlog.Info(output)
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

		if serverConfig.TeamSettings.MaxUsersPerTeam < 50000 {
			cmdlog.Infof("MaxUsersPerTeam is %v, attempt to set to 50000 for the load test...", serverConfig.TeamSettings.MaxUsersPerTeam)
			serverConfig.TeamSettings.MaxUsersPerTeam = 50000
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
