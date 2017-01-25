// Copyright (c) 2017 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package testlib

import (
	"fmt"
	"os"

	"github.com/mattermost/platform/api"
	"github.com/mattermost/platform/app"
	"github.com/mattermost/platform/model"
	"github.com/mattermost/platform/utils"
)

func changeToPlatformDir() {
	vendorDir := "vendor/"
	found := false
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(vendorDir); err == nil {
			found = true
			break
		}
		vendorDir = "../" + vendorDir
	}

	if found {
		fmt.Println(vendorDir)
		os.Chdir(vendorDir + "github.com/mattermost/platform")
	}
}

func RunMattermostServer() {
	changeToPlatformDir()

	utils.TranslationsPreInit()
	utils.LoadConfig("config.json")
	utils.InitTranslations(utils.Cfg.LocalizationSettings)
	utils.Cfg.TeamSettings.MaxUsersPerTeam = 50
	*utils.Cfg.RateLimitSettings.Enable = false
	utils.Cfg.EmailSettings.SendEmailNotifications = true
	utils.Cfg.EmailSettings.SMTPServer = "dockerhost"
	utils.Cfg.EmailSettings.SMTPPort = "2500"
	utils.Cfg.EmailSettings.FeedbackEmail = "test@example.com"
	utils.DisableDebugLogForTest()
	app.NewServer()
	app.InitStores()
	api.InitRouter()
	app.StartServer()
	api.InitApi()
	utils.EnableDebugLogForTest()
	app.Srv.Store.MarkSystemRanUnitTests()

	*utils.Cfg.TeamSettings.EnableOpenServer = true

	if _, err := app.GetUserByEmail("test@test.com"); err != nil {
		user := &model.User{
			Email:    "test@test.com",
			Username: "tester",
			Password: "passwd",
		}
		app.CreateUser(user)
	}

}
