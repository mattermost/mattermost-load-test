// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package loadtestconfig

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
)

type ServerStateUser struct {
	Id           string
	SessionToken string
}

type ServerStateTeam struct {
	Id string
}

type ServerStateChannel struct {
	Id     string
	TeamId string
}

type ServerState struct {
	Users    []ServerStateUser
	Teams    []ServerStateTeam
	Channels []ServerStateChannel
}

func ServerStateFromJson(data io.Reader) *ServerState {
	decoder := json.NewDecoder(data)
	var out ServerState
	if err := decoder.Decode(&out); err != nil {
		return nil
	}
	return &out
}

func ServerStateFromStdin() *ServerState {
	stat, err := os.Stdin.Stat()
	if err == nil && ((stat.Mode() & os.ModeCharDevice) == 0) {
		reader := bufio.NewReader(os.Stdin)
		return ServerStateFromJson(reader)
	}

	return &ServerState{}
}

func (state *ServerState) ToJson() string {
	if result, err := json.Marshal(state); err != nil {
		return ""
	} else {
		return string(result)
	}
}

func (state *ServerState) GetUserIds() []string {
	users := make([]string, 0, len(state.Users))
	for _, user := range state.Users {
		users = append(users, user.Id)
	}

	return users
}

func (state *ServerState) GetTeamIds() []string {
	teamIds := make([]string, 0, len(state.Teams))
	for _, team := range state.Teams {
		teamIds = append(teamIds, team.Id)
	}

	return teamIds
}

func (state *ServerState) GetChannelIds() []string {
	channelIds := make([]string, 0, len(state.Channels))
	for _, channel := range state.Channels {
		channelIds = append(channelIds, channel.Id)
	}

	return channelIds
}
