// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package main

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/mattermost/mattermost-load-test/loadtestconfig"
)

const (
	STATUS_LAUNCHING      int = iota
	STATUS_ACTIVE         int = iota
	STATUS_STOPPED        int = iota
	STATUS_ERROR          int = iota
	STATUS_FAILED         int = iota
	STATUS_ACTION_SEND    int = iota
	STATUS_ACTION_RECIEVE int = iota
)

type UserEntityStatusReport struct {
	Status  int
	Err     error
	Details string
	Config  *UserEntityConfig
}

func statusString(status int) string {
	switch status {
	case STATUS_LAUNCHING:
		return "LAUNCHING"
	case STATUS_ACTIVE:
		return "ACTIVE"
	case STATUS_STOPPED:
		return "STOPPED"
	case STATUS_ERROR:
		return "ERROR"
	case STATUS_FAILED:
		return "FAILED"
	case STATUS_ACTION_SEND:
		return "ACTION_SEND"
	case STATUS_ACTION_RECIEVE:
		return "ACTION_RECIEVE"
	}
	return "SOMTHING BAD"
}

func (report UserEntityStatusReport) String() string {
	if report.Err == nil {
		return fmt.Sprintf("#%v [%v]: %v", report.Config.Id, statusString(report.Status), report.Details)
	}
	return fmt.Sprintf("#%v [%v]: %v, %v", report.Config.Id, statusString(report.Status), report.Details, report.Err)
}

func processEntityStatusReport(out io.Writer, report UserEntityStatusReport, stats *UserEntityStatistics) {
	stats.updateEntityStatistics(report)
	out.Write([]byte(fmt.Sprintln("UserId: ", report.Config.EntityUser.Id, report)))
}

func UserEntityStatusPrinter(out UserEntityLogger, statusChan <-chan UserEntityStatusReport, stopChan <-chan bool, stopWait *sync.WaitGroup, users []loadtestconfig.ServerStateUser) {
	defer stopWait.Done()
	logfile, err := os.Create("status.log")
	if err != nil {
		out.Println("Unable to open log file for entity statuses")
		return
	}
	defer func() {
		logfile.Sync()
		logfile.Close()
	}()

	stats := NewUserEntityStatistics(1 * time.Second)

	go doPrintStats(out, stats, stopChan)

	// This strange thing makes sure that the statusChan is drained before it will listen to the stopChan
	for {
		select {
		case report := <-statusChan:
			processEntityStatusReport(logfile, report, stats)
		default:
			select {
			case report := <-statusChan:
				processEntityStatusReport(logfile, report, stats)
			case <-stopChan:
				return
			}
		}
	}
}
