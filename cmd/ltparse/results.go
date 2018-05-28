package main

import (
	"fmt"
	"io"
	"os"

	"github.com/mattermost/mattermost-load-test/ltparse"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var results = &cobra.Command{
	Use:   "results",
	Short: "Parses structured logs from a loadtest and generates results",
	RunE:  resultsCmd,
}

func resultsCmd(cmd *cobra.Command, args []string) error {
	var config ltparse.ResultsConfig
	config.File, _ = cmd.Flags().GetString("file")
	config.BaselineFile, _ = cmd.Flags().GetString("baseline")
	config.Display, _ = cmd.Flags().GetString("display")
	config.Aggregate, _ = cmd.Flags().GetBool("aggregate")

	switch config.Display {
	case "text":
	case "markdown":
	default:
		return fmt.Errorf("unexpected --display flag: %s", config.Display)
	}

	var input io.Reader
	if config.File == "" {
		input = os.Stdin
	} else {
		file, err := os.Open(config.File)
		if err != nil {
			return errors.Wrap(err, "failed to open structured log file")
		}
		defer file.Close()
		input = file
	}

	if err := ltparse.ParseResults(&config, input); err != nil {
		return err
	}

	return nil
}

func init() {
	results.Flags().StringP("file", "f", "", "a file containing structured logs from a loadtest")
	results.Flags().StringP("display", "d", "text", "one of 'text' or 'markdown'")
	results.Flags().BoolP("aggregate", "a", false, "aggregate all results found instead of just picking the last")
	results.Flags().StringP("baseline", "b", "", "a file containing structured logs to which to compare results")

	rootCmd.AddCommand(results)
}
