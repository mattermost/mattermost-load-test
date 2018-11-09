// Copyright (c) 2017 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package loadtest

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"
	sqlx "github.com/jmoiron/sqlx"

	_ "github.com/lib/pq"

	"github.com/icrowley/fake"
	"github.com/mattermost/mattermost-load-test/randutil"
	"github.com/mattermost/mattermost-server/mlog"
	"github.com/mattermost/mattermost-server/model"
)

const (
	DEFAULT_PERMISSIONS_TEAM_ADMIN    = "edit_others_posts remove_user_from_team manage_team import_team manage_team_roles manage_channel_roles manage_others_webhooks manage_slash_commands manage_others_slash_commands manage_webhooks delete_post delete_others_posts"
	DEFAULT_PERMISSIONS_TEAM_USER     = "list_team_channels join_public_channels read_public_channel view_team create_public_channel manage_public_channel_properties delete_public_channel create_private_channel manage_private_channel_properties delete_private_channel invite_user add_user_to_team"
	DEFAULT_PERMISSIONS_CHANNEL_ADMIN = "manage_channel_roles"
	DEFAULT_PERMISSIONS_CHANNEL_USER  = "read_channel add_reaction remove_reaction manage_public_channel_members upload_file get_public_link create_post use_slash_commands manage_private_channel_members delete_post edit_post"
)

type LoadtestEnviromentConfig struct {
	NumTeams                  int
	NumChannelsPerTeam        int
	NumPrivateChannelsPerTeam int
	NumDirectMessageChannels  int
	NumGroupMessageChannels   int
	NumUsers                  int
	NumTeamSchemes            int
	NumChannelSchemes         int
	NumEmoji                  int
	NumPlugins                int

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

	PercentCustomSchemeTeams    float64
	PercentCustomSchemeChannels float64
}

type LineImportData struct {
	Type          string                   `json:"type"`
	Scheme        *SchemeImportData        `json:"scheme,omitempty"`
	Team          *TeamImportData          `json:"team,omitempty"`
	Channel       *ChannelImportData       `json:"channel,omitempty"`
	User          *UserImportData          `json:"user,omitempty"`
	Post          *PostImportData          `json:"post,omitempty"`
	DirectChannel *DirectChannelImportData `json:"direct_channel,omitempty"`
	Emoji         *EmojiImportData         `json:"emoji,omitempty"`
	Version       int                      `json:"version"`
}

type TeamImportData struct {
	Name            string `json:"name"`
	DisplayName     string `json:"display_name"`
	Type            string `json:"type"`
	Description     string `json:"description,omitempty"`
	AllowOpenInvite bool   `json:"allow_open_invite,omitempty"`
	Scheme          string `json:"scheme,omitempty"`
}

type ChannelImportData struct {
	Team        string `json:"team"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Type        string `json:"type"`
	Header      string `json:"header,omitempty"`
	Purpose     string `json:"purpose,omitempty"`
	Scheme      string `json:"scheme,omitempty"`
}

type DirectChannelImportData struct {
	Members     *[]string `json:"members"`
	FavoritedBy *[]string `json:"favorited_by"`

	Header *string `json:"header"`
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

type SchemeImportData struct {
	Name                    string          `json:"name"`
	DisplayName             string          `json:"display_name"`
	Description             string          `json:"description"`
	Scope                   string          `json:"scope"`
	DefaultTeamAdminRole    *RoleImportData `json:"default_team_admin_role,omitempty"`
	DefaultTeamUserRole     *RoleImportData `json:"default_team_user_role,omitempty"`
	DefaultChannelAdminRole *RoleImportData `json:"default_channel_admin_role,omitempty"`
	DefaultChannelUserRole  *RoleImportData `json:"default_channel_user_role,omitempty"`
}

type RoleImportData struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

type EmojiImportData struct {
	Name  string `json:"name"`
	Image string `json:"image"`
}

type GenerateBulkloadFileResult struct {
	File     bytes.Buffer
	Users    []UserImportData
	Teams    []TeamImportData
	Channels []ChannelImportData
	Schemes  []SchemeImportData
	Emojis   []EmojiImportData
}

func (c *LoadtestEnviromentConfig) PickEmoji(r *rand.Rand) string {
	return fmt.Sprintf("loadtestemoji%v", r.Intn(c.NumEmoji-1))
}

func (s *UserImportData) PickTeam(r *rand.Rand) *UserTeamImportData {
	if len(s.TeamChoice) == 0 {
		return nil
	}
	item, err := randutil.WeightedChoice(r, s.TeamChoice)
	if err != nil {
		panic(err)
	}
	teamIndex := item.Item.(int)
	team := &s.Teams[teamIndex]

	return team
}

func (s *UserImportData) PickTeamChannel(r *rand.Rand) (*UserTeamImportData, *UserChannelImportData) {
	team := s.PickTeam(r)
	if team == nil {
		return nil, nil
	}

	return team, team.PickChannel(r)
}

func (team *UserTeamImportData) PickChannel(r *rand.Rand) *UserChannelImportData {
	if len(team.ChannelChoice) == 0 {
		return nil
	}
	item2, err2 := randutil.WeightedChoice(r, team.ChannelChoice)
	if err2 != nil {
		panic(err2)
	}
	channelIndex := item2.Item.(int)
	channel := &team.Channels[channelIndex]

	return channel
}

func generateTeams(numTeams int, r *rand.Rand, percentCustomSchemeTeams float64, teamSchemes *[]SchemeImportData) []TeamImportData {
	teams := make([]TeamImportData, 0, numTeams)

	scheme := ""
	if len(*teamSchemes) > 0 && r.Float64() < percentCustomSchemeTeams {
		scheme = (*teamSchemes)[r.Intn(len(*teamSchemes))].Name
	}

	for teamNum := 0; teamNum < numTeams; teamNum++ {
		teams = append(teams, TeamImportData{
			Name:            "loadtestteam" + strconv.Itoa(teamNum),
			DisplayName:     "Loadtest Team " + strconv.Itoa(teamNum),
			Type:            model.TEAM_OPEN,
			Description:     "This is loadtest team " + strconv.Itoa(teamNum),
			AllowOpenInvite: true,
			Scheme:          scheme,
		})
	}

	return teams
}

func generateTeamSchemes(numSchemes int) *[]SchemeImportData {
	teamSchemes := make([]SchemeImportData, 0, numSchemes)

	for schemeNum := 0; schemeNum < numSchemes; schemeNum++ {
		teamSchemes = append(teamSchemes, SchemeImportData{
			Name:        "loadtestteamscheme" + strconv.Itoa(schemeNum),
			DisplayName: "Loadtest Team Scheme " + strconv.Itoa(schemeNum),
			Scope:       "team", // model.SCHEME_SCOPE_TEAM
			DefaultTeamAdminRole: &RoleImportData{
				Name:        "loadtest_tsta_role_" + strconv.Itoa(schemeNum),
				DisplayName: "Loadtest Team Scheme DTA Role " + strconv.Itoa(schemeNum),
				Permissions: strings.Fields(DEFAULT_PERMISSIONS_TEAM_ADMIN),
			},
			DefaultTeamUserRole: &RoleImportData{
				Name:        "loadtest_tstu_role_" + strconv.Itoa(schemeNum),
				DisplayName: "Loadtest Team Scheme DTU Role " + strconv.Itoa(schemeNum),
				Permissions: strings.Fields(DEFAULT_PERMISSIONS_TEAM_USER),
			},
			DefaultChannelAdminRole: &RoleImportData{
				Name:        "loadtest_tsca_role_" + strconv.Itoa(schemeNum),
				DisplayName: "Loadtest Team Scheme DCA Role " + strconv.Itoa(schemeNum),
				Permissions: strings.Fields(DEFAULT_PERMISSIONS_CHANNEL_ADMIN),
			},
			DefaultChannelUserRole: &RoleImportData{
				Name:        "loadtest_tscu_role_" + strconv.Itoa(schemeNum),
				DisplayName: "Loadtest Team Scheme DCU Role " + strconv.Itoa(schemeNum),
				Permissions: strings.Fields(DEFAULT_PERMISSIONS_CHANNEL_USER),
			},
		})
	}

	return &teamSchemes
}

func generateChannelSchemes(numSchemes int) *[]SchemeImportData {
	channelSchemes := make([]SchemeImportData, 0, numSchemes)

	for schemeNum := 0; schemeNum < numSchemes; schemeNum++ {
		channelSchemes = append(channelSchemes, SchemeImportData{
			Name:        "loadtestchannelscheme" + strconv.Itoa(schemeNum),
			DisplayName: "Loadtest Channel Scheme " + strconv.Itoa(schemeNum),
			Scope:       "channel", // model.SCHEME_SCOPE_CHANNEL
			DefaultChannelAdminRole: &RoleImportData{
				Name:        "loadtest_csca_role_" + strconv.Itoa(schemeNum),
				DisplayName: "Loadtest Channel Scheme DCA Role " + strconv.Itoa(schemeNum),
				Permissions: strings.Fields(DEFAULT_PERMISSIONS_CHANNEL_ADMIN),
			},
			DefaultChannelUserRole: &RoleImportData{
				Name:        "loadtest_cscu_role_" + strconv.Itoa(schemeNum),
				DisplayName: "Loadtest Channel Scheme DCU Role " + strconv.Itoa(schemeNum),
				Permissions: strings.Fields(DEFAULT_PERMISSIONS_CHANNEL_USER),
			},
		})
	}

	return &channelSchemes
}

func generateEmoji(numEmoji int) []EmojiImportData {
	emojis := make([]EmojiImportData, 0, numEmoji)

	for emojiNum := 0; emojiNum < numEmoji; emojiNum++ {
		emojis = append(emojis, EmojiImportData{
			Name:  "loadtestemoji" + strconv.Itoa(emojiNum),
			Image: "./testfiles/test_emoji.png",
		})
	}

	return emojis
}

func makeChannelName(channelNumber int) string {
	return strings.Join(strings.Fields(fake.WordsN(2)), "-") + "-loadtestchannel" + strconv.Itoa(channelNumber)
}

func makeChannelDisplayName(channelNumber int) string {
	return fake.WordsN(2) + " Loadtest Channel " + strconv.Itoa(channelNumber)
}

func makeChannelHeader() string {
	return fake.Sentence()
}

func makeChannelPurpose() string {
	return fake.Sentence()
}

// makeDirectChannels creates the requested number of direct message channels.
func makeDirectChannels(config *LoadtestEnviromentConfig, r *rand.Rand, users []UserImportData) []DirectChannelImportData {
	// Constrain the requested number of direct message channels to the maximum possible given
	// the number of users.
	numDirectMessageChannels := config.NumDirectMessageChannels
	maxDirectMessageChannels := int(len(users) / 2 * (len(users) - 1))
	if numDirectMessageChannels > maxDirectMessageChannels {
		mlog.Warn("Insufficient users to generate direct message channels.", mlog.Int("requested", numDirectMessageChannels), mlog.Int("cap", maxDirectMessageChannels))
		numDirectMessageChannels = maxDirectMessageChannels
	}

	directChannels := make([]DirectChannelImportData, 0, numDirectMessageChannels)

	directMessageChannelsPermutation := r.Perm(numDirectMessageChannels)
	for k := range directMessageChannelsPermutation {
		// https://stackoverflow.com/questions/27086195/linear-index-upper-triangular-matrix
		n := len(users)
		i := n - 2 - int(math.Floor(math.Sqrt(float64(-8*k+4*n*(n-1)-7))/2.0-0.5))
		j := k + i + 1 - n*(n-1)/2 + (n-i)*((n-i)-1)/2

		header := makeChannelHeader()
		members := []string{
			users[i].Username,
			users[j].Username,
		}

		directChannels = append(directChannels,
			DirectChannelImportData{
				Members: &members,
				Header:  &header,
			},
		)
	}

	return directChannels
}

// makeGroupChannels attempts to create the requested number of group message channels.
//
// Note that the implementation of this method is more random than makeDirectChannels and can fail
// to generate sufficient group message channels if there aren't enough users. In practice, the
// number of group messages is relatively low compared to direct messages, and the algorithm for
// generating same is simpler when random.
func makeGroupChannels(config *LoadtestEnviromentConfig, r *rand.Rand, users []UserImportData) []DirectChannelImportData {
	groupChannels := make([]DirectChannelImportData, 0, config.NumGroupMessageChannels)
	usernames := make([]string, 0, len(users))
	for _, user := range users {
		usernames = append(usernames, user.Username)
	}

	seenGroupMessageChannels := make(map[string]bool)

	pickRandomUsernames := func(count int) ([]string, error) {
		maxAttempts := 3
		for attempts := maxAttempts; attempts > 0; attempts-- {
			currentLength := len(usernames)
			randomUsernames := make([]string, 0, count)

			for i := 0; i < count; i++ {
				if currentLength == 0 {
					return nil, fmt.Errorf("ran out of usernames picking %d random usernames", count)
				}

				index := r.Intn(currentLength)
				randomUsernames = append(randomUsernames, usernames[index])

				// Eliminate the selected username from the next random selection.
				usernames[index], usernames[currentLength-1] = usernames[currentLength-1], usernames[index]
				currentLength--
			}

			// Keep track of the random usernames seen to avoid generating the same
			// group message again.
			sort.Strings(randomUsernames)
			if seenGroupMessageChannels[strings.Join(randomUsernames, ",")] {
				// Try again with a different random selection.
				continue
			} else {
				seenGroupMessageChannels[strings.Join(randomUsernames, ",")] = true
			}

			return randomUsernames, nil
		}

		return nil, fmt.Errorf("failed to generate unique group message channel after %d tries", maxAttempts)
	}

	for i := 0; i < config.NumGroupMessageChannels; i++ {
		// Generate at least 3 members, but randomly generate larger group sizes.
		count := 3
		for count < model.CHANNEL_GROUP_MAX_USERS && r.Float32() < 0.05 {
			count++
		}

		header := makeChannelHeader()
		members, err := pickRandomUsernames(count)
		if err != nil {
			mlog.Warn("failed to pick random usernames for group message channel", mlog.Int("count_group_message_channels", len(groupChannels)), mlog.Err(err))
			break
		}

		groupChannels = append(groupChannels,
			DirectChannelImportData{
				Members: &members,
				Header:  &header,
			},
		)
	}

	return groupChannels
}

func GenerateBulkloadFile(config *LoadtestEnviromentConfig) GenerateBulkloadFileResult {
	r := rand.New(rand.NewSource(29))
	fake.Seed(r.Int63())

	users := make([]UserImportData, 0, config.NumUsers)

	totalChannelsPerTeam := config.NumChannelsPerTeam + config.NumPrivateChannelsPerTeam
	channels := make([]ChannelImportData, 0, totalChannelsPerTeam*config.NumTeams)

	teamSchemes := generateTeamSchemes(config.NumTeamSchemes)
	teams := generateTeams(config.NumTeams, r, config.PercentCustomSchemeTeams, teamSchemes)

	channelSchemes := generateChannelSchemes(config.NumChannelSchemes)

	channelsByTeam := make([][]int, 0, config.NumChannelsPerTeam*config.NumTeams)

	emojis := generateEmoji(config.NumEmoji)

	for teamNum := 0; teamNum < config.NumTeams; teamNum++ {
		teamName := "loadtestteam" + strconv.Itoa(teamNum)
		channelsByTeam = append(channelsByTeam, make([]int, 0, totalChannelsPerTeam))

		for channelNum := 0; channelNum < totalChannelsPerTeam; channelNum++ {
			scheme := ""
			if len(*channelSchemes) > 0 && r.Float64() < config.PercentCustomSchemeChannels {
				scheme = (*channelSchemes)[r.Intn(len(*channelSchemes))].Name
			}

			channelType := model.CHANNEL_OPEN
			if channelNum >= config.NumChannelsPerTeam {
				channelType = model.CHANNEL_PRIVATE
			}

			channels = append(channels, ChannelImportData{
				Team:        teamName,
				Name:        makeChannelName(channelNum),
				DisplayName: makeChannelDisplayName(channelNum),
				Type:        channelType,
				Header:      makeChannelHeader(),
				Purpose:     makeChannelPurpose(),
				Scheme:      scheme,
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
				Name: currentTeam.Name,
			}
			users[userPermutation[userNum]].Teams = append(users[userPermutation[userNum]].Teams, *userTeamImportData)
			users[userPermutation[userNum]].TeamChoice = append(users[userPermutation[userNum]].TeamChoice, randutil.Choice{
				Item:   len(users[userPermutation[userNum]].Teams) - 1,
				Weight: selectWeight,
			})
			usersInTeam = append(usersInTeam, userPermutation[userNum])
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
					Name: channel.Name,
				}
				users[userNum].Teams[len(users[userNum].Teams)-1].Channels = append(users[userNum].Teams[len(users[userNum].Teams)-1].Channels, *userChannelImportData)
				users[userNum].Teams[len(users[userNum].Teams)-1].ChannelChoice = append(users[userNum].Teams[len(users[userNum].Teams)-1].ChannelChoice, randutil.Choice{
					Item:   len(users[userNum].Teams[len(users[userNum].Teams)-1].Channels) - 1,
					Weight: selectWeightChannel,
				})
			}

		}
	}

	directChannels := makeDirectChannels(config, r, users)
	directChannels = append(directChannels, makeGroupChannels(config, r, users)...)

	lineObjectsChan := make(chan *LineImportData, 100)
	doneChan := make(chan struct{})

	var output bytes.Buffer
	go func() {
		jenc := json.NewEncoder(&output)
		for lineObject := range lineObjectsChan {
			if err := jenc.Encode(lineObject); err != nil {
				fmt.Println("Problem marshaling: " + err.Error())
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
	for i := range *teamSchemes {
		lineObjectsChan <- &LineImportData{
			Type:    "scheme",
			Scheme:  &(*teamSchemes)[i],
			Version: 1,
		}
	}

	for i := range *channelSchemes {
		lineObjectsChan <- &LineImportData{
			Type:    "scheme",
			Scheme:  &(*channelSchemes)[i],
			Version: 1,
		}
	}

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

	usersTeams := make([][]UserTeamImportData, len(users))
	for i := range users {
		// The Mattermost server currently chokes on users with a large number of channel
		// memberships, but only because the bufio.Scanner has a fixed-size buffer and
		// fails on longer lines. As a temporary workaround that requires no server
		// changes, we chunk the channel membership data across multiple, "duplicate"
		// users. The caveat is that lines are loaded in parallel, which can result in
		// import errors if the user has not yet been created. Avoid this by sending a
		// "different type" of import line between users and memberships.

		usersTeams[i] = users[i].Teams
		users[i].Teams = make([]UserTeamImportData, 0, len(usersTeams[i]))
		for _, userTeam := range usersTeams[i] {
			users[i].Teams = append(users[i].Teams, UserTeamImportData{
				Name:  userTeam.Name,
				Roles: userTeam.Roles,
			})
		}

		// Output a version of the user without any channel memberships.
		lineObjectsChan <- &LineImportData{
			Type:    "user",
			User:    &users[i],
			Version: 1,
		}
	}

	for i := range emojis {
		lineObjectsChan <- &LineImportData{
			Type:    "emoji",
			Emoji:   &emojis[i],
			Version: 1,
		}
	}

	for i := range users {
		// See chunking strategy above.
		users[i].Teams = usersTeams[i]

		for _, userTeam := range users[i].Teams {
			channels := []UserChannelImportData{}

			sendChannelsChunk := func(channels []UserChannelImportData) {
				teamChannels := make([]UserChannelImportData, len(channels))
				copy(teamChannels, channels)

				user := UserImportData{
					Username: users[i].Username,
					Email:    users[i].Email,
					Roles:    users[i].Roles,
					Password: users[i].Password,

					Teams: []UserTeamImportData{
						{
							Name:     userTeam.Name,
							Roles:    userTeam.Roles,
							Channels: teamChannels,
						},
					},
				}

				lineObjectsChan <- &LineImportData{
					Type:    "user",
					User:    &user,
					Version: 1,
				}
			}

			for _, userTeamChannel := range userTeam.Channels {
				channels = append(channels, userTeamChannel)
				if len(channels) >= 100 {
					sendChannelsChunk(channels)
					channels = channels[:0]
				}
			}
			if len(channels) > 0 {
				sendChannelsChunk(channels)
				channels = channels[:0]
			}
		}
	}

	for i := range directChannels {
		lineObjectsChan <- &LineImportData{
			Type:          "direct_channel",
			DirectChannel: &directChannels[i],
			Version:       1,
		}
	}

	close(lineObjectsChan)
	<-doneChan

	return GenerateBulkloadFileResult{
		File:     output,
		Users:    users,
		Teams:    teams,
		Channels: channels,
		Emojis:   emojis,
	}
}

func ConnectToDB(driverName, dataSource string) *sqlx.DB {
	url, err := url.Parse(dataSource)
	if err != nil {
		fmt.Println("Unable to parse datasource: " + err.Error())
		return nil
	}
	if driverName == "mysql" {
		url.RawQuery = "charset=utf8mb4,utf8"
	}
	db, err := sqlx.Open(driverName, url.String())
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
	fake.Seed(0)

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

		if _, err := db.Exec("SET autocommit=0,unique_checks=0,foreign_key_checks=0"); err != nil {
			mlog.Error("Couldn't set temporary values for performance.", mlog.Err(err))
		}
		defer func() {
			if _, err := db.Exec("SET autocommit=1,unique_checks=1,foreign_key_checks=1"); err != nil {
				mlog.Critical("Couldn't set temporary values back to defaults.", mlog.Err(err))
				return
			}
		}()

		csvLines := make(chan []string)
		var wg sync.WaitGroup
		for i := 0; i < 4; i++ {
			wg.Add(1)
			go func() {
				importCSVToSQL(csvLines, db)
				wg.Done()
			}()
		}

		mlog.Info("Thread splitting", mlog.String("team", team.Name))
		ThreadSplit(len(channels), 4, func(channelNum int) {
			mlog.Info("Thread", mlog.Int("channel_num", channelNum))
			// Only recognizes multiples of 100
			type rootPost struct {
				id      string
				created time.Time
			}
			rootPosts := make([]rootPost, 0)
			for i := 0; i < numPostsPerChannel; i += 100 {
				for j := 0; j < 100; j++ {
					message := "PL" + fake.SentencesN(1)
					now := randomTime(nil)
					id := model.NewId()
					zero := "0"
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

					results := make([]string, 0, 19)
					results = append(results, id, fmt.Sprint(now.Unix()*1000), fmt.Sprint(now.Unix()*1000), zero, zero, zero, users[(j+i+channelNum)%len(users)].Id, channels[channelNum].Id, parentRoot, parentRoot, "", message, "", emptyobject, "", emptyarray, emptyarray, zero)
					csvLines <- results
				}
			}
		})

		wg.Wait()
	}
}

func importCSVToSQL(csvLines chan []string, db *sqlx.DB) {
	done := false
	for !done {
		tmpfile, err := ioutil.TempFile("", "loadtestload")
		if err != nil {
			mlog.Error("Can't create a temporary file for loading posts.", mlog.Err(err))
			return
		}
		csvWriter := csv.NewWriter(tmpfile)

		numLines := 0
		for {
			line, ok := <-csvLines
			if !ok {
				done = true
				break
			}
			if err := csvWriter.Write(line); err != nil {
				mlog.Error("Failed to write csv line.", mlog.Err(err))
			}
			numLines += 1
			if numLines >= 1000 {
				break
			}
		}

		csvWriter.Flush()
		tmpfile.Close()

		mysql.RegisterLocalFile(tmpfile.Name())
		_, err = db.Exec("LOAD DATA LOCAL INFILE '" + tmpfile.Name() + "' INTO TABLE Posts")
		if err != nil {
			mlog.Error("Couldn't load csv data", mlog.Err(err))
		}
		os.Remove(tmpfile.Name())
	}
}
