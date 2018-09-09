package terraform

import (
	"bufio"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mattermost/mattermost-load-test/ltops"

	"github.com/mattermost/mattermost-load-test/sshtools"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type logrusWriter struct {
	*io.PipeWriter

	reader *io.PipeReader
	done   chan bool
}

func newLogrusWriter(logger logrus.FieldLogger) *logrusWriter {
	reader, writer := io.Pipe()
	done := make(chan bool)

	go func() {
		// Log lines one at a time and mapped into logrus calls for visual clarity. The
		// raw output written to the file will be unaffected.
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			var jsonText interface{}
			if err := json.Unmarshal(scanner.Bytes(), &jsonText); err != nil {
				logger.Info(scanner.Text())
			} else if jsonTextMap, ok := jsonText.(map[string]interface{}); !ok {
				logger.Info(scanner.Text())
			} else {
				level, _ := jsonTextMap["level"].(string)
				msg, _ := jsonTextMap["msg"].(string)

				// Clean up unused fields
				delete(jsonTextMap, "msg")
				delete(jsonTextMap, "level")
				delete(jsonTextMap, "ts")
				delete(jsonTextMap, "caller")

				// Special case timings output. This will still be available
				// in the results for use by ltparse.
				if msg == "Timings" {
					jsonTextMap["timings"] = "(omitted)"
				}

				loggerWithFields := logger.WithFields(logrus.Fields(jsonTextMap))

				switch level {
				case "debug":
					loggerWithFields.Debug(msg)
				case "info":
					loggerWithFields.Info(msg)
				case "warn":
					loggerWithFields.Warn(msg)
				case "error":
					fallthrough
				default:
					loggerWithFields.Error(msg)
				}
			}
		}
		if err := scanner.Err(); err != nil {
			logrus.Errorf("failed to scan and log: %s", err.Error())

			// Drain the reader, otherwise the ssh session won't end.
			// TODO: Really, we still want to just dump a really long line, but this is a test.
			io.Copy(ioutil.Discard, reader)
		}
		close(done)
	}()

	return &logrusWriter{
		PipeWriter: writer,
		reader:     reader,
		done:       done,
	}
}

func (w *logrusWriter) Close() error {
	w.PipeWriter.Close()
	w.reader.Close()
	<-w.done

	return nil
}

func (c *Cluster) loadtestInstance(logger logrus.FieldLogger, addr string, instanceNum int, configFile []byte) error {
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
		if err := sshtools.UploadBytes(client, configFile, "mattermost-load-test/loadtestconfig.json"); err != nil {
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

	logrusWriter := newLogrusWriter(logger)
	defer logrusWriter.Close()
	go io.Copy(io.MultiWriter(outfile, logrusWriter), sessionPipeReader)

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
			logger := logrus.StandardLogger().WithField("instance", addr)
			if err = c.loadtestInstance(logger, addr, instanceNum, configFile); err != nil {
				logrus.Error(err)
			}
			wg.Done()
		}()
	}

	wg.Wait()

	return nil
}
