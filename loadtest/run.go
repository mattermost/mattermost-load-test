// Copyright (c) 2017 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information

package loadtest

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/viper"

	"github.com/mattermost/mattermost-load-test/randutil"
	"github.com/mattermost/mattermost-server/mlog"
	"github.com/mattermost/mattermost-server/model"
)

func RunTest(test *TestRun) error {
	interruptChannel := make(chan os.Signal)
	signal.Notify(interruptChannel, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	clientTimingStats := NewClientTimingStats()

	cfg := &LoadTestConfig{}
	if err := viper.Unmarshal(cfg); err != nil {
		return errors.Wrap(err, "failed to read loadtest configuration")
	}

	db := ConnectToDB(cfg.ConnectionConfiguration.DriverName, cfg.ConnectionConfiguration.DataSource)
	if db == nil {
		return fmt.Errorf("failed to connect to database")
	}

	loadtestInstance, err := NewInstance(db, cfg.UserEntitiesConfiguration.NumActiveEntities)
	if err != nil {
		return err
	}
	defer func() {
		if err := loadtestInstance.Close(); err != nil {
			mlog.Error("failed to close instance", mlog.Err(err))
		}
	}()
	mlog.Info(
		"Registered loadtest instance",
		mlog.String("instance_id", loadtestInstance.Id),
		mlog.Int("entity_start_num", loadtestInstance.EntityStartNum),
		mlog.Int64("seed", loadtestInstance.Seed),
	)

	if loadtestInstance.EntityStartNum+cfg.UserEntitiesConfiguration.NumActiveEntities > cfg.LoadtestEnviromentConfig.NumUsers {
		return fmt.Errorf(
			"Cannot start %d entities starting at %d with only %d users",
			cfg.UserEntitiesConfiguration.NumActiveEntities,
			loadtestInstance.EntityStartNum,
			cfg.LoadtestEnviromentConfig.NumUsers,
		)
	}

	mlog.Info("Setting up server.")
	serverData, err := SetupServer(cfg)
	if err != nil {
		return err
	}

	// Stop channels and wait groups, to stop and wait verious things
	// For entity monitoring routines
	stopMonitors := make(chan bool)
	var waitMonitors sync.WaitGroup
	// For entities
	stopEntity := make(chan bool)
	var waitEntity sync.WaitGroup

	// Data channels
	// Channel to recieve user entity status reports
	statusChannel := make(chan UserEntityStatusReport, 10000)
	// Channels to recieve timing information from the clients
	clientTimingChannel := make(chan TimedRoundTripperReport, 10000)
	clientTimingChannel3 := make(chan TimedRoundTripperReport, 10000)

	waitMonitors.Add(1)
	go ProcessClientRoundTripReports(clientTimingStats, clientTimingChannel3, clientTimingChannel, stopMonitors, &waitMonitors)

	adminClient := getAdminClient(cfg.ConnectionConfiguration.ServerURL, cfg.ConnectionConfiguration.AdminEmail, cfg.ConnectionConfiguration.AdminPassword, nil)
	if adminClient == nil {
		return fmt.Errorf("Unable create admin client.")
	}

	if cfg.UserEntitiesConfiguration.EnableRequestTiming {
		adminClient.HttpClient.Transport = NewTimedRoundTripper(clientTimingChannel)
	}

	mlog.Info("Logging in as users.")
	tokens := loginAsUsers(cfg, adminClient, loadtestInstance.EntityStartNum, loadtestInstance.Seed)
	if len(tokens) == 0 {
		return fmt.Errorf("Failed to login as any users")
	} else if len(tokens) != cfg.UserEntitiesConfiguration.NumActiveEntities {
		mlog.Info(fmt.Sprintf("Started only %d of %d entities", len(tokens), cfg.UserEntitiesConfiguration.NumActiveEntities))
	}

	numEntities := len(tokens)
	mlog.Info("Starting entities", mlog.Int("num_entities", numEntities), mlog.Int("entity_start_num", loadtestInstance.EntityStartNum))
	for i := 0; i < numEntities; i++ {
		entityNum := loadtestInstance.EntityStartNum + i
		entityToken := tokens[i]

		var usertype UserEntityWithRateMultiplier
		if userTypeChoice, err := randutil.WeightedChoice(test.UserEntities); err != nil {
			mlog.Error("Failed to pick user entity", mlog.Int("entity_num", entityNum))
			continue
		} else {
			usertype = userTypeChoice.Item.(UserEntityWithRateMultiplier)
		}

		mlog.Info("Starting entity", mlog.Int("entity_num", entityNum), mlog.String("entity_name", usertype.Entity.Name))

		// Create some clients
		userClient := newClientFromToken(entityToken, cfg.ConnectionConfiguration.ServerURL)
		if cfg.UserEntitiesConfiguration.EnableRequestTiming {
			userClient.HttpClient.Transport = NewTimedRoundTripper(clientTimingChannel)
		}

		// Websocket client
		websocketURL := cfg.ConnectionConfiguration.WebsocketURL
		userWebsocketClient, err := model.NewWebSocketClient4(websocketURL, entityToken)
		if err != nil {
			mlog.Error("Unable to connect websocket: " + err.Error())
		}

		// How fast to spam the server
		actionRate := time.Duration(float64(cfg.UserEntitiesConfiguration.ActionRateMilliseconds)*usertype.RateMultiplier) * time.Millisecond

		entityConfig := &EntityConfig{
			EntityNumber:        entityNum,
			EntityName:          usertype.Entity.Name,
			EntityActions:       usertype.Entity.Actions,
			UserData:            serverData.BulkloadResult.Users[entityNum],
			ChannelMap:          serverData.ChannelIdMap,
			TeamMap:             serverData.TeamIdMap,
			TownSquareMap:       serverData.TownSquareIdMap,
			AdminClient:         adminClient,
			Client:              userClient,
			WebSocketClient:     userWebsocketClient,
			ActionRate:          actionRate,
			LoadTestConfig:      cfg,
			StatusReportChannel: statusChannel,
			StopChannel:         stopEntity,
			StopWaitGroup:       &waitEntity,
			Info:                make(map[string]interface{}),
		}

		waitEntity.Add(1)
		go runEntity(entityConfig)

		waitEntity.Add(1)
		go websocketListen(entityConfig)

		if cfg.UserEntitiesConfiguration.DoStatusPolling {
			waitEntity.Add(1)
			go doStatusPolling(entityConfig)
		}

		sleepTime := actionRate / time.Duration(numEntities)

		select {
		case <-interruptChannel:
			close(stopEntity)
			return nil
		case <-time.After(sleepTime):
		}
	}

	mlog.Info("Done starting entities")

	mlog.Info(fmt.Sprintf("Test set to run for %v minutes", cfg.UserEntitiesConfiguration.TestLengthMinutes))
	timeoutchan := time.After(time.Duration(cfg.UserEntitiesConfiguration.TestLengthMinutes) * time.Minute)

	if cfg.ResultsConfiguration.PProfDelayMinutes != 0 {
		mlog.Info(fmt.Sprintf("Will run PProf after %v minutes.", cfg.ResultsConfiguration.PProfDelayMinutes))
		go func() {
			time.Sleep(time.Duration(cfg.ResultsConfiguration.PProfDelayMinutes) * time.Minute)
			mlog.Info("Running PProf.")
			RunProfile(cfg.ConnectionConfiguration.PProfURL, cfg.ResultsConfiguration.PProfLength)
		}()
	}

	select {
	case <-interruptChannel:
		mlog.Info("Interupted!")
	case <-timeoutchan:
		mlog.Info("Test finished normally")
	}
	close(stopEntity)

	mlog.Info("Waiting for user entities. Timout is 10 seconds.")
	waitWithTimeout(&waitEntity, 10*time.Second)

	mlog.Info("Stopping monitor routines. Timeout is 10 seconds.")
	close(stopMonitors)
	waitWithTimeout(&waitMonitors, 10*time.Second)

	clientTimingStats.CalcResults()
	mlog.Info("Settings", mlog.String("tag", "report"), mlog.Any("configuration", *cfg))
	mlog.Info("Timings", mlog.String("tag", "timings"), mlog.Any("timings", *clientTimingStats))

	// ioutil.WriteFile("results.txt", []byte(report), 0644)

	// files := []string{
	// 	"results.txt",
	// 	"loadtest.log",
	// }

	// if cfg.ResultsConfiguration.PProfDelayMinutes != 0 {
	// 	files = append(files, "goroutine.svg", "block.svg", "profile.svg")
	// }

	// if cfg.ResultsConfiguration.SendReportToMMServer {
	// 	mlog.Info("Sending results to mm server.")
	// 	sendResultsToMMServer(
	// 		cfg.ResultsConfiguration.ResultsServerURL,
	// 		cfg.ResultsConfiguration.ResultsUsername,
	// 		cfg.ResultsConfiguration.ResultsPassword,
	// 		cfg.ResultsConfiguration.ResultsChannelId,
	// 		cfg.ResultsConfiguration.CustomReportText,
	// 		files,
	// 	)
	// }

	mlog.Info("DONE!")

	return nil
}

func RunProfile(pprofurl string, profileLength int) {
	cmdgoroutine := exec.Command("go", "tool", "pprof", "-svg", pprofurl+"/goroutine")
	cmdblock := exec.Command("go", "tool", "pprof", "-svg", pprofurl+"/block")
	cmdprofile := exec.Command("go", "tool", "pprof", "-seconds="+strconv.Itoa(profileLength), "-svg", pprofurl+"/profile")

	datagoroutine, err := cmdgoroutine.Output()
	if err != nil {
		mlog.Error("Error running goroutine profile: " + err.Error())
	}
	ioutil.WriteFile("goroutine.svg", datagoroutine, 0644)

	datablock, err := cmdblock.Output()
	if err != nil {
		mlog.Error("Error running block profile: " + err.Error())
	}
	ioutil.WriteFile("block.svg", datablock, 0644)

	dataprofile, err := cmdprofile.Output()
	if err != nil {
		mlog.Error("Error running cpu profile: " + err.Error())
	}
	ioutil.WriteFile("profile.svg", dataprofile, 0644)
}

func sendResultsToMMServer(server, username, password, channelId, message string, attachments []string) error {
	client := model.NewAPIv4Client(server)

	user, resp := client.Login(username, password)
	if resp.Error != nil {
		return resp.Error
	}

	var fileIds []string
	if len(attachments) != 0 {
		for _, filename := range attachments {
			file, err := os.Open(filename)
			if err != nil {
				fmt.Print("Unable to find: " + filename)
				fmt.Println(" Error: " + err.Error())
				continue
			}
			data := &bytes.Buffer{}
			if _, err := io.Copy(data, file); err != nil {
				fmt.Print("Unable to copy file: " + filename)
				fmt.Println(" Error: " + err.Error())
				continue
			}
			file.Close()

			fileUploadResp, resp := client.UploadFile(data.Bytes(), channelId, filename)
			if resp.Error != nil || fileUploadResp == nil || len(fileUploadResp.FileInfos) != 1 {
				fmt.Print("Unable to upload file: " + filename)
				fmt.Println(" Error: " + resp.Error.Error())
				continue
			}

			fileIds = append(fileIds, fileUploadResp.FileInfos[0].Id)
		}
	}

	_, resp = client.CreatePost(&model.Post{
		UserId:    user.Id,
		ChannelId: channelId,
		Message:   message,
		Type:      model.POST_DEFAULT,
		FileIds:   fileIds,
	})
	if resp != nil && resp.Error != nil {
		fmt.Print("Unable to create post.")
		fmt.Println(" Error: " + resp.Error.Error())
	}

	return nil
}
