// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package autocreation

import (
	"strconv"

	"github.com/mattermost/mattermost-load-test/loadtestconfig"
	"github.com/mattermost/platform/model"
)

type UsersCreationResult struct {
	Users  []*model.User
	Errors []error
}

func CreateUsers(client *model.Client, config *loadtestconfig.UserCreationConfiguration) *UsersCreationResult {
	userResults := &UsersCreationResult{
		Users:  make([]*model.User, config.Num),
		Errors: make([]error, config.Num),
	}

	ThreadSplit(config.Num, config.CreateThreads, func(userNum int) {
		randomId := ""
		if config.UseRandomId {
			randomId = model.NewId()
		}
		user := &model.User{
			Email:     config.EmailPrefix + randomId + strconv.Itoa(userNum) + config.EmailDomain,
			FirstName: config.FirstName + strconv.Itoa(userNum),
			LastName:  config.LastName + strconv.Itoa(userNum),
			Username:  config.Username + randomId + strconv.Itoa(userNum),
			Password:  config.Password,
		}

		result, err := client.CreateUser(user, "")
		if err != nil {
			userResults.Errors[userNum] = err
		} else {
			userResults.Users[userNum] = result.Data.(*model.User)
		}
	})

	return userResults
}
