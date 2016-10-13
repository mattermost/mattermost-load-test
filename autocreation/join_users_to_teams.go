// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package autocreation

import "github.com/mattermost/platform/model"

type JoinUsersResult struct {
	Errors []error
}

func JoinUsersToTeams(client *model.Client, userIds []string, teamIds []string, numThreads int) *JoinUsersResult {
	joinResult := &JoinUsersResult{
		Errors: make([]error, len(teamIds)*len(userIds)),
	}

	for _, team := range teamIds {
		ThreadSplit(len(userIds), numThreads, func(iUser int) {
			_, err := client.AddUserToTeam(team, userIds[iUser])
			if err != nil {
				joinResult.Errors[iUser] = err
			}
		})
	}

	return joinResult
}
