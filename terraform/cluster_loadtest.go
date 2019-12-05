package terraform

import (
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mattermost/mattermost-load-test/ltops"

	"github.com/mattermost/mattermost-load-test/sshtools"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (c *Cluster) loadtestInstance(logger logrus.FieldLogger, addr string, instanceNum int, configFile []byte) error {
	debugLogWriter := newLogrusWriter(logger, logrus.DebugLevel)
	defer debugLogWriter.Close()

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

	if len(configFile) > 0 {
		if err := sshtools.UploadBytes(client, configFile, "mattermost-load-test/loadtestconfig.json", debugLogWriter); err != nil {
			return errors.Wrap(err, "failed to upload config file")
		}
	}

	if err := configureLoadtestInstance(instanceNum, client, c, logger); err != nil {
		return errors.Wrap(err, "failed to configure loadtest instance")
	}

	commandOutputFile := filepath.Join(c.Env.WorkingDirectory, "results", "loadtest-out-"+addr+".txt")
	logger.Infof("Logging to %s", commandOutputFile)

	if err := os.MkdirAll(filepath.Dir(commandOutputFile), 0700); err != nil {
		return errors.Wrap(err, "Unable to create results directory.")
	}
	outfile, err := os.Create(commandOutputFile)
	if err != nil {
		return errors.Wrap(err, "Unable to create loadtest results file.")
	}
	defer outfile.Close()

	// Unlike os.Exec, there's a data race if session.Stdout == session.Stderr.
	// https://github.com/golang/go/issues/5582. Avoid this by using a pipe, which is safe
	// when written to concurrently.
	sessionPipeReader, sessionPipeWriter := io.Pipe()
	session.Stdout = sessionPipeWriter
	session.Stderr = sessionPipeWriter
	defer sessionPipeWriter.Close()

	infoLogWriter := newLogrusWriter(logger, logrus.InfoLevel)
	defer infoLogWriter.Close()
	go io.Copy(io.MultiWriter(outfile, infoLogWriter), sessionPipeReader)

	logger.Info("Running loadtest")
	if err := session.Run("cd mattermost-load-test && ./bin/loadtest all"); err != nil {
		return err
	}

	return nil
}

func (c *Cluster) Loadtest(options *ltops.LoadTestOptions) error {
	loadtestInstancesAddrs, err := c.GetLoadtestInstancesAddrs()
	if err != nil || len(loadtestInstancesAddrs) <= 0 {
		return errors.Wrap(err, "Unable to get loadtest instance addresses")
	}

	var configFile []byte
	if len(options.ConfigFile) > 0 {
		data, err := ltops.GetFileOrURL(options.ConfigFile)
		if err != nil {
			return errors.Wrap(err, "failed to load config file")
		}

		configFile = data
	}

	var wg sync.WaitGroup
	for instanceNum, addr := range loadtestInstancesAddrs {
		if instanceNum > 0 {
			// Stagger the instance starts to avoid races.
			time.Sleep(time.Second * 10)
		}

		addr := addr
		wg.Add(1)
		go func() {
			defer wg.Done()
			logger := logrus.StandardLogger().WithField("instance", addr)
			if err = c.loadtestInstance(logger, addr, instanceNum, configFile); err != nil {
				logrus.Error(err)
			}
		}()
	}

	wg.Wait()

	return nil
}
