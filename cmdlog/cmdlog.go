// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package cmdlog

import (
	"os"

	"github.com/mattermost/platform/model"
	logging "github.com/op/go-logging"
)

var FORMAT_STRING_FILE logging.Formatter = logging.MustStringFormatter(`[%{level}] %{message}`)
var FORMAT_STRING_CONSOLE logging.Formatter = logging.MustStringFormatter(`[%{level}] %{message}`)

var log = logging.MustGetLogger("loadtest")
var logfile *os.File
var formattedFileBackend logging.Backend
var stBackend StringChannelBackend

type StringChannelBackend struct {
	Messages chan string
}

func (me *StringChannelBackend) Log(level logging.Level, calldepth int, rec *logging.Record) error {
	me.Messages <- rec.Formatted(calldepth)
	return nil
}

func init() {
	logfile, err := os.Create("loadtest.log")
	if err != nil {
		panic("Can't open log file")
	}

	formattedFileBackend = logging.NewBackendFormatter(logging.NewLogBackend(logfile, "", 0), FORMAT_STRING_FILE)

	log.SetBackend(logging.MultiLogger(formattedFileBackend))
}

func CloseLog() {
	log.SetBackend(logging.SetBackend())
	logfile.Sync()
	logfile.Close()
}

func GetStringChannelBackend() <-chan string {
	stBackend.Messages = make(chan string, 10)
	formattedStringChannelBackend := logging.NewBackendFormatter(&stBackend, FORMAT_STRING_CONSOLE)
	log.SetBackend(logging.MultiLogger(formattedFileBackend, formattedStringChannelBackend))
	return stBackend.Messages
}

func SetConsoleLog() {
	formattedLogBackend := logging.NewBackendFormatter(logging.NewLogBackend(os.Stdout, "", 0), FORMAT_STRING_CONSOLE)
	log.SetBackend(logging.MultiLogger(formattedFileBackend, formattedLogBackend))
}

func Debug(a ...interface{}) {
	log.Debug("", a...)
}

func Error(a ...interface{}) {
	log.Error("", a...)
}

func Info(a ...interface{}) {
	log.Info("", a...)
}

func Debugf(format string, a ...interface{}) {
	log.Debugf(format, a...)
}

func Errorf(format string, a ...interface{}) {
	log.Errorf(format, a...)
}

func Infof(format string, a ...interface{}) {
	log.Infof(format, a...)
}

func Println(a ...interface{}) {
	log.Info("", a...)
}

func AppError(err *model.AppError) {
	Errorf("eid: %v\n\tmsg: %v\n\tdtl: %v", err.Id, err.Message, err.DetailedError)
}
