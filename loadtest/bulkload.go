// Copyright (c) 2017 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package loadtest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"

	sqlx "github.com/jmoiron/sqlx"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"

	"github.com/icrowley/fake"
	"github.com/mattermost/mattermost-load-test/randutil"
	"github.com/mattermost/mattermost-server/mlog"
	"github.com/mattermost/mattermost-server/model"
)

type LoadtestEnviromentConfig struct {
	NumTeams           int
	NumChannelsPerTeam int
	NumUsers           int

	PercentHighVolumeChannels float64
	PercentMidVolumeChannels  float64
	PercentLowVolumeChannels  float64

	PercentUsersHighVolumeChannel float64
	PercentUsersMidVolumeChannel  float64
	PercentUsersLowVolumeChannel  float64

	PercentHighVolumeTeams float64
	PercentMidVolumeTeams  float64
	PercentLowVolumeTeams  float64

	PercentUsersHighVolumeTeams float64
	PercentUsersMidVolumeTeams  float64
	PercentUsersLowVolumeTeams  float64

	HighVolumeTeamSelectionWeight int
	MidVolumeTeamSelectionWeight  int
	LowVolumeTeamSelectionWeight  int

	HighVolumeChannelSelectionWeight int
	MidVolumeChannelSelectionWeight  int
	LowVolumeChannelSelectionWeight  int

	NumPosts      int
	PostTimeRange int64
	ReplyChance   float64
}

type LineImportData struct {
	Type    string             `json:"type"`
	Team    *TeamImportData    `json:"team,omitempty"`
	Channel *ChannelImportData `json:"channel,omitempty"`
	User    *UserImportData    `json:"user,omitempty"`
	Post    *PostImportData    `json:"post,omitempty"`
	Version int                `json:"version"`
}

type TeamImportData struct {
	Name            string `json:"name"`
	DisplayName     string `json:"display_name"`
	Type            string `json:"type"`
	Description     string `json:"description,omitempty"`
	AllowOpenInvite bool   `json:"allow_open_invite,omitempty"`
}

type ChannelImportData struct {
	Team        string `json:"team"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Type        string `json:"type"`
	Header      string `json:"header,omitempty"`
	Purpose     string `json:"purpose,omitempty"`
}

type UserImportData struct {
	Username    string `json:"username"`
	Email       string `json:"email"`
	AuthService string `json:"auth_service,omitempty"`
	AuthData    string `json:"auth_data,omitempty"`
	Password    string `json:"password,omitempty"`
	Nickname    string `json:"nickname,omitempty"`
	FirstName   string `json:"first_name,omitempty"`
	LastName    string `json:"last_name,omitempty"`
	Position    string `json:"position,omitempty"`
	Roles       string `json:"roles"`
	Locale      string `json:"locale,omitempty"`

	Teams      []UserTeamImportData `json:"teams"`
	TeamChoice []randutil.Choice    `json:"-"`

	Theme              string `json:"theme,omitempty"`
	SelectedFont       string `json:"display_font,omitempty"`
	UseMilitaryTime    string `json:"military_time,omitempty"`
	NameFormat         string `json:"teammate_name_display,omitempty"`
	CollapsePreviews   string `json:"link_previews,omitempty"`
	MessageDisplay     string `json:"message_display,omitempty"`
	ChannelDisplayMode string `json:"channel_display_mode,omitempty"`
}

type UserTeamImportData struct {
	Name          string                  `json:"name"`
	Roles         string                  `json:"roles"`
	Channels      []UserChannelImportData `json:"channels"`
	ChannelChoice []randutil.Choice       `json:"-"`
}

func (team *UserTeamImportData) PickChannel() *UserChannelImportData {
	if len(team.ChannelChoice) == 0 {
		return nil
	}
	item2, err2 := randutil.WeightedChoice(team.ChannelChoice)
	if err2 != nil {
		panic(err2)
	}
	channelIndex := item2.Item.(int)
	channel := &team.Channels[channelIndex]

	return channel
}

type UserChannelImportData struct {
	Name  string `json:"name"`
	Roles string `json:"roles"`
}

type VersionImportData struct {
	Type    string `json:"type"`
	Version string `json:"version"`
}

type PostImportData struct {
	Team    *string `json:"team"`
	Channel *string `json:"channel"`
	User    *string `json:"user"`

	Message  *string `json:"message"`
	CreateAt *int64  `json:"create_at"`

	FlaggedBy *[]string `json:"flagged_by"`
}

type GenerateBulkloadFileResult struct {
	File     bytes.Buffer
	Users    []UserImportData
	Teams    []TeamImportData
	Channels []ChannelImportData
}

func (s *UserImportData) PickTeam() *UserTeamImportData {
	if len(s.TeamChoice) == 0 {
		return nil
	}
	item, err := randutil.WeightedChoice(s.TeamChoice)
	if err != nil {
		panic(err)
	}
	teamIndex := item.Item.(int)
	team := &s.Teams[teamIndex]

	return team
}

func (s *UserImportData) PickTeamChannel() (*UserTeamImportData, *UserChannelImportData) {
	team := s.PickTeam()
	if team == nil {
		return nil, nil
	}

	return team, team.PickChannel()
}

func generateTeams(numTeams int) []TeamImportData {
	teams := make([]TeamImportData, 0, numTeams)

	for teamNum := 0; teamNum < numTeams; teamNum++ {
		teams = append(teams, TeamImportData{
			Name:            "loadtestteam" + strconv.Itoa(teamNum),
			DisplayName:     "Loadtest Team " + strconv.Itoa(teamNum),
			Type:            "O",
			Description:     "This is loadtest team " + strconv.Itoa(teamNum),
			AllowOpenInvite: true,
		})
	}

	return teams
}

func GenerateBulkloadFile(config *LoadtestEnviromentConfig) GenerateBulkloadFileResult {
	users := make([]UserImportData, 0, config.NumUsers)
	channels := make([]ChannelImportData, 0, config.NumChannelsPerTeam*config.NumTeams)

	teams := generateTeams(config.NumTeams)

	channelsByTeam := make([][]int, 0, config.NumChannelsPerTeam*config.NumTeams)

	for teamNum := 0; teamNum < config.NumTeams; teamNum++ {
		channelsByTeam = append(channelsByTeam, make([]int, 0, config.NumChannelsPerTeam))
		for channelNum := 0; channelNum < config.NumChannelsPerTeam; channelNum++ {
			channels = append(channels, ChannelImportData{
				Team:        "loadtestteam" + strconv.Itoa(teamNum),
				Name:        "loadtestchannel" + strconv.Itoa(channelNum),
				DisplayName: "Loadtest Channel " + strconv.Itoa(channelNum),
				Type:        "O",
				Header:      "Hea: This is loadtest channel " + strconv.Itoa(teamNum) + " on team " + strconv.Itoa(teamNum),
				Purpose:     "Pur: This is loadtest channel " + strconv.Itoa(teamNum) + " on team " + strconv.Itoa(teamNum),
			})
			channelsByTeam[teamNum] = append(channelsByTeam[teamNum], len(channels)-1)
		}
	}

	for userNum := 0; userNum < config.NumUsers; userNum++ {
		users = append(users, UserImportData{
			Username: "user" + strconv.Itoa(userNum),
			Roles:    "system_user",
			Email:    "success+user" + strconv.Itoa(userNum) + "@simulator.amazonses.com",
			Password: "Loadtestpassword1",
		})
	}

	numHighVolumeTeams := int(math.Floor(float64(config.NumTeams) * config.PercentHighVolumeTeams))
	numMidVolumeTeams := int(math.Floor(float64(config.NumTeams) * config.PercentMidVolumeTeams))

	numUsersInHighVolumeTeam := int(math.Floor(float64(config.NumUsers) * config.PercentUsersHighVolumeTeams))
	numUsersInMidVolumeTeam := int(math.Floor(float64(config.NumUsers) * config.PercentUsersMidVolumeTeams))
	numUsersInLowVolumeTeam := int(math.Floor(float64(config.NumUsers) * config.PercentUsersLowVolumeTeams))

	r := rand.New(rand.NewSource(29))

	teamPermutation := r.Perm(len(teams))
	for sequenceNum, teamNum := range teamPermutation {
		currentTeam := teams[teamNum]
		channelsInTeam := channelsByTeam[teamNum]

		numUsersToAdd := numUsersInHighVolumeTeam
		selectWeight := config.HighVolumeTeamSelectionWeight
		if sequenceNum > numHighVolumeTeams {
			numUsersToAdd = numUsersInMidVolumeTeam
			selectWeight = config.MidVolumeTeamSelectionWeight
		}
		if sequenceNum > (numHighVolumeTeams + numMidVolumeTeams) {
			numUsersToAdd = numUsersInLowVolumeTeam
			selectWeight = config.LowVolumeTeamSelectionWeight
		}

		userPermutation := r.Perm(len(users))
		usersInTeam := make([]int, 0, numUsersToAdd)
		for userNum := 0; userNum < numUsersToAdd; userNum++ {
			userTeamImportData := &UserTeamImportData{
				Name:  currentTeam.Name,
				Roles: "team_user",
			}
			users[userPermutation[userNum]].Teams = append(users[userPermutation[userNum]].Teams, *userTeamImportData)
			users[userPermutation[userNum]].TeamChoice = append(users[userPermutation[userNum]].TeamChoice, randutil.Choice{
				Item:   len(users[userPermutation[userNum]].Teams) - 1,
				Weight: selectWeight,
			})
			usersInTeam = append(usersInTeam, userPermutation[userNum])

			userTownSquareImportData := &UserChannelImportData{
				Name:  "town-square",
				Roles: "channel_user",
			}

			permutation := userPermutation[userNum]
			users[permutation].Teams[len(users[permutation].Teams)-1].Channels = append(users[permutation].Teams[len(users[permutation].Teams)-1].Channels, *userTownSquareImportData)
		}

		numHighVolumeChannels := int(math.Floor(float64(len(channelsInTeam)) * config.PercentHighVolumeChannels))
		numMidVolumeChannels := int(math.Floor(float64(len(channelsInTeam)) * config.PercentMidVolumeChannels))

		numUsersInHighVolumeChannel := int(math.Floor(float64(numUsersToAdd) * config.PercentUsersHighVolumeChannel))
		numUsersInMidVolumeChannel := int(math.Floor(float64(numUsersToAdd) * config.PercentUsersMidVolumeChannel))
		numUsersInLowVolumeChannel := int(math.Floor(float64(numUsersToAdd) * config.PercentUsersLowVolumeChannel))

		for channelSequenceNum, channelNum := range channelsInTeam {
			channel := channels[channelNum]
			numUsersToAddChannel := numUsersInHighVolumeChannel
			selectWeightChannel := config.HighVolumeChannelSelectionWeight

			if channelSequenceNum > numHighVolumeChannels {
				numUsersToAddChannel = numUsersInMidVolumeChannel
				selectWeightChannel = config.MidVolumeChannelSelectionWeight
			}
			if channelSequenceNum > (numHighVolumeChannels + numMidVolumeChannels) {
				numUsersToAddChannel = numUsersInLowVolumeChannel
				selectWeightChannel = config.LowVolumeChannelSelectionWeight
			}
			if numUsersToAddChannel > len(usersInTeam) {
				numUsersToAddChannel = len(usersInTeam)
			}
			usersInTeamPermutation := r.Perm(len(usersInTeam))
			for userInTeamNum := 0; userInTeamNum < numUsersToAddChannel; userInTeamNum++ {
				userNum := usersInTeam[usersInTeamPermutation[userInTeamNum]]
				userChannelImportData := &UserChannelImportData{
					Name:  channel.Name,
					Roles: "channel_user",
				}
				users[userNum].Teams[len(users[userNum].Teams)-1].Channels = append(users[userNum].Teams[len(users[userNum].Teams)-1].Channels, *userChannelImportData)
				users[userNum].Teams[len(users[userNum].Teams)-1].ChannelChoice = append(users[userNum].Teams[len(users[userNum].Teams)-1].ChannelChoice, randutil.Choice{
					Item:   len(users[userNum].Teams[len(users[userNum].Teams)-1].Channels) - 1,
					Weight: selectWeightChannel,
				})
			}

		}
	}

	lineObjectsChan := make(chan *LineImportData, 100)
	doneChan := make(chan struct{})

	var output bytes.Buffer

	/*f, err := os.OpenFile("loadtestbulkload.json", os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		fmt.Println("Problem opening file: " + err.Error())
	}*/
	go func() {
		jenc := json.NewEncoder(&output)
		for lineObject := range lineObjectsChan {
			if err := jenc.Encode(lineObject); err != nil {
				fmt.Println("Probablem marshaling: " + err.Error())
			}
		}
		close(doneChan)
	}()

	version := LineImportData{
		Type:    "version",
		Version: 1,
	}
	lineObjectsChan <- &version

	// Convert all the objects to line objects
	for i := range teams {
		lineObjectsChan <- &LineImportData{
			Type:    "team",
			Team:    &teams[i],
			Version: 1,
		}
	}

	for i := range channels {
		lineObjectsChan <- &LineImportData{
			Type:    "channel",
			Channel: &channels[i],
			Version: 1,
		}
	}

	for i := range users {
		lineObjectsChan <- &LineImportData{
			Type:    "user",
			User:    &users[i],
			Version: 1,
		}
	}

	close(lineObjectsChan)
	<-doneChan

	return GenerateBulkloadFileResult{
		File:     output,
		Users:    users,
		Teams:    teams,
		Channels: channels,
	}
}

func ConnectToDB(driverName, dataSource string) *sqlx.DB {
	db, err := sqlx.Open(driverName, dataSource)
	if err != nil {
		fmt.Println("Unable to open database: " + err.Error())
		return nil
	}
	if err := db.Ping(); err != nil {
		fmt.Println("Unable to ping DB: " + err.Error())
		return nil
	}

	return db
}

func LoadPosts(cfg *LoadTestConfig, driverName, dataSource string) {
	mlog.Info("Loading posts")
	db := ConnectToDB(driverName, dataSource)
	if db == nil {
		return
	}

	adminClient := model.NewAPIv4Client(cfg.ConnectionConfiguration.ServerURL)
	if _, resp := adminClient.Login(cfg.ConnectionConfiguration.AdminEmail, cfg.ConnectionConfiguration.AdminPassword); resp.Error != nil {
		mlog.Error("Unable to login as admin for loadposts", mlog.Err(resp.Error))
	}

	teams, resp := adminClient.GetAllTeams("", 0, cfg.LoadtestEnviromentConfig.NumTeams+200)
	if resp.Error != nil {
		mlog.Error("Unable to get all theams", mlog.Err(resp.Error))
		return
	}

	numPostsPerChannel := int(math.Floor(float64(cfg.LoadtestEnviromentConfig.NumPosts) / float64(cfg.LoadtestEnviromentConfig.NumTeams*cfg.LoadtestEnviromentConfig.NumChannelsPerTeam)))

	statementStr := "INSERT INTO Posts (Id, CreateAt, UpdateAt, EditAt, DeleteAt, IsPinned, UserId, ChannelId, RootId, ParentId, OriginalId, Message, Type, Props, Hashtags, Filenames, FileIds, HasReactions) VALUES "
	for i := 0; i < 100; i++ {
		statementStr += "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?),"
	}
	statementStr = statementStr[0 : len(statementStr)-1]
	statement, err := db.Prepare(db.Rebind(statementStr))
	if err != nil {
		mlog.Error("Unable to prepare statment", mlog.String("statement", statementStr), mlog.Err(err))
		return
	}

	// Generate a random time before now, either within the configured post time range or after the given time.
	randomTime := func(after *time.Time) time.Time {
		now := time.Now()
		if after == nil {
			return now.Add(-1 * time.Duration(rand.Int63n(cfg.LoadtestEnviromentConfig.PostTimeRange)) * time.Second)
		} else {
			return after.Add(time.Duration(rand.Int63n(int64(now.Sub(*after)))))
		}
	}

	mlog.Info("Done pre-setup")
	for _, team := range teams {
		if !strings.HasPrefix(team.Name, "loadtestteam") {
			continue
		}

		mlog.Info("Grabbing channels", mlog.String("team", team.Name))
		channels := make([]*model.Channel, 0)
		numReceived := 200
		for page := 0; numReceived == 200; page++ {
			if newchannels, resp2 := adminClient.GetPublicChannelsForTeam(team.Id, page, 200, ""); resp2.Error != nil {
				mlog.Error("Could not get public channels.", mlog.String("team_id", team.Id), mlog.Err(resp2.Error))
				return
			} else {
				numReceived = len(newchannels)
				channels = append(channels, newchannels...)
			}
		}

		mlog.Info("Grabbing users", mlog.String("team", team.Name))
		users := make([]*model.User, 0)
		numReceived = 200
		total := 0
		for page := 0; numReceived == 200; page++ {
			mlog.Info(fmt.Sprintf("Page %d", page))
			if newusers, resp2 := adminClient.GetUsersInTeam(team.Id, page, 200, ""); resp2.Error != nil {
				mlog.Error("Could not get user.", mlog.String("team_id", team.Id), mlog.Err(resp2.Error))
				return
			} else if numReceived = len(newusers); numReceived > 0 {
				total += numReceived
				mlog.Info(fmt.Sprintf("User %v", newusers[0].Username))
				users = append(users, newusers...)

				if total >= 10000 {
					break
				}
			}
		}

		mlog.Info("Thread splitting", mlog.String("team", team.Name))
		ThreadSplit(len(channels), 16, PrintCounter, func(channelNum int) {
			mlog.Info("Thread", mlog.Int("channel_num", channelNum))
			// Only recognizes multiples of 100
			type rootPost struct {
				id      string
				created time.Time
			}
			rootPosts := make([]rootPost, 0)
			for i := 0; i < numPostsPerChannel; i += 100 {
				vals := []interface{}{}

				for j := 0; j < 100; j++ {
					message := "PL" + fake.SentencesN(1)
					now := randomTime(nil)
					id := model.NewId()
					zero := 0
					emptyobject := "{}"
					emptyarray := "[]"

					parentRoot := ""
					if j == 0 {
						rootPosts = append(rootPosts, rootPost{id, now})
					} else {
						if rand.Float64() < cfg.LoadtestEnviromentConfig.ReplyChance {
							rootPost := rootPosts[rand.Intn(len(rootPosts))]
							parentRoot = rootPost.id
							now = randomTime(&rootPost.created)
						}
					}

					vals = append(vals, id, now.Unix()*1000, now.Unix()*1000, zero, zero, zero, users[(j+i+channelNum)%len(users)].Id, channels[channelNum].Id, parentRoot, parentRoot, "", message, "", emptyobject, "", emptyarray, emptyarray, zero)
				}

				if _, err := statement.Exec(vals...); err != nil {
					mlog.Error("Error running statement", mlog.Err(err))
				}
			}
		})
	}
	statement.Close()
}
