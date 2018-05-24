package ltparse

import (
	"encoding/json"
	"os"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-load-test/loadtest"
)

type ResultsConfig struct {
	File      string
	Display   string
	Aggregate bool
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

	allTimings := []*loadtest.ClientTimingStats{}

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
			timings := &loadtest.ClientTimingStats{}
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

	var timings *loadtest.ClientTimingStats
	if !config.Aggregate {
		timings = allTimings[len(allTimings)-1]
	} else {
		for _, t := range allTimings {
			timings = timings.Merge(t)
		}
	}

	switch config.Display {
	case "markdown":
		if err := dumpTimingsMarkdown(timings); err != nil {
			return errors.Wrap(err, "failed to dump timings")
		}
	case "text":
		fallthrough
	default:
		if err := dumpTimingsText(timings); err != nil {
			return errors.Wrap(err, "failed to dump timings")
		}
	}

	return nil
}
