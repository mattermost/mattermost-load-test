package lib

// Status is a Test Plan Activity Status
type Status int

// StatusActive active status for activity
const (
	StatusActive Status = iota
	StatusInactive
	StatusLaunching
	StatusError
	StatusAction
	StatusLaunchFailed
	StatusIncoming
)

// Activity structures represent messages between user and group
type Activity struct {
	Status  Status
	ID      int
	Err     error
	Message string
}
