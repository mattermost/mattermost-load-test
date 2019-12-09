// Copyright (c) 2017 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information

package loadtest

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
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
	"github.com/mattermost/mattermost-server/v5/mlog"
	"github.com/mattermost/mattermost-server/v5/model"
)

func RunTest(test *TestRun) error {
	r := rand.New(rand.NewSource(0))

	interruptChannel := make(chan os.Signal, 1)
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
	mlog.Info("Settings", mlog.String("tag", "report"), mlog.Any("configuration", *cfg), mlog.String("instance_id", loadtestInstance.Id))

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

	// Stop channels and wait groups, to stop and wait various things
	// For entity monitoring routines
	var waitMonitors sync.WaitGroup
	// For entities
	stopEntity := make(chan bool)
	var waitEntity sync.WaitGroup

	// Data channels
	// Channel to receive user entity status reports
	statusChannel := make(chan UserEntityStatusReport, 10000)
	// Channels to receive timing information from the clients
	clientTimingChannel := make(chan TimedRoundTripperReport, 10000)

	waitMonitors.Add(1)
	go func() {
		defer waitMonitors.Done()

		for timingReport := range clientTimingChannel {
			clientTimingStats.AddTimingReport(timingReport)

			if clientTimingStats.CountResults() > 100 {
				mlog.Info("Timings", mlog.String("tag", "timings"), mlog.Any("timings", *clientTimingStats), mlog.String("instance_id", loadtestInstance.Id))
				clientTimingStats.Reset()
			}
		}

		if clientTimingStats.CountResults() > 0 {
			mlog.Info("Timings", mlog.String("tag", "timings"), mlog.Any("timings", *clientTimingStats), mlog.String("instance_id", loadtestInstance.Id))
			clientTimingStats.Reset()
		}
	}()

	// Mirror http.DefaultTransport to start
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          cfg.ConnectionConfiguration.MaxIdleConns,
		MaxIdleConnsPerHost:   cfg.ConnectionConfiguration.MaxIdleConnsPerHost,
		IdleConnTimeout:       time.Duration(cfg.ConnectionConfiguration.IdleConnTimeoutMilliseconds) * time.Millisecond,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	httpClient := &http.Client{Transport: transport}

	adminClient := getAdminClient(httpClient, cfg.ConnectionConfiguration.ServerURL, cfg.ConnectionConfiguration.AdminEmail, cfg.ConnectionConfiguration.AdminPassword, nil)
	if adminClient == nil {
		return fmt.Errorf("Unable create admin client.")
	}

	adminClient.HttpClient.Transport = NewTimedRoundTripper(clientTimingChannel)

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
		if userTypeChoice, err := randutil.WeightedChoice(r, test.UserEntities); err != nil {
			mlog.Error("Failed to pick user entity", mlog.Int("entity_num", entityNum))
			continue
		} else {
			usertype = userTypeChoice.Item.(UserEntityWithRateMultiplier)
		}

		mlog.Info("Starting entity", mlog.Int("entity_num", entityNum), mlog.String("entity_name", usertype.Entity.Name))

		// Create some clients
		userClient := newClientFromToken(httpClient, entityToken, cfg.ConnectionConfiguration.ServerURL)
		userClient.HttpClient.Transport = NewTimedRoundTripper(clientTimingChannel)

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
			Users:               serverData.BulkloadResult.Users,
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
			r:                   rand.New(rand.NewSource(time.Now().UnixNano())),
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
			mlog.Info("Running PProf", mlog.String("url", cfg.ConnectionConfiguration.PProfURL), mlog.Int("duration_s", cfg.ResultsConfiguration.PProfLength))
			RunProfile(cfg.ConnectionConfiguration.PProfURL, cfg.ResultsConfiguration.PProfLength)
		}()
	}

	select {
	case <-interruptChannel:
		mlog.Info("Interrupted!")
	case <-timeoutchan:
		mlog.Info("Test finished normally")
	}
	close(stopEntity)

	mlog.Info("Waiting for user entities. Timout is 10 seconds.")
	waitWithTimeout(&waitEntity, 10*time.Second)

	mlog.Info("Stopping monitor routines. Timeout is 10 seconds.")
	close(clientTimingChannel)
	waitWithTimeout(&waitMonitors, 10*time.Second)

	mlog.Info("Finished loadtest")

	return nil
}

func goCmd(args ...string) ([]byte, error) {
	cmd := exec.Command("go", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, errors.Wrapf(err, "cmd.Run(go %v) failed: %s", args, string(stderr.Bytes()))
	}

	return stdout.Bytes(), nil
}

func pprofSvg(pprofUrl, svgFilename string, otherArgs ...string) error {
	args := []string{"tool", "pprof"}
	args = append(args, otherArgs...)
	args = append(args, "-svg", pprofUrl)

	svgBytes, err := goCmd(args...)
	if err != nil {
		return errors.Wrap(err, "pprof svg failed")
	}

	return ioutil.WriteFile(svgFilename, svgBytes, 0644)
}

func RunProfile(pprofUrl string, profileLength int) {
	if err := pprofSvg(pprofUrl+"/goroutine", "goroutine.svg"); err != nil {
		mlog.Error("Error running goroutine profile: " + err.Error())
	}

	if err := pprofSvg(pprofUrl+"/block", "block.svg"); err != nil {
		mlog.Error("Error running block profile: " + err.Error())
	}

	if err := pprofSvg(pprofUrl+"/profile", "profile.svg", "-seconds="+strconv.Itoa(profileLength)); err != nil {
		mlog.Error("Error running block profile: " + err.Error())
	}
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
				mlog.Error("Unable to find", mlog.String("filename", filename), mlog.Err(err))
				continue
			}
			data := &bytes.Buffer{}
			if _, err := io.Copy(data, file); err != nil {
				mlog.Error("Unable to copy file", mlog.String("filename", filename), mlog.Err(err))
				continue
			}
			_ = file.Close()

			fileUploadResp, resp := client.UploadFile(data.Bytes(), channelId, filename)
			if resp.Error != nil || fileUploadResp == nil || len(fileUploadResp.FileInfos) != 1 {
				mlog.Error("Unable to upload file", mlog.String("filename", filename), mlog.Err(resp.Error))
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
		mlog.Error("Unable to create post", mlog.Err(resp.Error))
	}

	return nil
}
