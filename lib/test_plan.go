package lib

// TestPlanGen will create and initialize a test plan
type TestPlanGen func(int, chan<- Activity) TestPlan

// TestPlan is the General Interface for the test runner
type TestPlan interface {
	Generator(id int, activityChannel chan<- Activity) TestPlan
	Start() bool
	Stop()
	PanicCheck()
	GlobalSetup() (err error)
}
