// Copyright (c) 2017 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information

package loadtest

import (
	"bufio"
	"bytes"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/mlog"
	"github.com/mattermost/mattermost-server/model"
)

type ServerSetupData struct {
	// TeamIdMap maps team name to team id
	TeamIdMap map[string]string
	// ChannelIdMap maps team name and channel name to channel id
	ChannelIdMap map[string]map[string]string
	// TownSquareIdMap maps team name to the channel id of the corresponding default channel.
	TownSquareIdMap map[string]string

	BulkloadResult GenerateBulkloadFileResult
}

func SetupServer(cfg *LoadTestConfig) (*ServerSetupData, error) {
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
		mlog.Error("Unable to connect issue mattermost commands. Continuing anyway... Got error: " + cmderr.Error())
		cmdrun = nil
	} else {
		mlog.Info("Testing ability to run commands.")
		if success, output := cmdrun.RunPlatformCommand("version"); !success {
			mlog.Error("Unable to connect issue mattermost commands. Continuing anyway... Got Output: " + output)
			cmdrun.Close()
			cmdrun = nil
		}
	}

	if cmdrun != nil {
		defer cmdrun.Close()
	}

	adminClient := getAdminClient(&http.Client{}, cfg.ConnectionConfiguration.ServerURL, cfg.ConnectionConfiguration.AdminEmail, cfg.ConnectionConfiguration.AdminPassword, cmdrun)
	if adminClient == nil {
		return nil, fmt.Errorf("Unable create admin client.")
	}

	mlog.Info("Checking configuration parameters.")
	if err := checkConfigForLoadtests(adminClient); err != nil {
		return nil, err
	}

	if cfg.LoadtestEnviromentConfig.NumPlugins > 0 {
		mlog.Info("Setting up plugins.")
		if cfg.LoadtestEnviromentConfig.NumPlugins > 1 {
			mlog.Error("Bulk-loading supports at most one plugin deployment at this time")
		}

		plugin, err := os.Open("testfiles/com.mattermost.sample-plugin-webapp-only.tar.gz")
		if err != nil {
			return nil, err
		}

		if _, resp := adminClient.UploadPlugin(plugin); resp.Error != nil {
			return nil, resp.Error
		}
		if _, resp := adminClient.EnablePlugin("com.mattermost.sample-plugin"); resp.Error != nil {
			return nil, resp.Error
		}
	}

	mlog.Info("Generating users for loadtest.")
	bulkloadResult := GenerateBulkloadFile(&cfg.LoadtestEnviromentConfig)

	if !cfg.ConnectionConfiguration.SkipBulkload {
		if cmdrun == nil {
			return nil, fmt.Errorf("Failed to bulk import users because was unable to connect to app server to issue mattermost CLI commands. Please fill in SSH info and see errors above. You can also use `loadtest genbulkload` to load the users manually without having to provide SSH info.")
		}
		mlog.Info("Acquiring bulkload lock")
		if getBulkloadLock(adminClient) {
			defer releaseBulkloadLock(adminClient)
			mlog.Info("Sending loadtest file.")

			// sendChunk sends and loads a subset of the whole bulkload file
			sendChunk := func(chunk *bytes.Buffer, line int) error {
				mlog.Info("Running bulk import on chunk", mlog.Int("line", line))
				if err := cmdrun.SendLoadtestFile(chunk); err != nil {
					return err
				}
				if success, output := cmdrun.RunPlatformCommand("import bulk --workers 64 --apply loadtestusers.json"); !success {
					return fmt.Errorf("Failed to bulk import users: " + output)
				} else {
					mlog.Info(output)
				}

				return nil
			}

			// Chunk the bulkload file into 1000 line segments. This gives visibility
			// into the long bulkloading process without requiring server changes, and
			// theoretically allows us to resume a bulkloading in the future.
			// None of this would be necessary if bulkloading was more performant.
			var chunk bytes.Buffer
			lineScanner := bufio.NewScanner(&bulkloadResult.File)
			line := 1
			var versionLine []byte
			for lineScanner.Scan() {
				if line == 1 {
					// Save the version line for repeated use
					versionLine = make([]byte, len(lineScanner.Bytes()))
					copy(versionLine, lineScanner.Bytes())
				}

				chunk.Write(lineScanner.Bytes())
				chunk.WriteString("\n")
				line++

				if line%1000 == 0 {
					if err := sendChunk(&chunk, line); err != nil {
						return nil, err
					}

					chunk.Reset()
					chunk.Write(versionLine)
					chunk.WriteString("\n")
				}
			}
			if err := lineScanner.Err(); err != nil {
				return nil, errors.Wrapf(err, "failed to scan bulkload result at line %d", line)
			}

			if err := sendChunk(&chunk, line); err != nil {
				return nil, errors.Wrap(err, "failed to send bulkload result chunk")
			}
		}
	}

	mlog.Info("Clearing caches")
	if success, _ := adminClient.InvalidateCaches(); !success {
		mlog.Error("Could not clear caches")
	}

	teamIdMap := make(map[string]string)
	channelIdMap := make(map[string]map[string]string)
	townSquareIdMap := make(map[string]string)
	teams, resp := adminClient.GetAllTeams("", 0, cfg.LoadtestEnviromentConfig.NumTeams+200)
	if resp.Error != nil {
		return nil, resp.Error
	}

	mlog.Info("Found teams", mlog.Int("teams", len(teams)))
	for _, team := range teams {
		channelIdMap[team.Name] = make(map[string]string)

		teamIdMap[team.Name] = team.Id
		numReceived := 200
		for page := 0; numReceived == 200; page++ {
			channels, resp2 := adminClient.GetPublicChannelsForTeam(team.Id, page, 200, "")
			if resp2.Error != nil {
				mlog.Error("Could not get public channels for team", mlog.String("team_id", team.Id), mlog.Err(resp2.Error))
				return nil, resp2.Error
			}

			numReceived = len(channels)
			for _, channel := range channels {
				channelIdMap[team.Name][channel.Name] = channel.Id
				if channel.Name == "town-square" {
					mlog.Info("Found town-square", mlog.String("team", team.Name))
					townSquareIdMap[team.Name] = channel.Id
				}
			}
		}

		mlog.Info("Found team channels", mlog.String("team", team.Name), mlog.Int("channels", len(channelIdMap[team.Name])))
	}

	return &ServerSetupData{
		TeamIdMap:       teamIdMap,
		ChannelIdMap:    channelIdMap,
		TownSquareIdMap: townSquareIdMap,
		BulkloadResult:  bulkloadResult,
	}, nil
}

func checkConfigForLoadtests(adminClient *model.Client4) error {
	serverConfig, resp := adminClient.GetConfig()
	if serverConfig == nil {
		mlog.Error("Failed to get the server config", mlog.Err(resp.Error))
		return resp.Error
	}

	if !*serverConfig.EmailSettings.EnableSignInWithEmail {
		err := fmt.Errorf("EnableSignInWithEmail is disabled on app server. Cannot continue")
		mlog.Error("Failed to get the server config", mlog.Err(err))
		return err
	}

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

	if !*serverConfig.ServiceSettings.EnableIncomingWebhooks {
		mlog.Info("Enabing incoming webhooks for the load test...")
		*serverConfig.ServiceSettings.EnableIncomingWebhooks = true
		if _, resp := adminClient.UpdateConfig(serverConfig); resp.Error != nil {
			mlog.Error("Failed to set EnableIncomingWebhooks", mlog.Err(resp.Error))
			return resp.Error
		}
	}

	mlog.Info("EnableIncomingWebhooks is true")

	if *serverConfig.ServiceSettings.DEPRECATED_DO_NOT_USE_EnableOnlyAdminIntegrations {
		mlog.Info("Disabling only admin integrations for loadtest.")
		*serverConfig.ServiceSettings.DEPRECATED_DO_NOT_USE_EnableOnlyAdminIntegrations = false
		if _, resp := adminClient.UpdateConfig(serverConfig); resp.Error != nil {
			mlog.Error("Failed to set EnableOnlyAdminIntegrations", mlog.Err(resp.Error))
			return resp.Error
		}
	}
	mlog.Info("EnableOnlyAdminIntegrations is false")

	if !*serverConfig.PluginSettings.Enable {
		mlog.Info("Enabling plugins for loadtest.")
		*serverConfig.PluginSettings.Enable = true
		if _, resp := adminClient.UpdateConfig(serverConfig); resp.Error != nil {
			mlog.Error("Failed to set PluginSettings.Enable", mlog.Err(resp.Error))
			return resp.Error
		}
	}
	mlog.Info("PluginSettings.Enable is true")

	if !*serverConfig.PluginSettings.EnableUploads {
		mlog.Warn("Cannot enable plugin uploads via API. Must manually enable to test with plugins.")
	}
	mlog.Info("PluginSettings.EnableUploads is true")

	return nil
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
			return true
		}
	}
}

func releaseBulkloadLock(adminClient *model.Client4) {
	mlog.Info("Releasing bulkload lock")
	if user, resp := adminClient.GetMe(""); resp.Error != nil {
		mlog.Error("Unable to get admin user while trying to release lock. Note that system will be in a bad state. You need to change the system admin user's nickname to blank to fix things.", mlog.Err(resp.Error))
	} else if user.Nickname == "" {
		mlog.Warn("Bulkload lock was already released")
	} else {
		user.Nickname = ""
		if _, resp := adminClient.UpdateUser(user); resp.Error != nil {
			mlog.Error("Unable to update admin user while trying to release lock 1", mlog.Err(resp.Error))
		}
	}
}
