package lib

import (
	"fmt"
	"log"
	"time"
)

// GroupManager creates and controls groups
type GroupManager struct {
	TestID string
	Group  Group
	DB     *Database
	Info   *log.Logger
	Error  *log.Logger
}

// InitDB will create the database connection and store it in group manager
func (gm *GroupManager) InitDB(url, username, password string) {
	db, err := CreateDBConnection(url, username, password)
	if err != nil {
		panic(err)
	}
	gm.DB = &db
}

// Start kicks off each group
func (gm *GroupManager) Start(tp TestPlanGen, count, offset, SecRamp int) {
	if tp == nil {
		panic("Failed to receive test plan generator")
	}

	gm.panicWithoutDB()
	gm.Group = Group{}
	go gm.startGroupCheck()
	go gm.Group.Kickstart(tp, count, offset, SecRamp)

	select {}
}

func (gm *GroupManager) startGroupCheck() {
	defer gm.PanicCheck()

	for {
		checkin := Checkin{
			Time:        time.Now(),
			ThreadCount: gm.Group.Total,
			LaunchCount: gm.Group.LaunchCount,
			ActiveCount: gm.Group.ActiveCount,
			ActionCount: gm.Group.ActionCount,
			Errors:      gm.Group.Errors,
			TestID:      gm.TestID,
		}
		if gm.Info != nil {
			gm.logGroupCheck(checkin)
		}
		if gm.Error != nil {
			gm.logErrors(gm.Group.Errors)
		}
		if gm.DB != nil {
			gm.DB.writeCheckin(checkin)
		}
		gm.Group.Errors = []string{}
		gm.Group.ActionCount = 0
		time.Sleep(time.Second * 5)
	}
}

// PanicCheck will be defer called in case of panic
func (gm *GroupManager) PanicCheck() {
	if r := recover(); r != nil {
		if gm.Error != nil {
			gm.Error.Printf("ERROR ON GMANAGER: %v", r)
		} else {
			fmt.Printf("ERROR ON GMANAGER: %v", r)
		}
	}
}

func (gm *GroupManager) panicWithoutDB() {
	if &gm.DB == nil {
		panic("Failed to find Database during start, did you call InitDB?")
	}
}

func (gm *GroupManager) logGroupCheck(c Checkin) {
	gm.Info.Printf(`STATUS
				Total: %v
				Launching: %v
				Active: %v
				Actions: %v
				Errors: %d`,
		gm.Group.Total, gm.Group.LaunchCount, gm.Group.ActiveCount, gm.Group.ActionCount, len(gm.Group.Errors))
}

func (gm *GroupManager) logErrors(errors []string) {
	for _, s := range gm.Group.Errors {
		gm.Error.Printf("Thread Error: %v", s)
	}
}
