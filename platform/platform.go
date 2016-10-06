package platform

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"strings"
	"time"

	model "github.com/mattermost/platform/model"
)

// Platform is the interface with Mattermost. It holds relevant connection
// state and provides helper functions to work with the api.
type Platform struct {
	url             string
	client          *model.Client
	mmUser          *model.User
	team            *model.Team
	initialLoad     *model.InitialLoad
	webSocketClient *model.WebSocketClient
}

// GeneratePlatform will create the default platform
func GeneratePlatform(url string) Platform {
	return Platform{url: url, client: model.NewClient(url)}
}

// SetAuthToken will set stored token to skip login
func (p *Platform) SetAuthToken(token string) {
	p.client.SetOAuthToken(token)
}

// GetAuthToken returns the current auth token
func (p *Platform) GetAuthToken() string {
	return p.client.AuthToken
}

// GetClientID will return the mmUser client ID if set
func (p *Platform) GetClientID() (id string) {
	if p.mmUser == nil {
		return ""
	}
	return p.mmUser.Id
}

// PingServer pings mattermost error, returns server properties
func (p *Platform) PingServer() (map[string]string, error) {
	res, err := p.client.GetPing()
	if err != nil {
		return nil, err
	}
	return res, nil
}

// NewSocketClient creates a WebSocketClient from platform
func (p *Platform) NewSocketClient(url string) (*model.WebSocketClient, error) {
	webSocketClient, err := model.NewWebSocketClient(url, p.client.AuthToken)
	if err != nil {
		return nil, err
	}
	return webSocketClient, nil
}

// Login will attempt to login to mattermost and return error otherwise
func (p *Platform) Login(email, password string) error {
	res, err := p.client.Login(email, password)
	if err != nil {
		return err
	}
	p.mmUser = res.Data.(*model.User)
	return nil
}

// Login will attempt to login to mattermost and return error otherwise
func (p *Platform) LoginById(id, password string) error {
	res, err := p.client.LoginById(id, password)
	if err != nil {
		return err
	}
	p.mmUser = res.Data.(*model.User)
	return nil
}

// UpdateProfile with values if unset
func (p *Platform) UpdateProfile(first, last, username string) error {
	if first != "" {
		p.mmUser.FirstName = first
	}
	if last != "" {
		p.mmUser.LastName = last
	}
	if username != "" {
		p.mmUser.Username = username
	}

	res, err := p.client.UpdateUser(p.mmUser)
	if err != nil {
		return err
	}
	p.mmUser = res.Data.(*model.User)
	return nil
}

// InitialLoad saves mm essential initial data
func (p *Platform) InitialLoad() error {
	res, err := p.client.GetInitialLoad()
	if err != nil {
		return err
	}
	p.initialLoad = res.Data.(*model.InitialLoad)
	return nil
}

//GetMe returns current user
func (p *Platform) GetMe() *model.AppError {
	_, err := p.client.GetMe("")
	return err
}

// FindTeam will look for teamname in initialLoad data
func (p *Platform) FindTeam(teamname string, setClient bool) (*model.Team, error) {
	for _, team := range p.initialLoad.Teams {
		if team.Name == teamname {
			p.team = team
			if setClient {
				p.client.SetTeamId(team.Id)
			}
			return team, nil
		}
	}
	return nil, errors.New("User not part of team")
}

// SetTeam in the platform client
func (p *Platform) SetTeam(teamname string) {
	p.client.SetTeamId(teamname)
}

// GetChannels recieves list of channels
func (p *Platform) GetChannels() (*model.ChannelList, error) {
	res, err := p.client.GetChannels("")
	if err != nil {
		return nil, err
	}
	return res.Data.(*model.ChannelList), nil
}

// GetChannel find the channel from all channels the user has joined
func (p *Platform) GetChannel(channelName string) (*model.Channel, error) {
	res, err := p.GetChannels()
	if err != nil {
		return nil, err
	}
	FQCN := strings.Replace(channelName, " ", "-", -1)
	for _, channel := range res.Channels {
		if channel.Name == FQCN {
			return channel, nil
		}
	}
	return nil, errors.New("Channel not found")
}

// SGetChannel find the channel by name
func (p *Platform) SGetChannel(channelName string) (*model.Channel, error) {
	FQCD := strings.Replace(channelName, " ", "-", -1)
	res, err := p.client.GetChannel(FQCD, "")
	if err != nil {
		return nil, err
	}
	return res.Data.(*model.Channel), nil
}

// JoinChannel find the channel
func (p *Platform) JoinChannel(channelName string) error {
	FQCD := strings.Replace(channelName, " ", "-", -1)
	_, err := p.client.JoinChannelByName(FQCD)
	if err != nil {
		return err
	}
	return nil
}

// CreateChannel makes a new channel
func (p *Platform) CreateChannel(channelName string, private bool) (*model.Channel, error) {
	FQCD := strings.Replace(channelName, " ", "-", -1)
	channel := model.Channel{}
	channel.Name = FQCD
	channel.DisplayName = channelName
	channel.Purpose = "This was Channel created by a Dummy Test User"
	if private {
		channel.Type = model.CHANNEL_PRIVATE
	} else {
		channel.Type = model.CHANNEL_OPEN
	}

	res, err := p.client.CreateChannel(&channel)
	if err != nil {
		return nil, err
	}
	return res.Data.(*model.Channel), nil
}

// SendMessage to Platform
func (p *Platform) SendMessage(channel *model.Channel, msg, replyToID string) error {
	post := model.Post{}

	post.ChannelId = channel.Id
	post.Message = msg
	if replyToID != "" {
		post.RootId = replyToID
	}

	_, err := p.client.CreatePost(&post)
	return err
}

// SendAttachment to Message with attachment
func (p *Platform) SendAttachment(channel *model.Channel, msg string, filenames []string, replyToID string) error {
	post := model.Post{}

	post.ChannelId = channel.Id
	post.Message = msg
	if replyToID != "" {
		post.RootId = replyToID
	}
	if len(filenames) > 0 {
		post.Filenames = filenames
	}

	_, err := p.client.CreatePost(&post)
	return err
}

// StubSend mocked out server
func (p *Platform) StubSend(channel *model.Channel, msg, replyToID string) error {
	time.Sleep(1)
	return nil
}

// UploadRandomImage test image and returns file response
func (p *Platform) UploadRandomImage(channel *model.Channel, rm RandomMessage) (res *model.FileUploadResponse, err error) {
	name, data, ok := rm.Media()
	if !ok {
		return nil, errors.New("Could not load random image for MultipartImage")
	}
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("files", name)
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, data)
	if err != nil {
		return nil, err
	}
	field, err := writer.CreateFormField("channel_id")
	if err != nil {
		return nil, err
	}
	_, err = field.Write([]byte(channel.Id))
	if err != nil {
		return nil, err
	}
	err = writer.Close()
	if err != nil {
		return nil, err
	}
	upload, err := p.client.UploadPostAttachment(body.Bytes(), writer.FormDataContentType())
	return upload.Data.(*model.FileUploadResponse), nil
}
