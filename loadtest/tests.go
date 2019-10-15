// Copyright (c) 2017 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package loadtest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"bytes"

	"github.com/icrowley/fake"
	"github.com/mattermost/mattermost-load-test/randutil"
	"github.com/mattermost/mattermost-server/mlog"
	"github.com/mattermost/mattermost-server/model"
)

const (
	OPENGRAPH_TEST_URL = "https://s3.amazonaws.com/mattermost-load-test-media/index.html"
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
	file, err := os.Open(name)
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
	files, err := ioutil.ReadDir("./testfiles")
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
	return b, err, filepath.Join("./testfiles", file.Name())
}

func actionGetStatuses(c *EntityConfig) {
	idsI, ok := c.Info["statusUserIds"+c.UserData.Username]
	var ids []string
	if !ok {
		team, channel := c.UserData.PickTeamChannel(c.r)
		if team == nil || channel == nil {
			return
		}
		channelId, err := c.GetTeamChannelId(team.Name, channel.Name)
		if err != nil {
			mlog.Error("Unable to get channel from map", mlog.String("team", team.Name), mlog.String("channel", channel.Name), mlog.Err(err))
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

func actionLeaveJoinChannel(c *EntityConfig) {
	team, channel := c.UserData.PickTeamChannel(c.r)

	if team == nil || channel == nil {
		return
	}

	channelId, err := c.GetTeamChannelId(team.Name, channel.Name)
	if err != nil {
		mlog.Error("Unable to get channel from map", mlog.String("team", team.Name), mlog.String("channel", channel.Name), mlog.Err(err))
		return
	}

	userId := ""
	if user, resp := c.Client.GetMe(""); resp.Error != nil {
		mlog.Error("Failed to get me", mlog.Err(resp.Error))
		return
	} else {
		userId = user.Id
	}

	if _, resp := c.Client.GetChannel(channelId, ""); resp.Error != nil {
		mlog.Error("Failed to get channel", mlog.String("channel_id", channelId), mlog.Err(resp.Error))
		return
	}

	removed, _ := c.Client.RemoveUserFromChannel(channelId, userId)

	if removed {
		time.Sleep(1 * time.Second)
	}

	_, resp := c.Client.AddChannelMember(channelId, userId)
	if resp.Error != nil {
		mlog.Error("Failed to add user to channel", mlog.String("channel_id", channelId), mlog.String("user_id", userId), mlog.Err(resp.Error))
		return
	}

	if !removed {
		time.Sleep(1 * time.Second)
		_, resp = c.Client.RemoveUserFromChannel(channelId, userId)
		if resp.Error != nil {
			mlog.Error("Failed remove user from channel", mlog.String("channel_id", channelId), mlog.String("user_id", userId), mlog.Err(resp.Error))
			return
		}
	}
}

func actionLeaveJoinTeam(c *EntityConfig) {
	importTeam := c.UserData.PickTeam(c.r)
	if importTeam == nil {
		return
	}

	teamId := c.TeamMap[importTeam.Name]
	if teamId == "" {
		mlog.Error("Unable to get team from map", mlog.String("team", importTeam.Name))
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

	if rand.Float64() > 0.5 {
		if _, resp := c.Client.AddTeamMemberFromInvite("", inviteId); resp.Error != nil {
			mlog.Error("Failed to join team with invite_id", mlog.String("team_id", teamId), mlog.String("invite_id", inviteId), mlog.Err(resp.Error))
			return
		}
	} else {
		if _, resp := c.Client.AddTeamMember(teamId, userId); resp.Error != nil {
			mlog.Error("Failed to join team", mlog.String("team_id", teamId), mlog.String("user_id", userId), mlog.Err(resp.Error))
			return
		}
	}
}

func actionPostToTownSquare(c *EntityConfig) {
	team := c.UserData.PickTeam(c.r)
	if team == nil {
		mlog.Error("Unable to get team for town-square")
		return
	}

	channelId := c.TownSquareMap[team.Name]

	if channelId == "" {
		mlog.Error("Unable to get town-square from map", mlog.String("team", team.Name))
		return
	}

	mlog.Info("Posted to town-square")
	createPost(c, team, channelId)
}

func actionPost(c *EntityConfig) {
	team, channel := c.UserData.PickTeamChannel(c.r)
	if team == nil || channel == nil {
		return
	}
	channelId, err := c.GetTeamChannelId(team.Name, channel.Name)
	if err != nil {
		mlog.Error("Unable to get channel from map", mlog.String("team", team.Name), mlog.String("channel", channel.Name), mlog.Err(err))
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
		if channel := team.PickChannel(c.r); channel != nil {
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

	if rand.Float64() < c.LoadTestConfig.UserEntitiesConfiguration.LinkPreviewChance {
		post.Message = post.Message + " " + OPENGRAPH_TEST_URL
	}

	if rand.Float64() < c.LoadTestConfig.UserEntitiesConfiguration.CustomEmojiChance && c.LoadTestConfig.LoadtestEnviromentConfig.NumEmoji > 0 {
		name := c.LoadTestConfig.LoadtestEnviromentConfig.PickEmoji(c.r)
		post.Message = post.Message + " :" + name + ":"
	}

	post, resp := c.Client.CreatePost(post)
	if resp.Error != nil {
		mlog.Info("Failed to post", mlog.String("team_name", team.Name), mlog.String("channel_id", channelId), mlog.String("username", c.UserData.Username), mlog.String("auth_token", c.Client.AuthToken), mlog.Err(resp.Error))
	}

	if rand.Float64() < c.LoadTestConfig.UserEntitiesConfiguration.CustomEmojiReactionChance && c.LoadTestConfig.LoadtestEnviromentConfig.NumEmoji > 0 {
		name := c.LoadTestConfig.LoadtestEnviromentConfig.PickEmoji(c.r)
		addReaction(c, post.UserId, post.Id, name)
	}

	if rand.Float64() < c.LoadTestConfig.UserEntitiesConfiguration.SystemEmojiReactionChance {
		addReaction(c, post.UserId, post.Id, "smile")
	}
}

func searchRandomUsers(c *EntityConfig, teamId string, n int) ([]*model.User, error) {
	list := []*model.User{}
	for i := 0; i < n; i++ {
		userdata := PickUser(c.Users, c.r)
		if userdata == nil {
			return nil, errors.New("could not randomly pick user")
		}
		for j := 1; j <= len(userdata.Username); j++ {
			substring := userdata.Username[:j]
			users, resp := c.Client.SearchUsers(&model.UserSearch{TeamId: teamId, Term: substring})
			if resp.Error != nil {
				return nil, resp.Error
			}
			if len(users) == 1 {
				list = append(list, users[0])
				break
			}
			time.Sleep(time.Millisecond * 150)
		}
	}
	return list, nil
}

func createChannel(c *EntityConfig, teamId string, userId string) (*model.Channel, error) {
	const (
		PUBLIC_CHANNEL = iota + 1
		PRIVATE_CHANNEL
		DIRECT_CHANNEL
		GROUP_CHANNEL
	)

	choices := []randutil.Choice{
		{
			Weight: int(c.LoadTestConfig.UserEntitiesConfiguration.PublicChannelCreationChance * 1000),
			Item:   PUBLIC_CHANNEL,
		},
		{
			Weight: int(c.LoadTestConfig.UserEntitiesConfiguration.PrivateChannelCreationChance * 1000),
			Item:   PRIVATE_CHANNEL,
		},
		{
			Weight: int(c.LoadTestConfig.UserEntitiesConfiguration.DirectChannelCreationChance * 1000),
			Item:   DIRECT_CHANNEL,
		},
		{
			Weight: int(c.LoadTestConfig.UserEntitiesConfiguration.GroupChannelCreationChance * 1000),
			Item:   GROUP_CHANNEL,
		},
	}

	choice, err := randutil.WeightedChoice(c.r, choices)
	if err != nil {
		return nil, err
	}

	if choice.Item == PUBLIC_CHANNEL || choice.Item == PRIVATE_CHANNEL {
		channel := &model.Channel{}
		channel.TeamId = teamId
		channel.Name = makeChannelName(c.r.Int())
		channel.DisplayName = makeChannelDisplayName(c.r.Int())

		if choice.Item == PUBLIC_CHANNEL {
			channel.Type = model.CHANNEL_OPEN
		} else {
			channel.Type = model.CHANNEL_PRIVATE
		}

		newChannel, resp := c.Client.CreateChannel(channel)
		if resp.Error != nil {
			return nil, resp.Error
		}

		return newChannel, nil
	} else if choice.Item == DIRECT_CHANNEL || choice.Item == GROUP_CHANNEL {
		var newChannel *model.Channel
		var resp *model.Response

		if choice.Item == DIRECT_CHANNEL {
			users, err := searchRandomUsers(c, teamId, 1)
			if err != nil {
				return nil, err
			} else if len(users) != 1 {
				return nil, errors.New("could not find user to create direct channel with")
			}

			newChannel, resp = c.Client.CreateDirectChannel(userId, users[0].Id)
		} else {
			users, err := searchRandomUsers(c, teamId, 2)
			if err != nil {
				return nil, err
			} else if len(users) != 2 {
				return nil, errors.New("could not find users to create group channel with")
			}

			userIds := []string{}
			for i := 0; i < len(users); i++ {
				userIds = append(userIds, users[i].Id)
			}

			newChannel, resp = c.Client.CreateGroupChannel(userIds)
		}

		if resp.Error != nil {
			return nil, resp.Error
		}

		return newChannel, nil
	}

	return nil, nil
}

func actionCreateDeleteChannel(c *EntityConfig) {
	team := c.UserData.PickTeam(c.r)
	if team == nil {
		return
	}
	teamId := c.TeamMap[team.Name]

	user, resp := c.Client.GetMe("")
	if resp.Error != nil {
		mlog.Error("Failed to get me", mlog.Err(resp.Error))
		return
	}

	channel, err := createChannel(c, teamId, user.Id)
	if err != nil {
		mlog.Error("Failed to create channel", mlog.Err(err))
		return
	}

	if channel.Type != model.CHANNEL_OPEN && channel.Type != model.CHANNEL_PRIVATE {
		return
	}

	if _, resp := c.Client.DeleteChannel(channel.Id); resp.Error != nil {
		mlog.Error("Failed to delete channel", mlog.Err(resp.Error), mlog.String("channel_id", channel.Id))
		return
	}
}

func addReaction(c *EntityConfig, userId, postId, name string) {
	reaction := &model.Reaction{
		UserId:    userId,
		PostId:    postId,
		EmojiName: name,
	}

	_, resp := c.Client.SaveReaction(reaction)
	if resp.Error != nil {
		mlog.Info("Failed to save reaction", mlog.String("user_id", reaction.UserId), mlog.String("post_id", reaction.PostId), mlog.String("emoji_name", reaction.EmojiName))
	}
}

func actionPostReactions(c *EntityConfig) {
	user, resp := c.Client.GetMe("")
	if resp.Error != nil {
		mlog.Error("Failed to get me", mlog.Err(resp.Error))
		return
	}

	team, channel := c.UserData.PickTeamChannel(c.r)
	if team == nil || channel == nil {
		return
	}

	teamId := c.TeamMap[team.Name]
	if teamId == "" {
		mlog.Error("Unable to get team from map", mlog.String("team", team.Name))
		return
	}

	channels, resp := c.Client.GetChannelsForTeamForUser(teamId, user.Id, "")
	if resp.Error != nil {
		mlog.Info("Unable to get channels for user", mlog.String("user_id", user.Id), mlog.Err(resp.Error))
		return
	}

	length := len(channels)
	if length == 0 {
		return
	}

	idx, _ := randutil.IntRange(c.r, 0, length)
	channelId := channels[idx].Id

	list, resp := c.Client.GetPostsForChannel(channelId, 0, 60, "")
	if resp.Error != nil {
		mlog.Error("Unable to get posts for channel", mlog.String("channel_id", channelId), mlog.Err(resp.Error))
		return
	}

	length = len(list.Order)
	if length == 0 {
		return
	}

	idx, _ = randutil.IntRange(c.r, 0, length)

	numReactions := c.LoadTestConfig.UserEntitiesConfiguration.NumPostReactionsPerUser

	for i := 0; i < numReactions; i++ {
		emojiName := c.LoadTestConfig.LoadtestEnviromentConfig.PickEmoji(c.r)
		addReaction(c, user.Id, list.Order[idx], emojiName)
		if i != (numReactions - 1) {
			time.Sleep(time.Duration(c.LoadTestConfig.UserEntitiesConfiguration.PostReactionsRateMilliseconds) * time.Millisecond)
		}
	}
}

func actionGetChannel(c *EntityConfig) {
	team, channel := c.UserData.PickTeamChannel(c.r)
	if team == nil || channel == nil {
		return
	}

	channelId, err := c.GetTeamChannelId(team.Name, channel.Name)
	if err != nil {
		mlog.Error("Unable to get channel from map", mlog.String("team", team.Name), mlog.String("channel", channel.Name), mlog.Err(err))
		return
	}

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

	// The webapp is observed to invoke ViewChannel once without a PrevChannelId, and once with
	// one specified. Duplicate that behaviour here.
	prevChannel := team.PickChannel(c.r)
	if prevChannel != nil {
		prevChannelId, err := c.GetTeamChannelId(team.Name, prevChannel.Name)
		if err != nil {
			mlog.Error("Unable to get channel from map", mlog.String("team", team.Name), mlog.String("channel", channel.Name), mlog.Err(err))
			return
		}

		if _, resp := c.Client.ViewChannel("me", &model.ChannelView{
			ChannelId:     channelId,
			PrevChannelId: prevChannelId,
		}); resp.Error != nil {
			mlog.Error("Unable to view channel.", mlog.String("channel_id", channelId), mlog.String("prev_channel_id", prevChannelId), mlog.String("username", c.UserData.Username))
		}
	}

	if posts, resp := c.Client.GetPostsForChannel(channelId, 0, 60, ""); resp.Error != nil {
		mlog.Error("Unable to get channel member.", mlog.String("channel_id", channelId), mlog.String("username", c.UserData.Username), mlog.Err(resp.Error))
	} else {
		if posts == nil {
			mlog.Error(fmt.Sprintf("Got nil posts for get posts for channel. Resp was: %#v", resp))
			return
		}
		for _, post := range posts.Posts {
			if post.Metadata != nil {
				for _, file := range post.Metadata.Files {
					if file.IsImage() {
						if _, resp := c.Client.GetFileThumbnail(file.Id); resp.Error != nil {
							mlog.Error("Unable to get file thumbnail for file.", mlog.String("channel_id", channelId), mlog.String("username", c.UserData.Username), mlog.String("post_id", post.Id), mlog.String("file_id", file.Id), mlog.Err(resp.Error))
						}
					}
				}
			} else {
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

				if rand.Float64() < c.LoadTestConfig.UserEntitiesConfiguration.LinkPreviewChance {
					if _, resp := c.Client.OpenGraph(OPENGRAPH_TEST_URL); resp.Error != nil {
						mlog.Error("Unable to get open graph for url.", mlog.String("url", OPENGRAPH_TEST_URL), mlog.String("user_id", post.UserId), mlog.Err(resp.Error))
					}
				}

				if rand.Float64() < c.LoadTestConfig.UserEntitiesConfiguration.CustomEmojiChance && c.LoadTestConfig.LoadtestEnviromentConfig.NumEmoji > 0 {
					name := c.LoadTestConfig.LoadtestEnviromentConfig.PickEmoji(c.r)
					if _, resp := c.Client.GetEmojiByName(name); resp.Error != nil {
						mlog.Error("Unable to get emoji.", mlog.String("emoji_name", name), mlog.String("user_id", post.UserId), mlog.Err(resp.Error))
					}
				}
			}
		}
	}

	usersToQueryById := make([]string, 0)
	for rand.Float64() < c.LoadTestConfig.UserEntitiesConfiguration.NeedsProfilesByIdChance {
		nextUser := "user" + strconv.Itoa(rand.Intn(c.LoadTestConfig.LoadtestEnviromentConfig.NumUsers))
		usersToQueryById = append(usersToQueryById, nextUser)
	}
	if len(usersToQueryById) > 0 {
		if _, resp := c.Client.GetUsersByIds(usersToQueryById); resp.Error != nil {
			mlog.Error("Unable to get users by ids", mlog.Err(resp.Error))
		}
	}

	usersToQueryByUsername := make([]string, 0)
	for rand.Float64() < c.LoadTestConfig.UserEntitiesConfiguration.NeedsProfilesByUsernameChance {
		if rand.Float64() > 0.5 {
			nextUser := "user" + strconv.Itoa(rand.Intn(c.LoadTestConfig.LoadtestEnviromentConfig.NumUsers))
			usersToQueryByUsername = append(usersToQueryByUsername, nextUser)
		} else {
			nextUser := model.NewId()
			usersToQueryByUsername = append(usersToQueryByUsername, nextUser)
		}
	}
	if len(usersToQueryByUsername) > 0 {
		if _, resp := c.Client.GetUsersByUsernames(usersToQueryByUsername); resp.Error != nil {
			mlog.Error("Unable to get users by usernames", mlog.Err(resp.Error))
		}
	}

	usersToQueryForStatusById := make([]string, 0)
	for rand.Float64() < c.LoadTestConfig.UserEntitiesConfiguration.NeedsProfileStatusChance {
		nextUser := "user" + strconv.Itoa(rand.Intn(c.LoadTestConfig.LoadtestEnviromentConfig.NumUsers))
		usersToQueryForStatusById = append(usersToQueryForStatusById, nextUser)
	}
	if len(usersToQueryForStatusById) > 0 {
		if _, resp := c.Client.GetUsersStatusesByIds(usersToQueryForStatusById); resp.Error != nil {
			mlog.Error("Unable to get user statuses by ids", mlog.Err(resp.Error))
		}
	}
}

func actionPerformSearch(c *EntityConfig) *model.PostList {
	team, _ := c.UserData.PickTeamChannel(c.r)
	if team == nil {
		return nil
	}
	teamId := c.TeamMap[team.Name]

	list, resp := c.Client.SearchPosts(teamId, fake.Words(), false)
	if resp.Error != nil {
		mlog.Error("Failed to search", mlog.Err(resp.Error))
		return nil
	}

	return list
}

func actionGetPostsBeforeAfter(c *EntityConfig) {
	list := actionPerformSearch(c)

	if list == nil {
		return
	}

	length := len(list.Order)
	if length == 0 {
		return
	}

	idx, _ := randutil.IntRange(c.r, 0, length)
	post := list.Posts[list.Order[idx]]

	_, resp := c.Client.GetPostsBefore(post.ChannelId, post.Id, 0, c.LoadTestConfig.UserEntitiesConfiguration.NumPostsGetBeforeAfter, "")

	if resp.Error != nil {
		mlog.Error("Failed to get posts before", mlog.String("channel_id", post.ChannelId), mlog.String("post_id", post.Id), mlog.Err(resp.Error))
		return
	}

	_, resp = c.Client.GetPostsAfter(post.ChannelId, post.Id, 0, c.LoadTestConfig.UserEntitiesConfiguration.NumPostsGetBeforeAfter, "")

	if resp.Error != nil {
		mlog.Error("Failed to get posts after", mlog.String("channel_id", post.ChannelId), mlog.String("post_id", post.Id), mlog.Err(resp.Error))
		return
	}
}

func actionAutocompleteChannel(c *EntityConfig) {
	team, channel := c.UserData.PickTeamChannel(c.r)
	if team == nil || channel == nil {
		return
	}
	teamId := c.TeamMap[team.Name]

	// Select a random fraction of the channel name to actually type
	typedName := channel.Name[:rand.Intn(len(channel.Name))]

	for i := 1; i <= len(typedName); i++ {
		currentSubstring := typedName[:i]
		go func() {
			if _, resp := c.Client.AutocompleteChannelsForTeam(teamId, currentSubstring); resp.Error != nil {
				mlog.Error("Unable to autocomplete channel", mlog.String("team_name", team.Name), mlog.String("channel_name", channel.Name), mlog.String("fragment", currentSubstring))
			}
		}()
		time.Sleep(time.Millisecond * 150)
	}
}

func actionSearchChannel(c *EntityConfig) {
	team, channel := c.UserData.PickTeamChannel(c.r)
	if team == nil || channel == nil {
		return
	}
	teamId := c.TeamMap[team.Name]

	// Select a random fraction of the channel name to actually type
	typedName := channel.Name[:rand.Intn(len(channel.Name))]

	for i := 1; i <= len(typedName); i++ {
		currentSubstring := typedName[:i]
		go func() {
			if _, resp := c.Client.SearchChannels(teamId, &model.ChannelSearch{Term: currentSubstring}); resp.Error != nil {
				mlog.Error("Unable to search channel", mlog.String("team_name", team.Name), mlog.String("channel_name", channel.Name), mlog.String("fragment", currentSubstring))
			}
		}()
		time.Sleep(time.Millisecond * 150)
	}
}

func actionSearchUser(c *EntityConfig) {
	team := c.UserData.PickTeam(c.r)
	if team == nil {
		return
	}
	teamId := c.TeamMap[team.Name]

	// Select a random field to search by
	var searchField string
	r := rand.Intn(4)
	switch r {
	case 0:
		searchField = c.UserData.Username
	case 1:
		searchField = c.UserData.FirstName
	case 2:
		searchField = c.UserData.LastName
	case 3:
		searchField = c.UserData.Nickname
	}
	// but use username if the other fields aren't set
	if searchField == "" {
		searchField = c.UserData.Username
	}
	if searchField == "" {
		return
	}

	// Select a random fraction of the username to actually type
	typedName := searchField[:(rand.Intn(len(searchField) + 1))]

	for i := 1; i <= len(typedName); i++ {
		currentSubstring := typedName[:i]
		go func() {
			if _, resp := c.Client.SearchUsers(&model.UserSearch{TeamId: teamId, Term: currentSubstring}); resp.Error != nil {
				mlog.Error("Unable to search users", mlog.String("team_name", team.Name), mlog.String("term", currentSubstring))
			}
		}()
		time.Sleep(time.Millisecond * 150)
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
		team, channel := c.UserData.PickTeamChannel(c.r)
		if team == nil || channel == nil {
			return
		}
		channelId, err := c.GetTeamChannelId(team.Name, channel.Name)
		if err != nil {
			mlog.Error("Unable to get channel from map", mlog.String("team", team.Name), mlog.String("channel", channel.Name), mlog.Err(err))
			return
		}

		webhook, resp := c.AdminClient.CreateIncomingWebhook(&model.IncomingWebhook{
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

func actionGetTeamUnreads(c *EntityConfig) {
	_, response := c.Client.GetTeamsUnreadForUser("me", "")
	if response.Error != nil {
		mlog.Error("Failed to get team unreads", mlog.String("user", c.UserData.Username), mlog.Err(response.Error))
	}
}

func actionGetChannelUnreads(c *EntityConfig) {
	user, resp := c.Client.GetMe("")
	if resp.Error != nil {
		mlog.Error("Failed to get me", mlog.Err(resp.Error))
		return
	}

	team, channel := c.UserData.PickTeamChannel(c.r)
	if team == nil || channel == nil {
		return
	}

	channelId, err := c.GetTeamChannelId(team.Name, channel.Name)
	if err != nil {
		mlog.Error("Unable to get channel from map", mlog.String("team", team.Name), mlog.String("channel", channel.Name), mlog.Err(err))
		return
	}

	if rand.Float64() < c.LoadTestConfig.UserEntitiesConfiguration.GetPostsAroundLastUnreadChance {
		numPosts := c.LoadTestConfig.UserEntitiesConfiguration.NumGetPostsAroundLastUnread
		_, resp := c.Client.GetPostsAroundLastUnread(channelId, user.Id, numPosts, numPosts)
		if resp.Error != nil {
			mlog.Info("Failed to get posts around last unread", mlog.String("channel_id", channelId), mlog.Err(resp.Error))
			return
		}
	}

	_, resp = c.Client.GetChannelUnread(channelId, user.Id)
	if resp.Error != nil {
		mlog.Info("Failed to get channel unreads", mlog.String("channel_id", channelId), mlog.Err(resp.Error))
		return
	}
}

func actionUpdateUserProfile(c *EntityConfig) {
	user, resp := c.Client.GetMe("")
	if resp.Error != nil {
		mlog.Error("Failed to get me", mlog.Err(resp.Error))
		return
	}

	patch := &model.UserPatch{}

	if rand.Float64() < c.LoadTestConfig.UserEntitiesConfiguration.UserProfileUpdateFullnameChance {
		patch.FirstName = model.NewString(fmt.Sprintf("%s_new", user.FirstName))
		patch.LastName = model.NewString(fmt.Sprintf("%s_new", user.LastName))
	}

	if rand.Float64() < c.LoadTestConfig.UserEntitiesConfiguration.UserProfileUpdateUsernameChance {
		patch.Username = model.NewString(fmt.Sprintf("%s_new", user.Username))
	}

	if rand.Float64() < c.LoadTestConfig.UserEntitiesConfiguration.UserProfileUpdateNicknameChance {
		patch.Nickname = model.NewString(fmt.Sprintf("%s_new", user.Nickname))
	}

	if rand.Float64() < c.LoadTestConfig.UserEntitiesConfiguration.UserProfileUpdatePositionChance {
		patch.Position = model.NewString(fmt.Sprintf("%s_new", user.Position))
	}

	if rand.Float64() < c.LoadTestConfig.UserEntitiesConfiguration.UserProfileUpdateEmailChance {
		patch.Email = model.NewString(fmt.Sprintf("new_%s", user.Email))
		patch.Password = model.NewString(c.UserData.Password)
	}

	if rand.Float64() < c.LoadTestConfig.UserEntitiesConfiguration.UserProfileUpdateImageChance {
		imagePath := "./testfiles/test.png"
		imageData, err := readTestFile(imagePath)
		if err != nil {
			mlog.Error("Failed reading testfile", mlog.String("filename", imagePath), mlog.Err(err))
			return
		}
		_, resp = c.Client.SetProfileImage(user.Id, imageData)
		if resp.Error != nil {
			mlog.Error("Failed to set user profile image", mlog.String("user_id", user.Id), mlog.Err(resp.Error))
			return
		}
	}

	if patch.FirstName == nil && patch.LastName == nil && patch.Username == nil && patch.Nickname == nil && patch.Position == nil && patch.Email == nil {
		return
	}

	_, resp = c.Client.PatchUser(user.Id, patch)
	if resp.Error != nil {
		mlog.Error("Failed to patch user", mlog.String("user_id", user.Id), mlog.Err(resp.Error))
		return
	}

	_, resp = c.Client.GetMe("")
	if resp.Error != nil {
		mlog.Error("Failed to get me", mlog.Err(resp.Error))
		return
	}

	user.Password = c.UserData.Password
	_, resp = c.Client.UpdateUser(user)
	if resp.Error != nil {
		mlog.Error("Failed to update user", mlog.String("user_id", user.Id), mlog.Err(resp.Error))
		return
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

var reactionsPoster UserEntity = UserEntity{
	Name: "ReactionsPoster",
	Actions: []randutil.Choice{
		{
			Item:   actionPostReactions,
			Weight: 1,
		},
	},
}

var TestPostReactions TestRun = TestRun{
	UserEntities: []randutil.Choice{
		{
			Item: UserEntityWithRateMultiplier{
				Entity:         reactionsPoster,
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

var searchUsersEntity UserEntity = UserEntity{
	Name: "Search Users",
	Actions: []randutil.Choice{
		{
			Item:   actionSearchUser,
			Weight: 1,
		},
	},
}

var TestSearchUsers TestRun = TestRun{
	UserEntities: []randutil.Choice{
		{
			Item: UserEntityWithRateMultiplier{
				Entity:         searchUsersEntity,
				RateMultiplier: 1.0,
			},
			Weight: 100,
		},
	},
}

var channelCreateDeleteEntity UserEntity = UserEntity{
	Name: "Create/Delete channel",
	Actions: []randutil.Choice{
		{
			Item:   actionCreateDeleteChannel,
			Weight: 1,
		},
	},
}

var TestChannelCreateDelete TestRun = TestRun{
	UserEntities: []randutil.Choice{
		{
			Item: UserEntityWithRateMultiplier{
				Entity:         channelCreateDeleteEntity,
				RateMultiplier: 1.0,
			},
			Weight: 100,
		},
	},
}

var updateUserProfileEntity UserEntity = UserEntity{
	Name: "Update User Profile",
	Actions: []randutil.Choice{
		{
			Item:   actionUpdateUserProfile,
			Weight: 1,
		},
	},
}

var TestUpdateUserProfile TestRun = TestRun{
	UserEntities: []randutil.Choice{
		{
			Item: UserEntityWithRateMultiplier{
				Entity:         updateUserProfileEntity,
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
			Item:   actionGetTeamUnreads,
			Weight: 41,
		},
		{
			Item:   actionGetChannelUnreads,
			Weight: 10,
		},
		{
			Item:   actionAutocompleteChannel,
			Weight: 1,
		},
		{
			Item:   actionSearchChannel,
			Weight: 10,
		},
		{
			Item:   actionDisconnectWebsocket,
			Weight: 4,
		},
		{
			Item:   actionMoreChannels,
			Weight: 4,
		},
		{
			Item:   actionSearchUser,
			Weight: 2,
		},
		{
			Item:   actionUpdateUserProfile,
			Weight: 1,
		},
		{
			Item:   actionGetPostsBeforeAfter,
			Weight: 2,
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

var channelLeaverJoinerUserEntity UserEntity = UserEntity{
	Name: "ChannelLeaverJoiner",
	Actions: []randutil.Choice{
		{
			Item:   actionLeaveJoinChannel,
			Weight: 1,
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

var TestLeaveJoinChannel TestRun = TestRun{
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
				Entity:         channelLeaverJoinerUserEntity,
				RateMultiplier: 1.0,
			},
			Weight: 10,
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
	if _, response := c.Client.Login(user.Email, "Loadtestpassword1@#%"); response != nil && response.Error != nil {
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

const CHANNELS_CHUNK_SIZE = 50
const CHANNELS_FETCH_SIZE = CHANNELS_CHUNK_SIZE * 2

func actionMoreChannels(c *EntityConfig) {
	team := c.UserData.PickTeam(c.r)
	if team == nil {
		return
	}

	teamId := c.TeamMap[team.Name]
	if teamId == "" {
		mlog.Error("Unable to get team from map", mlog.String("team", team.Name))
		return
	}

	numChannels := len(c.ChannelMap[team.Name])

	for i := 0; i < numChannels; i += CHANNELS_FETCH_SIZE {
		page := i * numChannels / CHANNELS_FETCH_SIZE
		if _, resp := c.Client.GetPublicChannelsForTeam(teamId, page, CHANNELS_FETCH_SIZE, ""); resp.Error != nil {
			mlog.Error("Failed to get public channels for team", mlog.String("team_id", teamId), mlog.Int("page", page), mlog.Err(resp.Error))
			return
		}

		// 30% chance of continuing to scroll to next page.
		if rand.Float64() > 0.30 {
			return
		}

		time.Sleep(time.Millisecond * 1000)
	}
}

var moreChannelsEntity UserEntity = UserEntity{
	Name: "MoreChannelsEntity",
	Actions: []randutil.Choice{
		{
			Item:   actionMoreChannels,
			Weight: 1,
		},
	},
}

var TestMoreChannelsBrowser TestRun = TestRun{
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
				Entity:         moreChannelsEntity,
				RateMultiplier: 1.0,
			},
			Weight: 30,
		},
	},
}

var autocompleterUserEntity UserEntity = UserEntity{
	Name: "AutocompleterUserEntity",
	Actions: []randutil.Choice{
		{
			Item:   actionSearchChannel,
			Weight: 5,
		},
		{
			Item:   actionAutocompleteChannel,
			Weight: 1,
		},
	},
}

var TestAutocomplete TestRun = TestRun{
	UserEntities: []randutil.Choice{
		{
			Item: UserEntityWithRateMultiplier{
				RateMultiplier: 1.0,
				Entity:         standardUserEntity,
			},
			Weight: 10.0,
		},
		{
			Item: UserEntityWithRateMultiplier{
				RateMultiplier: 1.0,
				Entity:         autocompleterUserEntity,
			},

			Weight: 90.0,
		},
	},
}

func actionWakeup(c *EntityConfig) {
	manifests, resp := c.Client.GetWebappPlugins()
	if resp.Error != nil {
		mlog.Error("Failed to get webapp plugins", mlog.Err(resp.Error))
		return
	}

	mlog.Debug("Found webapp plugins", mlog.Int("count", len(manifests)))
}
