// Copyright (c) 2017 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information

package loadtest

import "bytes"

type ServerCLICommandRunner interface {
	RunCommand(string) (bool, string)
	RunPlatformCommand(string) (bool, string)
	SendLoadtestFile(buf *bytes.Buffer) error
	Close() error
}
