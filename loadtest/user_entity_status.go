// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package loadtest

const (
	STATUS_LAUNCHING      int = iota
	STATUS_ACTIVE         int = iota
	STATUS_STOPPED        int = iota
	STATUS_ERROR          int = iota
	STATUS_FAILED_LAUNCH  int = iota
	STATUS_FAILED_ACTIVE  int = iota
	STATUS_ACTION_SEND    int = iota
	STATUS_ACTION_RECIEVE int = iota
)

type UserEntityStatusReport struct {
	Status  int
	Err     error
	Details string
	Config  *EntityConfig
}

/*
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
	case STATUS_FAILED_LAUNCH:
		return "FAILED_LAUNCH"
	case STATUS_FAILED_ACTIVE:
		return "FAILED_ACTIVE"
	case STATUS_ACTION_SEND:
		return "ACTION_SEND"
	case STATUS_ACTION_RECIEVE:
		return "ACTION_RECIEVE"
	}
	return "SOMTHING BAD"
}

func (report UserEntityStatusReport) String() string {
	if report.Err == nil {
		return fmt.Sprintf("#%v [%v]: %v", report.Config.EntityNumber, statusString(report.Status), report.Details)
	}
	return fmt.Sprintf("#%v [%v]: %v, %v", report.Config.EntityNumber, statusString(report.Status), report.Details, report.Err)
}

func processEntityStatusReport(out io.Writer, report UserEntityStatusReport, stats *UserEntityStatistics) {
	stats.updateEntityStatistics(report)
	out.Write([]byte(fmt.Sprintln("UserId: ", report.Config.EntityNumber, report)))
}

func OldUserEntityStatusPrinter(statusChan <-chan UserEntityStatusReport, clientTimingChannel <-chan TimedRoundTripperReport, stopChan <-chan bool, stopWait *sync.WaitGroup) {
	defer stopWait.Done()
	logfile, err := os.Create("status.log")
	if err != nil {
		cmdlog.Println("Unable to open log file for entity statuses")
		return
	}
	defer func() {
		logfile.Sync()
		logfile.Close()
	}()

	stats := NewUserEntityStatistics()

	go doPrintStats(stats, stopChan)

	// This strange thing makes sure that the statusChan is drained before it will listen to the stopChan
	for {
		select {
		case report := <-statusChan:
			processEntityStatusReport(logfile, report, stats)
		case timingReport := <-clientTimingChannel:
			stats.updateClientTimingStats(timingReport)
		default:
			select {
			case report := <-statusChan:
				processEntityStatusReport(logfile, report, stats)
			case timingReport := <-clientTimingChannel:
				stats.updateClientTimingStats(timingReport)
			case <-stopChan:
				return
			}
		}
	}
}*/
