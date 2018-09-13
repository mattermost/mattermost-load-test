package terraform

import (
	"bufio"
	"encoding/json"
	"io"
	"io/ioutil"

	"github.com/sirupsen/logrus"
)

type logrusWriter struct {
	*io.PipeWriter

	reader *io.PipeReader
	done   chan bool
}

func newLogrusWriter(logger logrus.FieldLogger, defaultLevel logrus.Level) *logrusWriter {
	reader, writer := io.Pipe()
	done := make(chan bool)

	log := func(logger logrus.FieldLogger, level, message string) {
		switch level {
		case "debug":
			logger.Debug(message)
		case "info":
			logger.Info(message)
		case "warn":
			logger.Warn(message)
		case "error":
			fallthrough
		default:
			logger.Error(message)
		}
	}

	go func() {
		// Log lines one at a time and mapped into logrus calls for visual clarity. The
		// raw output written to the file will be unaffected.
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			var jsonText interface{}
			if err := json.Unmarshal(scanner.Bytes(), &jsonText); err != nil {
				log(logger, defaultLevel.String(), scanner.Text())
			} else if jsonTextMap, ok := jsonText.(map[string]interface{}); !ok {
				log(logger, defaultLevel.String(), scanner.Text())
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

				log(logger.WithFields(logrus.Fields(jsonTextMap)), level, msg)
			}
		}
		if err := scanner.Err(); err != nil {
			logrus.Errorf("failed to scan and log: %s", err.Error())

			// Drain the reader, otherwise the ssh session may not end.
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
	<-w.done

	return nil
}
