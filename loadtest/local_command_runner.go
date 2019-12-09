// Copyright (c) 2017 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information

package loadtest

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/mattermost/mattermost-server/v5/mlog"
)

type MattermostLocalConnection struct {
	mattermostInstallDir string
}

func NewLocalConnection(mattermostInstallDir string) (*MattermostLocalConnection, error) {
	return &MattermostLocalConnection{
		mattermostInstallDir: mattermostInstallDir,
	}, nil
}

func (c *MattermostLocalConnection) RunCommand(command string) (bool, string) {
	mlog.Info("Running local command: " + command)
	split := strings.Fields(command)
	cmd := exec.Command(split[0], split[1:]...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, err.Error() + " : " + string(output)
	}

	return true, string(output)
}

func (c *MattermostLocalConnection) RunPlatformCommand(args string) (bool, string) {
	wd, err := os.Getwd()
	if err != nil {
		mlog.Warn("Unable to get working directory", mlog.Err(err))
	}
	os.Chdir(c.mattermostInstallDir)
	success, result := c.RunCommand("./bin/mattermost " + args)
	os.Chdir(wd)
	return success, result
}

func (c *MattermostLocalConnection) SendLoadtestFile(buf *bytes.Buffer) error {
	return ioutil.WriteFile(path.Join(c.mattermostInstallDir, "loadtestusers.json"), buf.Bytes(), 0666)
}

func (c *MattermostLocalConnection) Close() error {
	return nil
}
