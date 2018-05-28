// Copyright (c) 2017 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package loadtest

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"bytes"

	"github.com/icrowley/fake"
	"github.com/mattermost/mattermost-load-test/randutil"
	"github.com/mattermost/mattermost-server/mlog"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/utils"
)

type TestRun struct {
	UserEntities []randutil.Choice
}

type UserEntityWithRateMultiplier struct {
	Entity         UserEntity
	RateMultiplier float64
}

type UserEntity struct {
	Name    string
	Actions []randutil.Choice
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
			mlog.Error("Unable to get channel from map")
			return
		}

		members, resp := c.Client.GetChannelMembers(channelId, 0, 60, "")
		if resp.Error != nil {
			mlog.Error("Unable to get members for channel to seed action get status.", mlog.String("channel_id", channelId), mlog.Err(resp.Error))
			return
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
		mlog.Error("Unable to get user statuses by Ids. Error: " + resp.Error.Error())
	}
}

func actionLeaveJoinTeam(c *EntityConfig) {
	importTeam := c.UserData.PickTeam()
	if importTeam == nil {
		return
	}

	teamId := c.TeamMap[importTeam.Name]
	if teamId == "" {
		mlog.Error("Unable to get team from map")
		return
	}

	userId := ""
	if user, resp := c.Client.GetMe(""); resp.Error != nil {
		mlog.Error("Failed to get me", mlog.Err(resp.Error))
		return
	} else {
		userId = user.Id
	}

	inviteId := ""
	if team, resp := c.Client.GetTeam(teamId, ""); resp.Error != nil {
		mlog.Error("Failed to get team", mlog.Err(resp.Error))
		return
	} else {
		inviteId = team.InviteId
	}

	if _, resp := c.Client.RemoveTeamMember(teamId, userId); resp.Error != nil {
		mlog.Error("Failed to leave team", mlog.String("team_id", teamId), mlog.Err(resp.Error))
		return
	}

	time.Sleep(time.Second * 1)

	if _, resp := c.Client.AddTeamMemberFromInvite("", inviteId); resp.Error != nil {
		mlog.Error("Failed to join team with invite_id", mlog.String("team_id", teamId), mlog.String("invite_id", inviteId), mlog.Err(resp.Error))
		return
	}
}

func actionPostToTownSquare(c *EntityConfig) {
	team := c.UserData.PickTeam()
	if team == nil {
		mlog.Error("Unable to get team for town-square")
		return
	}

	channelId := c.TownSquareMap[team.Name]

	if channelId == "" {
		mlog.Error("Unable to get town-square from map")
		return
	}

	mlog.Info("Posted to town-square")
	createPost(c, team, channelId)
}

func actionPost(c *EntityConfig) {
	team, channel := c.UserData.PickTeamChannel()
	if team == nil || channel == nil {
		return
	}
	channelId := c.ChannelMap[team.Name+channel.Name]

	if channelId == "" {
		mlog.Error("Unable to get channel from map")
		return
	}

	createPost(c, team, channelId)
}

func createPost(c *EntityConfig, team *UserTeamImportData, channelId string) {
	post := &model.Post{
		ChannelId: channelId,
		Message:   fake.Sentences(),
	}

	if rand.Float64() < c.LoadTestConfig.UserEntitiesConfiguration.ChannelLinkChance {
		if channel := team.PickChannel(); channel != nil {
			post.Message = post.Message + " ~" + channel.Name
		}
	}

	if rand.Float64() < c.LoadTestConfig.UserEntitiesConfiguration.UploadImageChance {
		numFiles := rand.Intn(3) + 1
		fileIds := make([]string, numFiles, numFiles)
		for i := 0; i < numFiles; i++ {
			if data, err, filename := readRandomTestFile(); err != nil {
				mlog.Error("Problem reading test file.", mlog.Err(err))
			} else {
				if file, resp := c.Client.UploadFile(data, channelId, filename); resp.Error != nil {
					mlog.Error("Unable to upload file. Error: " + resp.Error.Error())
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
		mlog.Info("Failed to post", mlog.String("team_name", team.Name), mlog.String("channel_id", channelId), mlog.String("username", c.UserData.Username), mlog.String("auth_token", c.Client.AuthToken), mlog.Err(resp.Error))
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
		mlog.Error("Unable to view channel.", mlog.String("channel_id", channelId), mlog.String("username", c.UserData.Username))
	}

	if _, resp := c.Client.GetChannelMember(channelId, "me", ""); resp.Error != nil {
		mlog.Error("Unable to get channel member.", mlog.String("channel_id", channelId), mlog.String("username", c.UserData.Username), mlog.Err(resp.Error))
	}

	if _, resp := c.Client.GetChannelMembers(channelId, 0, 60, ""); resp.Error != nil {
		mlog.Error("Unable to get channel member.", mlog.String("channel_id", channelId), mlog.String("username", c.UserData.Username), mlog.Err(resp.Error))
	}

	if _, resp := c.Client.GetChannelStats(channelId, ""); resp.Error != nil {
		mlog.Error("Unable to get channel member.", mlog.String("channel_id", channelId), mlog.String("username", c.UserData.Username), mlog.Err(resp.Error))
	}

	if posts, resp := c.Client.GetPostsForChannel(channelId, 0, 60, ""); resp.Error != nil {
		mlog.Error("Unable to get channel member.", mlog.String("channel_id", channelId), mlog.String("username", c.UserData.Username), mlog.Err(resp.Error))
	} else {
		if posts == nil {
			mlog.Error(fmt.Sprintf("Got nil posts for get posts for channel. Resp was: %#v", resp))
			return
		}
		for _, post := range posts.Posts {
			if post.HasReactions {
				if _, resp := c.Client.GetReactions(post.Id); resp.Error != nil {
					mlog.Error("Unable to get reactions for post.", mlog.String("channel_id", channelId), mlog.String("username", c.UserData.Username), mlog.String("post_id", post.Id), mlog.Err(resp.Error))
				}
			}
			if len(post.FileIds) > 0 {
				if files, resp := c.Client.GetFileInfosForPost(post.Id, ""); resp.Error != nil {
					mlog.Error("Unable to get file infos for post.", mlog.String("channel_id", channelId), mlog.String("username", c.UserData.Username), mlog.String("post_id", post.Id), mlog.Err(resp.Error))
				} else {
					for _, file := range files {
						if file.IsImage() {
							if _, resp := c.Client.GetFileThumbnail(file.Id); resp.Error != nil {
								mlog.Error("Unable to get file thumbnail for file.", mlog.String("channel_id", channelId), mlog.String("username", c.UserData.Username), mlog.String("post_id", post.Id), mlog.String("file_id", file.Id), mlog.Err(resp.Error))
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
		mlog.Error("Failed to search", mlog.Err(resp.Error))
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
			mlog.Error("Unable to create incoming webhook. Error: " + resp.Error.Error())
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
		mlog.Error("Unable to marshal json for webhook request")
		return
	}

	var buf bytes.Buffer
	buf.WriteString(string(b))

	if resp, err := http.Post(c.LoadTestConfig.ConnectionConfiguration.ServerURL+"/hooks/"+hookId, "application/json", &buf); err != nil {
		mlog.Error("Failed to post by webhook. Error: " + err.Error())
	} else if resp != nil {
		resp.Body.Close()
	}
}

var posterEntity UserEntity = UserEntity{
	Name: "Poster",
	Actions: []randutil.Choice{
		{
			Item:   actionPost,
			Weight: 1,
		},
	},
}

var TestBasicPosting TestRun = TestRun{
	UserEntities: []randutil.Choice{
		{
			Item: UserEntityWithRateMultiplier{
				Entity:         posterEntity,
				RateMultiplier: 1.0,
			},
			Weight: 100,
		},
	},
}

var getChannelEntity UserEntity = UserEntity{
	Name: "Get Channel",
	Actions: []randutil.Choice{
		{
			Item:   actionGetChannel,
			Weight: 1,
		},
	},
}

var TestGetChannel TestRun = TestRun{
	UserEntities: []randutil.Choice{
		{
			Item: UserEntityWithRateMultiplier{
				Entity:         getChannelEntity,
				RateMultiplier: 1.0,
			},
			Weight: 100,
		},
	},
}

var searchEntity UserEntity = UserEntity{
	Name: "Search",
	Actions: []randutil.Choice{
		{
			Item:   actionPerformSearch,
			Weight: 1,
		},
	},
}

var TestSearch TestRun = TestRun{
	UserEntities: []randutil.Choice{
		{
			Item: UserEntityWithRateMultiplier{
				Entity:         searchEntity,
				RateMultiplier: 1.0,
			},
			Weight: 100,
		},
	},
}

var standardUserEntity UserEntity = UserEntity{
	Name: "Standard",
	Actions: []randutil.Choice{
		{
			Item:   actionPost,
			Weight: 8,
		},
		{
			Item:   actionPerformSearch,
			Weight: 2,
		},
		{
			Item:   actionGetChannel,
			Weight: 56,
		},
		{
			Item:   actionDisconnectWebsocket,
			Weight: 4,
		},
		{
			Item:   actionDeactivateReactivate,
			Weight: 1,
		},
	},
}

var webhookUserEntity UserEntity = UserEntity{
	Name: "Webhook",
	Actions: []randutil.Choice{
		{
			Item:   actionPostWebhook,
			Weight: 1,
		},
	},
}

var TestAll TestRun = TestRun{
	UserEntities: []randutil.Choice{
		{
			Item: UserEntityWithRateMultiplier{
				Entity:         standardUserEntity,
				RateMultiplier: 1.0,
			},
			Weight: 90,
		},
		{
			Item: UserEntityWithRateMultiplier{
				Entity:         webhookUserEntity,
				RateMultiplier: 1.5,
			},
			Weight: 10,
		},
	},
}

var townSquareSpammerUserEntity UserEntity = UserEntity{
	Name: "TownSquareSpammer",
	Actions: []randutil.Choice{
		{
			Item:   actionPostToTownSquare,
			Weight: 1,
		},
	},
}

var TestTownSquareSpam TestRun = TestRun{
	UserEntities: []randutil.Choice{
		{
			Item: UserEntityWithRateMultiplier{
				Entity:         standardUserEntity,
				RateMultiplier: 1.0,
			},
			Weight: 90,
		},
		{
			Item: UserEntityWithRateMultiplier{
				Entity:         townSquareSpammerUserEntity,
				RateMultiplier: 1.0,
			},
			Weight: 10,
		},
	},
}

var teamLeaverJoinerUserEntity UserEntity = UserEntity{
	Name: "TeamLeaverJoiner",
	Actions: []randutil.Choice{
		{
			Item:   actionLeaveJoinTeam,
			Weight: 1,
		},
	},
}

var TestLeaveJoinTeam TestRun = TestRun{
	UserEntities: []randutil.Choice{
		{
			Item: UserEntityWithRateMultiplier{
				Entity:         standardUserEntity,
				RateMultiplier: 1.0,
			},
			Weight: 90,
		},
		{
			Item: UserEntityWithRateMultiplier{
				Entity:         teamLeaverJoinerUserEntity,
				RateMultiplier: 1.0,
			},
			Weight: 10,
		},
	},
}

func actionDeactivateReactivate(c *EntityConfig) {
	user, resp := c.Client.GetMe("")
	if resp.Error != nil {
		mlog.Error("Failed to get me", mlog.Err(resp.Error))
		return
	}

	if ok, resp := c.AdminClient.UpdateUserActive(user.Id, false); !ok {
		mlog.Error("Failed to deactivate user", mlog.String("user_id", user.Id), mlog.Err(resp.Error))
	} else {
		mlog.Info("Deactivated user", mlog.String("user_id", user.Id))
	}

	time.Sleep(time.Second * 1)

	if ok, resp := c.AdminClient.UpdateUserActive(user.Id, true); !ok {
		mlog.Error("Failed to reactivate user", mlog.String("user_id", user.Id), mlog.Err(resp.Error))
	} else {
		mlog.Info("Reactivated user", mlog.String("user_id", user.Id))
	}

	// Login again since the token will have been invalidated.
	if _, response := c.Client.Login(user.Email, "Loadtestpassword1"); response != nil && response.Error != nil {
		mlog.Error("Failed to recreate client as user %s: %s", mlog.String("email", user.Email), mlog.Err(response.Error))
	} else {
		mlog.Info("Recreated client as user", mlog.String("email", user.Email))
	}
}

var deactivatingUserEntity UserEntity = UserEntity{
	Name: "DeactivatingUserEntity",
	Actions: []randutil.Choice{
		{
			Item:   actionDeactivateReactivate,
			Weight: 1,
		},
	},
}

var TestDeactivation TestRun = TestRun{
	UserEntities: []randutil.Choice{
		{
			Item: UserEntityWithRateMultiplier{
				Entity:         standardUserEntity,
				RateMultiplier: 1.0,
			},
			Weight: 70,
		},
		{
			Item: UserEntityWithRateMultiplier{
				Entity:         deactivatingUserEntity,
				RateMultiplier: 1.0,
			},
			Weight: 30,
		},
	},
}
