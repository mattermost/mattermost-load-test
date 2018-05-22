package ltparse

import (
	"encoding/json"
	"os"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

type ResultsConfig struct {
	File string
}

func ParseResults(config *ResultsConfig) error {
	var file *os.File
	var err error
	if config.File == "" {
		file = os.Stdin
	} else {
		file, err = os.Open(config.File)
		if err != nil {
			return errors.Wrap(err, "failed to open structured log file")
		}
	}

	allTimings := []*Timings{}

	decoder := json.NewDecoder(file)
	foundStructuredLogs := false
	for decoder.More() {
		log := map[string]interface{}{}
		if err := decoder.Decode(&log); err != nil {
			return errors.Wrap(err, "failed to decode")
		}
		foundStructuredLogs = true

		// Look for result logs
		if log["tag"] == "timings" {
			timings := &Timings{}
			err = mapstructure.Decode(log["timings"], timings)
			if err != nil {
				continue
			}

			allTimings = append(allTimings, timings)
		}
	}

	if !foundStructuredLogs {
		return errors.Wrap(err, "failed to find structured logs")
	}

	if len(allTimings) == 0 {
		return errors.Wrap(err, "failed to find results")
	}

	// Default to showing the last timings.
	if err := dumpTimingsText(allTimings[len(allTimings)-1]); err != nil {
		return errors.Wrap(err, "failed to dump timings")
	}

	return nil
}
