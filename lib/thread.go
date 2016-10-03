package lib

import "fmt"

// ThreadStatus represents the major states of this thread
type ThreadStatus int

// ThreadInactive inactive state
const (
	ThreadInactive ThreadStatus = 0
	ThreadActive   ThreadStatus = 1
)

// Thread will be responsible for testplan runtime
type Thread struct {
	id       int
	status   ThreadStatus
	tpThread TestPlan
	stopchan chan bool
}

func (t *Thread) Init(tp TestPlanGen, activityPipe chan<- Activity) {
	t.tpThread = tp(t.id, activityPipe)
}

// Start kicks off test plan
func (t *Thread) Start(activityPipe chan<- Activity) {
	t.stopchan = make(chan bool)
	shouldStart := true
	for {
		select {
		case <- t.stopchan:
			return
		default:
			activityPipe <- t.started()
			shouldStart = t.tpThread.Start()
			if !shouldStart {
				activityPipe <- t.stopped()
				return
			}
		}
	}
}

func (t *Thread) Stop() {
	t.tpThread.Stop()
	if t.stopchan != nil {
		defer close(t.stopchan)
		t.stopchan <- true
	}
}

func (t Thread) started() Activity {
	t.status = ThreadActive
	msg := fmt.Sprintf("Thread %d has started", t.id)
	return Activity{Status: StatusLaunching, ID: t.id, Message: msg}
}

func (t Thread) stopped() Activity {
	t.status = ThreadInactive
	msg := fmt.Sprintf("Thread %d has stopped", t.id)
	return Activity{Status: StatusInactive, ID: t.id, Message: msg}
}
