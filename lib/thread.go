package lib

import "fmt"

// ThreadStatus represents the major states of this thread
type ThreadStatus int

// ThreadInactive inactive state
const (
	ThreadInactive ThreadStatus = 0
	ThreadActive   ThreadStatus = 1
)

// Thread will will be responsible for testplan runtime
type Thread struct {
	id     int
	status ThreadStatus
}

// Start kicks off test plan
func (t *Thread) Start(tp TestPlanGen, activityPipe chan<- Activity) {
	shouldStart := true
	for {
		tpThread := tp(t.id, activityPipe)
		activityPipe <- t.started()
		runResult := tpThread.Start()
		shouldStart = tpThread.Stop(runResult)
		if !shouldStart {
			break
		}
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
