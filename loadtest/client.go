// Copyright (c) 2017 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information

package loadtest

import (
	"fmt"
	"math/rand"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/mattermost/mattermost-server/mlog"
	"github.com/mattermost/mattermost-server/model"
)

func newClientFromToken(httpClient *http.Client, token string, serverUrl string) *model.Client4 {
	// Lifted from model.NewAPIv4Client
	return &model.Client4{
		Url:        serverUrl,
		ApiUrl:     serverUrl + model.API_URL_SUFFIX,
		HttpClient: httpClient,
		AuthToken:  token,
		AuthType:   model.HEADER_BEARER,
	}
}

func loginAsUsers(cfg *LoadTestConfig, adminClient *model.Client4, entityStartNum int, seed int64) []string {
	tokens := make([]string, cfg.UserEntitiesConfiguration.NumActiveEntities)
	r := rand.New(rand.NewSource(seed))
	order := r.Perm(cfg.LoadtestEnviromentConfig.NumUsers)

	ThreadSplit(cfg.UserEntitiesConfiguration.NumActiveEntities, runtime.GOMAXPROCS(0)*2, PrintCounter, func(i int) {
		// Add the usernum to start from
		entityNum := i + entityStartNum
		userNum := entityNum
		client := model.NewAPIv4Client(cfg.ConnectionConfiguration.ServerURL)

		// Random selection if picked.
		if cfg.UserEntitiesConfiguration.RandomizeEntitySelection {
			userNum = order[entityNum]
		}

		email := "success+user" + strconv.Itoa(userNum) + "@simulator.amazonses.com"

		if user, response := adminClient.GetUserByEmail(email, ""); response.Error != nil {
			mlog.Error("Failed to find user by email", mlog.String("email", email), mlog.Err(response.Error))
		} else if ok, response := adminClient.UpdateUserActive(user.Id, true); !ok {
			mlog.Error("Failed to activate user", mlog.String("user_id", user.Id), mlog.Err(response.Error))
		} else if _, response := client.Login(email, "Loadtestpassword1"); response != nil && response.Error != nil {
			mlog.Error("Entity %v failed to login as user", mlog.Int("entity_num", entityNum), mlog.String("email", email), mlog.Err(response.Error))
		} else {
			mlog.Info("Entity has logged in", mlog.Int("entity_num", entityNum), mlog.String("email", email))
			tokens[i] = client.AuthToken
		}
	})

	activeTokens := make([]string, 0, cfg.UserEntitiesConfiguration.NumActiveEntities)
	for _, token := range tokens {
		if token != "" {
			activeTokens = append(activeTokens, token)
		}
	}

	return activeTokens
}

func getAdminClient(httpClient *http.Client, serverURL string, adminEmail string, adminPass string, cmdrun ServerCLICommandRunner) *model.Client4 {
	// Lifted from model.NewAPIv4Client
	client := &model.Client4{
		Url:        serverURL,
		ApiUrl:     serverURL + model.API_URL_SUFFIX,
		HttpClient: httpClient,
	}

	if success, resp := client.GetPing(); resp.Error != nil || success != "OK" {
		mlog.Error(fmt.Sprintf("Failed to ping server at %v", serverURL))
		if success != "" {
			mlog.Error(fmt.Sprintf("Got %v from ping", success))
		}
		mlog.Error("Did you follow the setup guide and modify loadtestconfig.json?", mlog.Err(resp.Error))
		return nil
	} else {
		mlog.Info("Successfully pinged server", mlog.String("server_url", serverURL))
	}

	var adminUser *model.User
	if user, _ := client.Login(adminEmail, adminPass); user == nil {
		mlog.Info("Failed to login as admin user.")
		if cmdrun == nil {
			mlog.Error("Unable to create admin user because was not able to connect to app server. Please create the admin user manually or fill in SSH information.")
			mlog.Error(fmt.Sprintf("Command to create admin user: ./bin/platform user create --email %v --password %v --system_admin --username ltadmin", adminEmail, adminPass))
			return nil
		}
		mlog.Info("Attempting to create admin user.")
		if success, output := cmdrun.RunPlatformCommand(fmt.Sprintf("user create --email %v --password %v --system_admin --username ltadmin", adminEmail, adminPass)); !success {
			mlog.Error("Failed to create admin user", mlog.String("output", output))
		}
		if success, output := cmdrun.RunPlatformCommand(fmt.Sprintf("user verify ltadmin")); !success {
			mlog.Error("Failed to verify email of admin user.", mlog.String("output", output))
		}
		time.Sleep(time.Second)
		if user2, resp2 := client.Login(adminEmail, adminPass); user2 == nil {
			mlog.Error("Failed to login to admin account.", mlog.Err(resp2.Error))
			return nil
		} else {
			adminUser = user2
		}
	} else {
		adminUser = user
	}

	mlog.Info("Successfully logged in", mlog.String("email", adminUser.Email), mlog.String("roles", adminUser.Roles))

	if !adminUser.IsInRole(model.PERMISSIONS_SYSTEM_ADMIN) {
		mlog.Error(fmt.Sprintf("%v is not a system admin, please run the command", adminUser.Email))
		mlog.Error(fmt.Sprintf("'./bin/platform roles system_admin %v", adminUser.Username))
		return nil
	}

	// Wait here because somtimes we are too fast in making our first request
	time.Sleep(time.Second)

	return client
}
