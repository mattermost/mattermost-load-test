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

	decoder := json.NewDecoder(file)
	foundStructuredLogs := false
	foundResults := false
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

			foundResults = true
			if err := dumpTimingsText(timings); err != nil {
				return errors.Wrap(err, "failed to dump timings")
			}
		}
	}

	if !foundStructuredLogs {
		return errors.Wrap(err, "failed to find structured logs")
	}
	if !foundResults {
		return errors.Wrap(err, "failed to find results")
	}

	return nil
}
