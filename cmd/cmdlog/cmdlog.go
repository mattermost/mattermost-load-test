// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package cmdlog

import (
	"fmt"
	"os"

	"github.com/mattermost/platform/model"
)

func Debug(a ...interface{}) (int, error) {
	fmt.Print("[DEBUG] ")
	return fmt.Fprintln(os.Stdout, a...)
}

func Error(a ...interface{}) (int, error) {
	fmt.Print("[ERROR] ")
	return fmt.Fprintln(os.Stdout, a...)
}

func Info(a ...interface{}) (int, error) {
	fmt.Print("[INFO] ")
	return fmt.Fprintln(os.Stdout, a...)
}

func Debugf(format string, a ...interface{}) (int, error) {
	return Debug(fmt.Sprintf(format, a...))
}

func Errorf(format string, a ...interface{}) (int, error) {
	return Error(fmt.Sprintf(format, a...))
}

func Infof(format string, a ...interface{}) (int, error) {
	return Info(fmt.Sprintf(format, a...))
}

func AppError(err *model.AppError) {
	Error("eid: %v\n\tmsg: %v\n\tdtl: %v", err.Id, err.Message, err.DetailedError)
}
