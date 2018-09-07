package main

import (
	"os"
	"path/filepath"

	"github.com/mattermost/mattermost-load-test/kubernetes"
	"github.com/mattermost/mattermost-load-test/ltops"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var loadTest = &cobra.Command{
	Use:   "loadtest",
	Short: "Runs a mattermost-load-test command against the given cluster",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().NFlag() == 0 {
			cmd.Help()
			os.Exit(0)
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		clusterName, _ := cmd.Flags().GetString("cluster")

		loadtestOptions := &ltops.LoadTestOptions{}
		loadtestOptions.ConfigFile, _ = cmd.Flags().GetString("config")
		loadtestOptions.ForceBulkLoad, _ = cmd.Flags().GetBool("force-bulk-load")
		loadtestOptions.SkipBulkLoad, _ = cmd.Flags().GetBool("skip-bulk-load")
		loadtestOptions.Workers, _ = cmd.Flags().GetInt("workers")

		if loadtestOptions.ForceBulkLoad && loadtestOptions.SkipBulkLoad {
			return errors.New("cannot have both force-bulk-load and skip-bulk-load set")
		}

		workingDir, err := defaultWorkingDirectory()
		if err != nil {
			return err
		}

		cluster, err := LoadCluster(filepath.Join(workingDir, clusterName))
		if err != nil {
			return errors.Wrap(err, "Couldn't load cluster")
		}

		if len(loadtestOptions.ConfigFile) > 0 && cluster.Type() == kubernetes.CLUSTER_TYPE {
			return errors.New("cannot override config file for Kubernetes")
		}

		return cluster.Loadtest(loadtestOptions)
	},
}

func init() {
	loadTest.Flags().StringP("cluster", "c", "", "cluster name (required)")
	loadTest.MarkFlagRequired("cluster")

	loadTest.Flags().StringP("config", "f", "", "a loadtest config file")
	loadTest.Flags().BoolP("force-bulk-load", "", false, "force bulk load even if bulk loading already complete")
	loadTest.Flags().BoolP("skip-bulk-load", "", false, "skip bulk load if bulk loading already complete or you loaded using other way")
	loadTest.Flags().IntP("workers", "", 32, "how many workers to execute the bulk import in parallel.")

	loadTest.Flags().SortFlags = false

	rootCmd.AddCommand(loadTest)
}
