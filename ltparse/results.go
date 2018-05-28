package ltparse

import (
	"encoding/json"
	"io"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-load-test/loadtest"
)

type ResultsConfig struct {
	Input         io.Reader
	BaselineInput io.Reader
	Output        io.Writer
	Display       string
	Aggregate     bool
}

func parseTimings(input io.Reader) ([]*loadtest.ClientTimingStats, error) {
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

func ParseResults(config *ResultsConfig) error {
	allTimings, err := parseTimings(config.Input)
	if err != nil {
		return err
	}

	allBaselineTimings := []*loadtest.ClientTimingStats{}
	if config.BaselineInput != nil {
		allBaselineTimings, err = parseTimings(config.BaselineInput)
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
		if err := dumpTimingsMarkdown(timings, baselineTimings, config.Output); err != nil {
			return errors.Wrap(err, "failed to dump timings")
		}
	case "text":
		if len(allBaselineTimings) > 0 {
			return errors.Wrap(err, "cannot compare to baseline using text display")
		}
		fallthrough
	default:
		if err := dumpTimingsText(timings, config.Output); err != nil {
			return errors.Wrap(err, "failed to dump timings")
		}
	}

	return nil
}
