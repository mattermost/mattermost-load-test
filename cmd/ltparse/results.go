package main

import (
	"github.com/mattermost/mattermost-load-test/ltparse"
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

	if err := ltparse.ParseResults(&config); err != nil {
		return err
	}

	return nil
}

func init() {
	results.Flags().StringP("file", "f", "", "a file containing structured logs from a loadtest")

	rootCmd.AddCommand(results)
}
