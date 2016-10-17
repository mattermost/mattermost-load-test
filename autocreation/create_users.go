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

func CreateUsers(client *model.Client, config *loadtestconfig.UsersConfiguration) *UsersCreationResult {
	userResults := &UsersCreationResult{
		Users:  make([]*model.User, config.NumUsers),
		Errors: make([]error, config.NumUsers),
	}

	ThreadSplit(config.NumUsers, config.CreateThreads, func(userNum int) {
		randomId := ""
		if config.UseRandomId {
			randomId = model.NewId()
		}
		user := &model.User{
			Email:     config.UserEmailPrefix + randomId + strconv.Itoa(userNum) + config.UserEmailDomain,
			FirstName: config.UserFirstName + strconv.Itoa(userNum),
			LastName:  config.UserLastName + strconv.Itoa(userNum),
			Username:  config.UserUsername + randomId + strconv.Itoa(userNum),
			Password:  config.UserPassword,
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
