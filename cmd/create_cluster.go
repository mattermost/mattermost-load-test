package cmd

import (
	"github.com/spf13/cobra"

	"github.com/mattermost/mattermost-loadtest-ops/ops"
)

var createCluster = &cobra.Command{
	Use:   "create-cluster",
	Short: "Creates a cluster to run Mattermost on for loadtesting",
	RunE: func(cmd *cobra.Command, args []string) error {
		var config ops.ClusterConfiguration
		config.Name, _ = cmd.Flags().GetString("name")
		config.AppInstanceType, _ = cmd.Flags().GetString("app-instance-type")
		config.AppInstanceCount, _ = cmd.Flags().GetInt("app-instance-count")
		config.DBInstanceType, _ = cmd.Flags().GetString("db-instance-type")
		return ops.CreateCluster(&config)
	},
}

func init() {
	createCluster.Flags().String("name", "", "a unique name for the cluster (required)")
	createCluster.MarkFlagRequired("name")

	createCluster.Flags().String("app-instance-type", "", "the app instance type (required)")
	createCluster.MarkFlagRequired("app-instance-type")

	createCluster.Flags().Int("app-instance-count", 1, "the number of app instances")

	createCluster.Flags().String("db-instance-type", "", "the db instance type (required)")
	createCluster.MarkFlagRequired("db-instance-type")

	rootCmd.AddCommand(createCluster)
}
