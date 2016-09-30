package lib

// Status is a Test Plan Activity Status
type Status int

// StatusActive active status for activity
const (
	StatusActive       Status = 0
	StatusInactive     Status = 1
	StatusLaunching    Status = 2
	StatusError        Status = 3
	StatusAction       Status = 4
	StatusLaunchFailed Status = 5
)

// Activity structures represent messages between user and group
type Activity struct {
	Status  Status
	ID      int
	Err     error
	Message string
}
