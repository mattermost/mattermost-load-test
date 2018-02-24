package cmd

import (
	"github.com/spf13/cobra"

	"github.com/mattermost/mattermost-load-test-ops/ops"
)

var loadtest = &cobra.Command{
	Use:   "loadtest [args...]",
	Short: "Runs a mattermost-load-test command againt the given cluster",
	RunE: func(cmd *cobra.Command, args []string) error {
		clusterName, _ := cmd.Flags().GetString("cluster-name")
		return ops.Loadtest(clusterName, args)
	},
}

func init() {
	loadtest.Flags().String("cluster-name", "", "the name of the cluster to loadtest to (required)")
	loadtest.MarkFlagRequired("cluster-name")

	rootCmd.AddCommand(loadtest)
}
