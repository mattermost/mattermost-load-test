package cmd

import (
	"github.com/spf13/cobra"

	"github.com/mattermost/mattermost-load-test-ops/ops"
)

var deploy = &cobra.Command{
	Use:   "deploy [distribution-path]",
	Short: "Deploys an app distribution to a load test cluster",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		clusterName, _ := cmd.Flags().GetString("cluster-name")
		return ops.Deploy(args[0], clusterName)
	},
}

func init() {
	deploy.Flags().String("cluster-name", "", "the name of the cluster to deploy to (required)")
	deploy.MarkFlagRequired("cluster-name")

	rootCmd.AddCommand(deploy)
}
