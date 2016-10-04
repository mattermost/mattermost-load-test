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

	for userNum := 1; userNum <= config.NumUsers; userNum++ {
		randomId := ""
		if config.UseRandomId {
			randomId = model.NewId()
		}
		user := &model.User{
			Email:     config.UserEmailPrefix + randomId + strconv.Itoa(userNum) + config.UserEmailDomain,
			FirstName: config.UserFirstName + strconv.Itoa(userNum),
			LastName:  config.UserLastName + strconv.Itoa(userNum),
			Password:  config.UserPassword,
		}

		result, err := client.CreateUser(user, "")
		if err != nil {
			userResults.Errors = append(userResults.Errors, err)
		} else {
			userResults.Users = append(userResults.Users, result.Data.(*model.User))
		}
	}

	return userResults
}
