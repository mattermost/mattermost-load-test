package ltparse

import (
	"encoding/json"
	"io"
	"os"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-load-test/loadtest"
)

type ResultsConfig struct {
	File         string
	BaselineFile string
	Display      string
	Aggregate    bool
}

func parseTimingsFromFile(input io.Reader) ([]*loadtest.ClientTimingStats, error) {
	allTimings := []*loadtest.ClientTimingStats{}
	decoder := json.NewDecoder(input)
	foundStructuredLogs := false
	for decoder.More() {
		log := map[string]interface{}{}
		if err := decoder.Decode(&log); err != nil {
			return nil, errors.Wrap(err, "failed to decode")
		}
		foundStructuredLogs = true

		// Look for result logs
		if log["tag"] == "timings" {
			timings := &loadtest.ClientTimingStats{}
			if err := mapstructure.Decode(log["timings"], timings); err != nil {
				continue
			}

			allTimings = append(allTimings, timings)
		}
	}

	if !foundStructuredLogs {
		return nil, errors.New("failed to find structured logs")
	}
	if len(allTimings) == 0 {
		return nil, errors.New("failed to find results")
	}

	return allTimings, nil
}

func ParseResults(config *ResultsConfig, input io.Reader) error {
	allTimings, err := parseTimingsFromFile(input)
	if err != nil {
		return err
	}

	allBaselineTimings := []*loadtest.ClientTimingStats{}
	if config.BaselineFile != "" {
		baselineFile, err := os.Open(config.BaselineFile)
		if err != nil {
			return errors.Wrap(err, "failed to open structured log file")
		}
		defer baselineFile.Close()

		allBaselineTimings, err = parseTimingsFromFile(baselineFile)
		if err != nil {
			return err
		}
	}

	var timings *loadtest.ClientTimingStats
	if !config.Aggregate {
		timings = allTimings[len(allTimings)-1]
	} else {
		for _, t := range allTimings {
			timings = timings.Merge(t)
		}
	}

	var baselineTimings *loadtest.ClientTimingStats
	if len(allBaselineTimings) > 0 {
		if !config.Aggregate {
			baselineTimings = allBaselineTimings[len(allBaselineTimings)-1]
		} else {
			for _, t := range allBaselineTimings {
				baselineTimings = timings.Merge(t)
			}
		}
	}

	switch config.Display {
	case "markdown":
		if err := dumpTimingsMarkdown(timings, baselineTimings); err != nil {
			return errors.Wrap(err, "failed to dump timings")
		}
	case "text":
		if len(allBaselineTimings) > 0 {
			return errors.Wrap(err, "cannot compare to baseline using text display")
		}
		fallthrough
	default:
		if err := dumpTimingsText(timings); err != nil {
			return errors.Wrap(err, "failed to dump timings")
		}
	}

	return nil
}
