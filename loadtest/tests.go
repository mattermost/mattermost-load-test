// Copyright (c) 2017 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package loadtest

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strconv"

	"bytes"

	"github.com/icrowley/fake"
	"github.com/mattermost/mattermost-load-test/cmdlog"
	"github.com/mattermost/mattermost-load-test/randutil"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/utils"
)

type TestRun struct {
	UserEntities []UserEntityFrequency
}

type UserEntityFrequency struct {
	Freq           float64
	RateMultiplier float64
	Entity         UserEntity
}

type UserEntity struct {
	Name    string
	Actions []randutil.Choice
}

type EntityActions interface {
	Init(c *EntityConfig)
	Action(c *EntityConfig)
}

func readTestFile(name string) ([]byte, error) {
	path, _ := utils.FindDir("testfiles")
	file, err := os.Open(path + "/" + name)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data := &bytes.Buffer{}
	if _, err := io.Copy(data, file); err != nil {
		return nil, err
	} else {
		return data.Bytes(), nil
	}
}

func readRandomTestFile() ([]byte, error, string) {
	path, _ := utils.FindDir("testfiles")
	files, err := ioutil.ReadDir(path)
	if err != nil {
		panic("Can't read testfiles directory.")
	}

	fileI := rand.Intn(len(files))
	file := files[fileI]
	for file.IsDir() {
		fileI = rand.Intn(len(files))
		file = files[fileI]
	}

	b, err := readTestFile(file.Name())
	return b, err, file.Name()
}

func actionGetStatuses(c *EntityConfig) {
	idsI, ok := c.Info["statusUserIds"+c.UserData.Username]
	var ids []string
	if !ok {
		team, channel := c.UserData.PickTeamChannel()
		if team == nil || channel == nil {
			return
		}
		channelId := c.ChannelMap[team.Name+channel.Name]

		if channelId == "" {
			cmdlog.Error("Unable to get channel from map")
			return
		}

		members, resp := c.Client.GetChannelMembers(channelId, 0, 60, "")
		if resp.Error != nil {
			cmdlog.Errorf("Unable to get members for channel %v to seed action get status. Error: %v", channelId, resp.Error.Error())
		}

		ids = make([]string, len(*members), len(*members))
		for i := 0; i < len(*members); i++ {
			ids[i] = (*members)[i].UserId
		}

		c.Info["statusUserIds"+c.UserData.Username] = ids
	} else {
		ids = idsI.([]string)
	}

	if _, resp := c.Client.GetUsersStatusesByIds(ids); resp.Error != nil {
		cmdlog.Error("Unable to get user statuses by Ids. Error: " + resp.Error.Error())
	}
}

func actionPost(c *EntityConfig) {
	team, channel := c.UserData.PickTeamChannel()
	if team == nil || channel == nil {
		return
	}
	channelId := c.ChannelMap[team.Name+channel.Name]

	if channelId == "" {
		cmdlog.Error("Unable to get channel from map")
		return
	}

	post := &model.Post{
		ChannelId: channelId,
		Message:   fake.Sentences(),
	}

	if rand.Float64() < c.LoadTestConfig.UserEntitiesConfiguration.UploadImageChance {
		numFiles := rand.Intn(3) + 1
		fileIds := make([]string, numFiles, numFiles)
		for i := 0; i < numFiles; i++ {
			if data, err, filename := readRandomTestFile(); err != nil {
				cmdlog.Errorf("Problem reading test file. Error %v", err.Error())
			} else {
				if file, resp := c.Client.UploadFile(data, channelId, filename); resp.Error != nil {
					cmdlog.Error("Unable to upload file. Error: " + resp.Error.Error())
					return
				} else {
					fileIds[i] = file.FileInfos[0].Id
				}
			}
		}
		post.FileIds = fileIds
	}

	_, resp := c.Client.CreatePost(post)
	if resp.Error != nil {
		cmdlog.Infof("Failed to post to team %v on channel %v as user %v with token %v. Error: %v", team.Name, channel.Name, c.UserData.Username, c.Client.AuthToken, resp.Error.Error())
	}
}

func actionJoinLeaveChannel(c *EntityConfig) {
	// figure out my user id
	var userId string
	if user, resp := c.Client.GetUserByEmail(c.UserData.Email, ""); resp.Error != nil {
		cmdlog.Errorf("Unable to get user by email. User: %v, Error: %v", c.UserData.Username, resp.Error.Error())
		return
	} else {
		userId = user.Id
	}

	// pick a random team to join a channel from
	team, _ := c.UserData.PickTeamChannel()
	if team == nil {
		cmdlog.Errorf("Unable to pick random team. User: %v", c.UserData.Username)
		return
	}
	teamId := c.TeamMap[team.Name]

	// get all the public channels on that team
	var channelIdsInTeam []string
	if channels, resp := c.Client.GetPublicChannelsForTeam(teamId, 0, 1000, ""); resp.Error != nil {
		cmdlog.Errorf("Unable to get public channels for team. Team id: %v, User: %v, Error: %v", teamId, c.UserData.Username, resp.Error.Error())
		return
	} else {
		channelIdsInTeam = make([]string, 0)
		for _, channel := range channels {
			channelIdsInTeam = append(channelIdsInTeam, channel.Id)
		}
	}

	// get channels the user is already a member of on that team
	var channelId string
	if channelMembers, resp := c.Client.GetChannelMembersForUser(userId, teamId, ""); resp.Error != nil {
		cmdlog.Errorf("Unable to get channel members for user. User: %v, User: %v, Error: %v", userId, c.UserData.Username, resp.Error.Error())
		return
	} else {
		// find the first channel on the team that the user isn't already a member of, and use it for the test
		userIsAlreadyInChannel := false
		for _, potentialChannelId := range channelIdsInTeam {
			for _, channelMember := range *channelMembers {
				if channelMember.ChannelId == potentialChannelId {
					userIsAlreadyInChannel = true
					break
				}
			}
			if !userIsAlreadyInChannel {
				channelId = potentialChannelId
				break
			}
		}
	}

	if channelId == "" {
		cmdlog.Errorf("Unable to pick an open channel to join. Team id: %v, User: %v", teamId, c.UserData.Username)
		return
	} else {
		cmdlog.Infof("User %v is joining/leaving channel %v", userId, channelId)
	}

	// join and then immediately leave the channel - this exercises the ChannelMemberHistory table
	if _, resp := c.Client.AddChannelMember(channelId, userId); resp.Error != nil {
		cmdlog.Errorf("Unable to join channel. Channel: %v, User: %v, Error: %v", channelId, c.UserData.Username, resp.Error.Error())
	}
	if _, resp := c.Client.RemoveUserFromChannel(channelId, userId); resp.Error != nil {
		cmdlog.Errorf("Unable to leave channel. Channel: %v, User: %v, Error: %v", channelId, c.UserData.Username, resp.Error.Error())
	}
}

func actionGetChannel(c *EntityConfig) {
	team, channel := c.UserData.PickTeamChannel()
	if team == nil || channel == nil {
		return
	}
	channelId := c.ChannelMap[team.Name+channel.Name]

	if _, resp := c.Client.ViewChannel("me", &model.ChannelView{
		ChannelId:     channelId,
		PrevChannelId: "",
	}); resp.Error != nil {
		cmdlog.Errorf("Unable to view channel. Channel: %v, User: %v", channelId, c.UserData.Username)
	}

	if _, resp := c.Client.GetChannelMember(channelId, "me", ""); resp.Error != nil {
		cmdlog.Errorf("Unable to get channel member. Channel: %v, User: %v, Error: %v", channelId, c.UserData.Username, resp.Error.Error())
	}

	if _, resp := c.Client.GetChannelMembers(channelId, 0, 60, ""); resp.Error != nil {
		cmdlog.Errorf("Unable to get channel member. Channel: %v, User: %v, Error: %v", channelId, c.UserData.Username, resp.Error.Error())
	}

	if _, resp := c.Client.GetChannelStats(channelId, ""); resp.Error != nil {
		cmdlog.Errorf("Unable to get channel stats. Channel: %v, User: %v, Error: %v", channelId, c.UserData.Username, resp.Error.Error())
	}

	if posts, resp := c.Client.GetPostsForChannel(channelId, 0, 60, ""); resp.Error != nil {
		cmdlog.Errorf("Unable to get posts for channel Channel: %v, User: %v, Error: %v", channelId, c.UserData.Username, resp.Error.Error())
	} else {
		for _, post := range posts.Posts {
			if post.HasReactions {
				if _, resp := c.Client.GetReactions(post.Id); resp.Error != nil {
					cmdlog.Errorf("Unable to get reactions for post. Channel: %v, User: %v, Post: %v, Error: %v", channelId, c.UserData.Username, post.Id, resp.Error.Error())
				}
			}
			if len(post.FileIds) > 0 {
				if files, resp := c.Client.GetFileInfosForPost(post.Id, ""); resp.Error != nil {
					cmdlog.Errorf("Unable to get file infos for post. Channel: %v, User: %v, Post: %v, Error: %v", channelId, c.UserData.Username, post.Id, resp.Error.Error())
				} else {
					for _, file := range files {
						if file.IsImage() {
							if _, resp := c.Client.GetFileThumbnail(file.Id); resp.Error != nil {
								cmdlog.Errorf("Unable to get file thumbnail for file. Channel: %v, User: %v, Post: %v, File: %v, Error: %v", channelId, c.UserData.Username, post.Id, file.Id, resp.Error.Error())
							}
						}
					}
				}
			}
		}
	}
}

func actionPerformSearch(c *EntityConfig) {
	team, _ := c.UserData.PickTeamChannel()
	if team == nil {
		return
	}
	teamId := c.TeamMap[team.Name]

	_, resp := c.Client.SearchPosts(teamId, fake.Words(), false)
	if resp.Error != nil {
		cmdlog.Errorf("Failed to search: %v", resp.Error.Error())
	}
}

func actionDisconnectWebsocket(c *EntityConfig) {
	c.WebSocketClient.Close()
}

func actionPostWebhook(c *EntityConfig) {
	infokey := "webhookid" + strconv.Itoa(c.EntityNumber)
	hookIdI, ok := c.Info[infokey]
	hookId := ""
	if !ok {
		team, channel := c.UserData.PickTeamChannel()
		if team == nil || channel == nil {
			return
		}
		channelId := c.ChannelMap[team.Name+channel.Name]

		webhook, resp := c.Client.CreateIncomingWebhook(&model.IncomingWebhook{
			ChannelId:   channelId,
			DisplayName: model.NewId(),
			Description: model.NewId(),
		})
		if resp.Error != nil {
			cmdlog.Error("Unable to create incoming webhook. Error: " + resp.Error.Error())
			return
		}
		c.Info[infokey] = webhook.Id
		hookId = webhook.Id
	} else {
		hookId = hookIdI.(string)
	}

	webhookRequest := &model.IncomingWebhookRequest{
		Text:     fake.Paragraphs(),
		Username: "ltwhuser",
		Type:     "",
	}
	b, err := json.Marshal(webhookRequest)
	if err != nil {
		cmdlog.Error("Unable to marshal json for webhook request")
		return
	}

	var buf bytes.Buffer
	buf.WriteString(string(b))

	if resp, err := http.Post(c.LoadTestConfig.ConnectionConfiguration.ServerURL+"/hooks/"+hookId, "application/json", &buf); err != nil {
		cmdlog.Error("Failed to post by webhook. Error: " + err.Error())
	} else if resp != nil {
		resp.Body.Close()
	}
}

var posterEntity = UserEntity{
	Name: "Poster",
	Actions: []randutil.Choice{
		{
			Item:   actionPost,
			Weight: 1,
		},
	},
}

var TestBasicPosting = TestRun{
	UserEntities: []UserEntityFrequency{
		{
			Freq:           100.0,
			RateMultiplier: 1.0,
			Entity:         posterEntity,
		},
	},
}

var getChannelEntity = UserEntity{
	Name: "Get Channel",
	Actions: []randutil.Choice{
		{
			Item:   actionGetChannel,
			Weight: 1,
		},
	},
}

var TestGetChannel = TestRun{
	UserEntities: []UserEntityFrequency{
		{
			Freq:           100.0,
			RateMultiplier: 1.0,
			Entity:         getChannelEntity,
		},
	},
}

var searchEntity = UserEntity{
	Name: "Search",
	Actions: []randutil.Choice{
		{
			Item:   actionPerformSearch,
			Weight: 1,
		},
	},
}

var TestSearch = TestRun{
	UserEntities: []UserEntityFrequency{
		{
			Freq:           100.0,
			RateMultiplier: 1.0,
			Entity:         searchEntity,
		},
	},
}

var standardUserEntity = UserEntity{
	Name: "Standard",
	Actions: []randutil.Choice{
		{
			Item:   actionPost,
			Weight: 4,
		},
		{
			Item:   actionPerformSearch,
			Weight: 1,
		},
		{
			Item:   actionGetChannel,
			Weight: 28,
		},
		{
			Item:   actionDisconnectWebsocket,
			Weight: 2,
		},
		{
			Item:   actionJoinLeaveChannel,
			Weight: 5,
		},
	},
}

var webhookUserEntity = UserEntity{
	Name: "Webhook",
	Actions: []randutil.Choice{
		{
			Item:   actionPostWebhook,
			Weight: 1,
		},
	},
}

var TestAll = TestRun{
	UserEntities: []UserEntityFrequency{
		{
			Freq:           90.0,
			RateMultiplier: 1.0,
			Entity:         standardUserEntity,
		},
		{
			Freq:           10.0,
			RateMultiplier: 1.5,
			Entity:         webhookUserEntity,
		},
	},
}
