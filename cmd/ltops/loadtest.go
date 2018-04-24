package main

import (
	"path/filepath"

	"github.com/mattermost/mattermost-load-test-ops/terraform"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var loadTest = &cobra.Command{
	Use:   "loadtest -- [args...]",
	Short: "Runs a mattermost-load-test command againt the given cluster",
	RunE: func(cmd *cobra.Command, args []string) error {
		clusterName, _ := cmd.Flags().GetString("cluster")
		//config, _ := cmd.Flags().GetString("config")

		workingDir, err := defaultWorkingDirectory()
		if err != nil {
			return err
		}

		cluster, err := terraform.LoadCluster(filepath.Join(workingDir, clusterName))
		if err != nil {
			return errors.Wrap(err, "Couldn't load cluster")
		}

		return cluster.Loadtest(nil)
	},
}

func init() {
	loadTest.Flags().StringP("cluster", "c", "", "cluster name (required)")
	loadTest.MarkFlagRequired("cluster")

	loadTest.Flags().StringP("config", "f", "", "a config file to use instead of the default (the ConnectionConfiguration section is mostly ignored)")

	rootCmd.AddCommand(loadTest)
}
