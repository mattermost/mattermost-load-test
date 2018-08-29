package main

import (
	"path/filepath"

	"github.com/mattermost/mattermost-load-test/ltops"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var loadTest = &cobra.Command{
	Use:   "loadtest -- [args...]",
	Short: "Runs a mattermost-load-test command against the given cluster",
	RunE: func(cmd *cobra.Command, args []string) error {
		loadtestOptions := &ltops.LoadTestOptions{}
		clusterName, _ := cmd.Flags().GetString("cluster")
		loadtestOptions.ForceBulkLoad, _ = cmd.Flags().GetBool("force-bulk-load")
		loadtestOptions.SkipBulkLoad, _ = cmd.Flags().GetBool("skip-bulk-load")
		loadtestOptions.Workers, _ = cmd.Flags().GetInt("workers")

		if loadtestOptions.ForceBulkLoad && loadtestOptions.SkipBulkLoad {
			return errors.New("cannot have both force-bulk-load and skip-bulk-load set")
		}
		//config, _ := cmd.Flags().GetString("config")

		workingDir, err := defaultWorkingDirectory()
		if err != nil {
			return err
		}

		cluster, err := LoadCluster(filepath.Join(workingDir, clusterName))
		if err != nil {
			return errors.Wrap(err, "Couldn't load cluster")
		}

		return cluster.Loadtest(loadtestOptions)
	},
}

func init() {
	loadTest.Flags().StringP("cluster", "c", "", "cluster name (required)")
	loadTest.MarkFlagRequired("cluster")

	loadTest.Flags().BoolP("force-bulk-load", "", false, "force bulk load even if bulk loading already complete")
	loadTest.Flags().BoolP("skip-bulk-load", "", false, "skip bulk load if bulk loading already complete or you loaded using other way")
	loadTest.Flags().IntP("workers", "", 32, "how many workers to execute the bulk import in parallel.")

	// TODO: Implement
	//loadTest.Flags().StringP("config", "f", "", "a config file to use instead of the default (the ConnectionConfiguration section is mostly ignored)")

	rootCmd.AddCommand(loadTest)
}
