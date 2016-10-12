// Copyright (c) 2016 Spinpunch, Inc. All Rights Reserved.
// See License.txt for license information.

package autocreation

import (
	"github.com/mattermost/mattermost-load-test/loadtestconfig"
	"github.com/mattermost/platform/model"
)

type LoginUsersResult struct {
	SessionTokens []string
	Errors        []error
}

func LoginUsers(client *model.Client, config *loadtestconfig.UsersConfiguration, users []string) *LoginUsersResult {
	loginResults := &LoginUsersResult{
		SessionTokens: make([]string, len(users)),
		Errors:        make([]error, len(users)),
	}

	ThreadSplit(len(users), 8, func(i int) {
		userId := users[i]
		m := make(map[string]string)
		m["id"] = userId
		m["password"] = config.UserPassword
		r, err := client.DoApiPost("/users/login", model.MapToJson(m))
		if err != nil {
			loginResults.Errors[i] = err
		} else {
			loginResults.SessionTokens[i] = r.Header.Get(model.HEADER_TOKEN)
		}
	})

	return loginResults
}
