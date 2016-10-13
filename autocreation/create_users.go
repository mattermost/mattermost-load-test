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
		Users:  make([]*model.User, 0, config.NumUsers),
		Errors: make([]error, 0, config.NumUsers),
	}

	userChan := make(chan *model.User, config.NumUsers)
	errorChan := make(chan error, config.NumUsers)

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
			errorChan <- err
		} else {
			userChan <- result.Data.(*model.User)
		}
	})

	close(userChan)
	close(errorChan)

	for user := range userChan {
		userResults.Users = append(userResults.Users, user)
	}

	for err := range errorChan {
		userResults.Errors = append(userResults.Errors, err)
	}

	return userResults
}
