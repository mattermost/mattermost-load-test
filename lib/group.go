package lib

import (
	"fmt"
	"time"
)

// Group is a simple container for the threads and stats aggregation
type Group struct {
	Total        int
	LaunchCount  int
	ActiveCount  int
	ActionCount  int
	Errors       []string
	threads      []Thread
	ActivityPipe chan Activity
	chanStop     chan bool
}

func (g *Group) initialize() {
	g.Errors = []string{}
	g.ActivityPipe = make(chan Activity, 40000)
	g.chanStop = make(chan bool)
	g.threads = []Thread{}
}

// Kickstart will kick off group and connect channel listeners
func (g *Group) Kickstart(tpGen TestPlanGen, total, offset, SecRamp int) {

	g.initialize()
	g.Total = total

	// Generate a Test Plan for Global Setup
	testPlan := tpGen(0, nil)
	err := testPlan.GlobalSetup()

	if err != nil {
		panic(err)
	}

	sleepIncrement := time.Duration(SecRamp) * time.Second / time.Duration(total)
	go g.spinUpThreads(tpGen, total, offset, sleepIncrement)

	for activity := range g.ActivityPipe {
		switch activity.Status {
		case StatusActive:
			g.registerThreadActive(activity)
		case StatusInactive:
			g.registerThreadInactive(activity)
		case StatusLaunching:
			g.registerThreadLaunching(activity)
		case StatusLaunchFailed:
			g.registerLaunchFail(activity)
		case StatusError:
			g.registerThreadError(activity)
		case StatusAction:
			g.registerThreadAction(activity)
		default:
			panic("Unhandled Activity type in group")
		}
	}
}

func (g *Group) spinUpThreads(tp TestPlanGen, total, start int, sleep time.Duration) {
	for i := start; i < total+start; i++ {
		select {
		case <-g.chanStop:
			return
		default:
			t := Thread{id: i}
			t.Init(tp, g.ActivityPipe)
			g.threads = append(g.threads, t)
			go t.Start(g.ActivityPipe)
			time.Sleep(sleep)
		}
	}
}

func (g *Group) Stop() {
	if g.Total > len(g.threads) {
		defer close(g.chanStop)
		g.chanStop <- true
	}

	for _, ok := range g.threads {
		ok.Stop()
	}
}

func (g *Group) registerThreadLaunching(activity Activity) {
	g.LaunchCount++
}

func (g *Group) registerLaunchFail(activity Activity) {
	g.LaunchCount--
}

func (g *Group) registerThreadFinished(activity Activity) {
	g.LaunchCount--
}

func (g *Group) registerThreadAction(activity Activity) {
	g.ActionCount++
}

func (g *Group) registerThreadActive(activity Activity) {
	g.LaunchCount--
	g.ActiveCount++
}

func (g *Group) registerThreadInactive(activity Activity) {
	g.ActiveCount--
}

func (g *Group) registerThreadError(activity Activity) {
	errMsg := fmt.Sprintf("Thread #%d - %v - %v", activity.ID, activity.Message, activity.Err.Error())
	g.Errors = append(g.Errors, errMsg)
}
