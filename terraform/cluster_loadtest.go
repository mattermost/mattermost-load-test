package terraform

import (
	"os"
	"path/filepath"

	"github.com/mattermost/mattermost-load-test-ops/sshtools"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (c *Cluster) loadtestInstance(addr string) error {
	client, err := sshtools.SSHClient(c.SSHKey(), addr)
	if err != nil {
		return errors.Wrap(err, "unable to connect to loadtest instance via ssh")
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return errors.Wrap(err, "unable to create ssh session")
	}
	defer session.Close()

	commandOutputFile := filepath.Join(c.Env.WorkingDirectory, "results", "loadtest-out-"+addr+".txt")
	if err := os.MkdirAll(filepath.Dir(commandOutputFile), 0700); err != nil {
		return errors.Wrap(err, "Unable to create results directory.")
	}
	outfile, err := os.Create(commandOutputFile)
	if err != nil {
		return errors.Wrap(err, "Unable to create loadtest results file.")
	}

	session.Stdout = outfile
	session.Stderr = outfile

	logrus.Info("Running loadtest on " + addr)
	if err := session.Run("cd mattermost-load-test && ./bin/loadtest all"); err != nil {
		return err
	}

	return nil
}

func (c *Cluster) Loadtest() error {
	loadtestInstancesAddrs, err := c.GetLoadtestInstancesAddrs()
	if err != nil || len(loadtestInstancesAddrs) <= 0 {
		return errors.Wrap(err, "Unable to get loadtest instance addresses")
	}

	for _, addr := range loadtestInstancesAddrs {
		err := c.loadtestInstance(addr)
		if err != nil {
			logrus.Error(err)
		}
	}

	return nil
}
