// Copyright (c) 2017 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information

package loadtest

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/mattermost/mattermost-server/mlog"
	"golang.org/x/crypto/ssh"
)

type MattermostSSHConnection struct {
	Client               *ssh.Client
	mattermostInstallDir string
	configFileLoc        string
}

func ConnectSSH(sshHostnamePort, sshKey, sshUsername, sshPassword, mattermostInstallDir string, configFileLoc string) (*MattermostSSHConnection, error) {
	var config *ssh.ClientConfig
	if sshKey != "" {
		key := []byte(sshKey)

		if !strings.Contains(sshKey, "\n") {
			var err error
			key, err = ioutil.ReadFile(sshKey)
			if err != nil {
				return nil, fmt.Errorf("Unable to read SSH key provided: %v", err.Error())
			}
		}

		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("Unable to parse SSH key provided: %v", err.Error())
		}

		config = &ssh.ClientConfig{
			User: sshUsername,
			Auth: []ssh.AuthMethod{
				ssh.PublicKeys(signer),
			},
		}
	} else {
		config = &ssh.ClientConfig{
			User: sshUsername,
			Auth: []ssh.AuthMethod{
				ssh.Password(sshPassword),
			},
		}
	}
	config.HostKeyCallback = ssh.InsecureIgnoreHostKey()
	config.Timeout = time.Second * 10

	client, err := ssh.Dial("tcp", sshHostnamePort, config)
	if err != nil {
		return nil, fmt.Errorf("Unable to connect to server " + err.Error())
	}

	return &MattermostSSHConnection{
		Client:               client,
		mattermostInstallDir: mattermostInstallDir,
		configFileLoc:        configFileLoc,
	}, nil
}

func (c *MattermostSSHConnection) RunCommand(command string) (bool, string) {
	mlog.Info("Running remote command: " + command)
	session, err := c.Client.NewSession()
	if err != nil {
		return false, "Failed to open session: " + err.Error()
	}
	defer session.Close()

	var b bytes.Buffer
	session.Stdout = &b
	session.Stderr = session.Stdout
	if err := session.Run(command); err != nil {
		return false, "Unable to run command in session. Error: " + err.Error() + ". Output: " + b.String()
	}
	session.Close()

	return true, b.String()
}

func (c *MattermostSSHConnection) RunPlatformCommand(args string) (bool, string) {
	if c.configFileLoc != "" {
		return c.RunCommand("cd " + c.mattermostInstallDir + " && ./bin/platform " + args + " --config " + c.configFileLoc)
	} else {
		return c.RunCommand("cd " + c.mattermostInstallDir + " && ./bin/platform " + args)
	}
}

func (c *MattermostSSHConnection) SendLoadtestFile(buf *bytes.Buffer) error {
	return sendBuffer(buf, 0666, "loadtestusers.json", c.mattermostInstallDir, c.Client)
}

func (c *MattermostSSHConnection) Close() error {
	return c.Client.Close()
}

// Inspired by https://github.com/tmc/scp
func sendBuffer(buf *bytes.Buffer, mode os.FileMode, fileName string, destination string, client *ssh.Client) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	go func() {
		w, _ := session.StdinPipe()
		defer w.Close()
		fmt.Fprintf(w, "C%#o %d %s\n", mode, buf.Len(), fileName)
		io.Copy(w, buf)
		fmt.Fprint(w, "\x00")
	}()
	cmd := fmt.Sprintf("scp -t %s", destination)
	if err := session.Run(cmd); err != nil {
		return err
	}
	return nil
}
