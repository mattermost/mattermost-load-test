package main

// Listen Test plan is basically the simple test, plus a socket connection
// to listen for messages. The test will reply on a psuedo-random method.
// there are 10 users per room. Plus users take a random sleep break
// after logging. Messages are sent after a random message break.

import (
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"time"

	"github.com/mattermost/platform/model"

	l "github.com/mattermost/mattermost-load-test/lib"
	p "github.com/mattermost/mattermost-load-test/platform"
)

// UserListenTestPlan - try out the test plan interface
type UserListenTestPlan struct {
	id              int
	activityChannel chan<- l.Activity
	mm              p.Platform
}

// Generator sets up & exports the channels
func (tp UserListenTestPlan) Generator(id int, activityChannel chan<- l.Activity) l.TestPlan {
	newPlan := new(UserListenTestPlan)
	newPlan.id = id
	newPlan.activityChannel = activityChannel
	newPlan.mm = p.GeneratePlatform(Config.PlatformURL)
	return newPlan
}

// Start is a long running function that should only quit on error
func (tp *UserListenTestPlan) Start() (shouldRestart bool) {

	defer tp.PanicCheck()

	userEmail := GeneratePlatformEmail(tp.id)
	userPassword := GeneratePlatformPass(tp.id)
	// Login User
	session := GetSession(userEmail)
	if session != "" {
		tp.mm.SetAuthToken(session)
	} else {
		err := tp.mm.Login(userEmail, userPassword)

		if err != nil {
			tp.registerLaunchFail()
			tp.handleError(err, "Login Failed", false)
			return false
		} else {
			SaveSession(userEmail, tp.mm.GetAuthToken())
		}
	}

	rand.Seed(int64(tp.id))
	randomInt := rand.Intn(Config.LoginBreak)
	sleepDuration := time.Duration(randomInt) * time.Second
	time.Sleep(sleepDuration)
	tp.registerActive()

	// Initial Load
	err := tp.mm.InitialLoad()
	if err != nil {
		return tp.handleError(err, "Initial Load Failed", true)
	}

	// Team Lookup Load
	_, err = tp.mm.FindTeam(Config.TeamName, true)
	if err != nil {
		if errr := tp.mm.GetMe(); errr != nil && errr.StatusCode == 401 {
			DeleteSession(userEmail)
			return true
		}
		return tp.handleError(err, "Team Lookup Failed", false)
	}

	channelExtension := tp.id / 10
	userChannel := fmt.Sprintf("%v%v", Config.TestChannel, channelExtension)

	channel, err := tp.mm.GetChannel(userChannel)
	if err != nil {
		return tp.handleError(err, "Get Channel Failed", true)
	}

	webSocketClient, err := tp.mm.NewSocketClient(Config.SocketURL)
	if err != nil {
		return tp.handleError(err, "Open Socket Failed", true)
	}

	webSocketClient.Listen()

	go func() {
		for {
			select {
			case event := <-webSocketClient.EventChannel:
				if event.Event != model.WEBSOCKET_EVENT_POSTED {
					continue
				}

				post := model.PostFromJson(strings.NewReader(event.Data["post"].(string)))
				if post != nil {
					if RandomChoice(Config.ReplyPercent) {
						message := p.RandomMessage{}.Plain()
						message = fmt.Sprintf("reply: %v", message)
						err = tp.mm.SendMessage(channel, message, "")
						if err != nil && !reflect.ValueOf(err).IsNil() {
							tp.handleError(err, "Message Send Failed", false)
							continue
						}
						tp.threadSendMessage()
					}
				}
			}
		}
	}()

	go func() {
		for {
			time.Sleep(time.Second * time.Duration(rand.Intn(Config.MessageBreak)))
			message := p.RandomMessage{}.Plain()
			err = tp.mm.SendMessage(channel, message, "")
			if err != nil && !reflect.ValueOf(err).IsNil() {
				tp.handleError(err, "Message Send Failed", false)
				continue
			}
			tp.threadSendMessage()
		}
	}()

	select {}
}

// Stop takes the result of start(), and can change return
// respond true if the thread should restart, false otherwise
func (tp *UserListenTestPlan) Stop(runResult bool) (shouldRestart bool) {
	return runResult
}

// GlobalSetup will run before the test plan. It will spin up a basic test plan
// from the Generator and will not be reused.
func (tp *UserListenTestPlan) GlobalSetup() (err error) {
	return nil
}

// PanicCheck will check for panics, used as a defer in test plan
func (tp *UserListenTestPlan) PanicCheck() {
	if r := recover(); r != nil {
		// if Error != nil {
		// 	Error.Printf("ERROR ON WORKER: %v", r)
		// } else {
		// 	fmt.Printf("ERROR ON WORKER: %v", r)
		// }
		switch x := r.(type) {
		case string:
			tp.handleError(errors.New(x), "Error caught unexpected (thread failed)", true)
		case error:
			tp.handleError(x, "Error caught unexpected (thread failed)", true)
		default:
			tp.handleError(errors.New("Unknown Panic"), "Error caught unexpected (thread failed)", true)
		}
	}
}

func (tp *UserListenTestPlan) registerActive() {
	tp.activityChannel <- l.Activity{
		Status:  l.StatusActive,
		ID:      tp.id,
		Message: "Thread active",
	}
}

func (tp *UserListenTestPlan) registerInactive() {
	tp.activityChannel <- l.Activity{
		Status:  l.StatusInactive,
		ID:      tp.id,
		Message: "Thread inactive",
	}
}

func (tp *UserListenTestPlan) registerLaunchFail() {
	tp.activityChannel <- l.Activity{
		Status:  l.StatusLaunchFailed,
		ID:      tp.id,
		Message: "Failed launch",
	}
}

func (tp *UserListenTestPlan) handleError(err error, message string, notify bool) bool {
	tp.activityChannel <- l.Activity{
		Status:  l.StatusError,
		ID:      tp.id,
		Message: message,
		Err:     err,
	}
	if notify {
		tp.registerInactive()
	}
	time.Sleep(time.Second * 5)
	return true
}

func (tp *UserListenTestPlan) threadSendMessage() {
	tp.activityChannel <- l.Activity{
		Status:  l.StatusAction,
		ID:      tp.id,
		Message: fmt.Sprintf("User %v sent a message", tp.id),
	}
}
