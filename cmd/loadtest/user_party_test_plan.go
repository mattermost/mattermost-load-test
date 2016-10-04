package main

// This test makes a few excessive calls, not current in users, please use user_simple

// This test is to simulate a all-hands channel. Everybody in the same room.
// It will likely need a rewrite in the same style as user simple test.

import (
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"time"

	l "github.com/mattermost/mattermost-load-test/lib"
	p "github.com/mattermost/mattermost-load-test/platform"
)

// UserPartyTestPlan - try out the test plan interface
type UserPartyTestPlan struct {
	id              int
	activityChannel chan<- l.Activity
	mm              p.Platform
}

// Generator sets up & exports the channels
func (tp UserPartyTestPlan) Generator(id int, activityChannel chan<- l.Activity) l.TestPlan {
	newPlan := new(UserPartyTestPlan)
	newPlan.id = id
	newPlan.activityChannel = activityChannel
	newPlan.mm = p.GeneratePlatform(Config.PlatformURL)
	return newPlan
}

// Start is a long running function that should only quit on error
func (tp *UserPartyTestPlan) Start() (shouldRestart bool) {

	defer tp.PanicCheck()

	userEmail := GeneratePlatformEmail(tp.id)
	userPassword := GeneratePlatformPass(tp.id)
	userName := GeneratePlatformUsername(tp.id)
	userFirst := GeneratePlatformFirst(tp.id)
	userLast := GeneratePlatformLast()

	// Ping Server
	_, err := tp.mm.PingServer()
	if err != nil {
		tp.registerLaunchFail()
		return tp.handleError(err, "Ping Failed", false)
	}

	// Login User
	err = tp.mm.Login(userEmail, userPassword)
	if err != nil {
		tp.registerLaunchFail()
		tp.handleError(err, "Login Failed", false)
		return false
	}

	tp.registerActive()

	// Update Good
	err = tp.mm.UpdateProfile(userFirst, userLast, userName)
	if err != nil {
		return tp.handleError(err, "Profile Update Failed", true)
	}

	// Initial Load
	err = tp.mm.InitialLoad()
	if err != nil {
		return tp.handleError(err, "Initial Load Failed", true)
	}

	// Team Lookup Load
	_, err = tp.mm.FindTeam(Config.TeamName, true)
	if err != nil {
		return tp.handleError(err, "Team Lookup Failed", true)
	}

	userChannel := Config.TestChannel

	//  Create/Get Test Channel
	channel, err := tp.mm.CreateChannel(userChannel, false)
	if err != nil {

		err = tp.mm.JoinChannel(userChannel)
		if err != nil {
			return tp.handleError(err, "Join Channel Failed", true)
		}

		channel, err = tp.mm.GetChannel(userChannel)
		if err != nil {
			return tp.handleError(err, "Create/Get Channel Failed", true)
		}
	}

	// Send Message
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

}

// Stop takes the result of start(), and can change return
// respond true if the thread should restart, false otherwise
func (tp *UserPartyTestPlan) Stop(runResult bool) (shouldRestart bool) {
	return runResult
}

// GlobalSetup will run before the test plan. It will spin up a basic test plan
// from the Generator and will not be reused
func (tp *UserPartyTestPlan) GlobalSetup() (err error) {
	return nil
}

// PanicCheck will check for panics, used as a defer in test plan
func (tp *UserPartyTestPlan) PanicCheck() {
	if r := recover(); r != nil {
		if Error != nil {
			Error.Printf("ERROR ON WORKER: %v", r)
		} else {
			fmt.Printf("ERROR ON WORKER: %v", r)
		}
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

func (tp *UserPartyTestPlan) registerActive() {
	tp.activityChannel <- l.Activity{
		Status:  l.StatusActive,
		ID:      tp.id,
		Message: "Thread active",
	}
}

func (tp *UserPartyTestPlan) registerInactive() {
	tp.activityChannel <- l.Activity{
		Status:  l.StatusInactive,
		ID:      tp.id,
		Message: "Thread inactive",
	}
}

func (tp *UserPartyTestPlan) registerLaunchFail() {
	tp.activityChannel <- l.Activity{
		Status:  l.StatusLaunchFailed,
		ID:      tp.id,
		Message: "Failed launch",
	}
}

func (tp *UserPartyTestPlan) handleError(err error, message string, notify bool) bool {
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

func (tp *UserPartyTestPlan) threadSendMessage() {
	tp.activityChannel <- l.Activity{
		Status:  l.StatusAction,
		ID:      tp.id,
		Message: fmt.Sprintf("User %v sent a message", tp.id),
	}
}
