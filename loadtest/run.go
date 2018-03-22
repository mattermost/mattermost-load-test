// Copyright (c) 2017 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information

package loadtest

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"sync"
	"syscall"

	"time"

	"github.com/gizak/termui"
	"github.com/mattermost/mattermost-load-test/cmdlog"
	"github.com/mattermost/mattermost-server/model"
)

func RunTest(test *TestRun) error {
	defer cmdlog.CloseLog()

	cfg, err := GetConfig()
	if err != nil {
		return fmt.Errorf("Unable to find configuration file: " + err.Error())
	}

	clientTimingStats := NewClientTimingStats()

	strchanbe := cmdlog.GetStringChannelBackend()
	logbuf := NewUIBuffer(10)
	go func() {
		for {
			select {
			case str := <-strchanbe:
				logbuf.Add(str)
			}
		}
	}()

	if cfg.DisplayConfiguration.ShowUI {
		go CreateLoadtestUI(clientTimingStats, logbuf)
	} else if cfg.DisplayConfiguration.LogToConsole {
		cmdlog.SetConsoleLog()
	}

	cmdlog.Info("Setting up server.")
	serverData, err := SetupServer(cfg)
	if err != nil {
		return err
	}

	cmdlog.Info("Logging in as users.")
	tokens := loginAsUsers(cfg)
	if len(tokens) == 0 {
		return fmt.Errorf("Failed to login as any users")
	} else if len(tokens) != cfg.UserEntitiesConfiguration.NumActiveEntities {
		cmdlog.Infof("Started only %d of %d entities", len(tokens), cfg.UserEntitiesConfiguration.NumActiveEntities)
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

	numEntities := len(tokens)
	entityNum := 0
	entitiesToSkip := cfg.UserEntitiesConfiguration.EntityStartNum
	for _, usertype := range test.UserEntities {
		numEntitesToCreateForType := int(math.Floor((float64(usertype.Freq) / 100.0) * float64(numEntities)))
		startEntity := 0
		if numEntitesToCreateForType <= entitiesToSkip {
			entitiesToSkip -= numEntitesToCreateForType
			continue
		} else {
			startEntity = entitiesToSkip
			entitiesToSkip = 0
		}
		cmdlog.Infof("Starting %d entities ", strconv.Itoa(numEntitesToCreateForType))
		for i := startEntity; i < numEntitesToCreateForType; i++ {
			cmdlog.Infof("Starting entity %v", entityNum)
			// Get the user auth token for this entity.
			entityToken := tokens[entityNum]

			// Create some clients
			userClient := newClientFromToken(entityToken, cfg.ConnectionConfiguration.ServerURL)
			if cfg.UserEntitiesConfiguration.EnableRequestTiming {
				userClient.HttpClient.Transport = NewTimedRoundTripper(clientTimingChannel)
			}

			// Websocket client
			websocketURL := cfg.ConnectionConfiguration.WebsocketURL
			userWebsocketClient, err := model.NewWebSocketClient4(websocketURL, entityToken)
			if err != nil {
				cmdlog.Error("Unable to connect websocket: " + err.Error())
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
			time.Sleep(sleepTime)

			entityNum++
		}
	}

	cmdlog.Info("Done starting entities")

	interrupChannel := make(chan os.Signal)
	signal.Notify(interrupChannel, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	cmdlog.Infof("Test set to run for %v minutes", cfg.UserEntitiesConfiguration.TestLengthMinutes)
	timeoutchan := time.After(time.Duration(cfg.UserEntitiesConfiguration.TestLengthMinutes) * time.Minute)

	if cfg.ResultsConfiguration.PProfDelayMinutes != 0 {
		cmdlog.Infof("Will run PProf after %v minutes.", cfg.ResultsConfiguration.PProfDelayMinutes)
		go func() {
			time.Sleep(time.Duration(cfg.ResultsConfiguration.PProfDelayMinutes) * time.Minute)
			cmdlog.Info("Running PProf.")
			RunProfile(cfg.ConnectionConfiguration.PProfURL, cfg.ResultsConfiguration.PProfLength)
		}()
	}

	select {
	case <-interrupChannel:
		cmdlog.Info("Interupted!")
	case <-timeoutchan:
		cmdlog.Info("Test finished normally")
	}
	close(stopEntity)
	termui.StopLoop()

	cmdlog.Info("Waiting for user entities. Timout is 10 seconds.")
	waitWithTimeout(&waitEntity, 10*time.Second)

	cmdlog.Info("Stopping monitor routines. Timeout is 10 seconds.")
	close(stopMonitors)
	waitWithTimeout(&waitMonitors, 10*time.Second)

	report := "\n"
	report += "--------------------------------------\n"
	report += "-------- Loadtest Results ------------\n"
	report += "--------------------------------------\n"
	timingsReport := clientTimingStats.PrintReport()
	configReport := cfg.PrintReport()

	report += configReport
	report += timingsReport

	cmdlog.Info(report)
	ioutil.WriteFile("results.txt", []byte(report), 0644)

	files := []string{
		"results.txt",
		"loadtest.log",
	}

	if cfg.ResultsConfiguration.PProfDelayMinutes != 0 {
		files = append(files, "goroutine.svg", "block.svg", "profile.svg")
	}

	if cfg.ResultsConfiguration.SendReportToMMServer {
		cmdlog.Info("Sending results to mm server.")
		sendResultsToMMServer(
			cfg.ResultsConfiguration.ResultsServerURL,
			cfg.ResultsConfiguration.ResultsUsername,
			cfg.ResultsConfiguration.ResultsPassword,
			cfg.ResultsConfiguration.ResultsChannelId,
			cfg.ResultsConfiguration.CustomReportText,
			files,
		)
	}

	cmdlog.Info("DONE!")

	return nil
}

func RunProfile(pprofurl string, profileLength int) {
	cmdgoroutine := exec.Command("go", "tool", "pprof", "-svg", pprofurl+"/goroutine")
	cmdblock := exec.Command("go", "tool", "pprof", "-svg", pprofurl+"/block")
	cmdprofile := exec.Command("go", "tool", "pprof", "-seconds="+strconv.Itoa(profileLength), "-svg", pprofurl+"/profile")

	datagoroutine, err := cmdgoroutine.Output()
	if err != nil {
		cmdlog.Error("Error running goroutine profile: " + err.Error())
	}
	ioutil.WriteFile("goroutine.svg", datagoroutine, 0644)

	datablock, err := cmdblock.Output()
	if err != nil {
		cmdlog.Error("Error running block profile: " + err.Error())
	}
	ioutil.WriteFile("block.svg", datablock, 0644)

	dataprofile, err := cmdprofile.Output()
	if err != nil {
		cmdlog.Error("Error running cpu profile: " + err.Error())
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
