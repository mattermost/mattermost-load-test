package main

import (
	ltops "github.com/mattermost/mattermost-load-test-ops"
	"github.com/spf13/cobra"
)

var createCluster = &cobra.Command{
	Use:   "create-cluster",
	Short: "Creates a cluster to run Mattermost on for load testing",
	RunE:  createClusterCmd,
}

func createClusterCmd(cmd *cobra.Command, args []string) error {
	var config ltops.ClusterConfig
	config.Name, _ = cmd.Flags().GetString("name")
	config.AppInstanceType, _ = cmd.Flags().GetString("app-instance-type")
	config.AppInstanceCount, _ = cmd.Flags().GetInt("app-instance-count")
	config.DBInstanceType, _ = cmd.Flags().GetString("db-instance-type")
	config.DBInstanceCount, _ = cmd.Flags().GetInt("db-instance-count")
	config.LoadtestInstanceCount, _ = cmd.Flags().GetInt("loadtest-instance-count")

	clusterService, err := createTerraformClusterService()
	if err != nil {
		return err
	}

	_, err = clusterService.CreateCluster(&config)
	return err
}

func init() {
	createCluster.Flags().String("name", "", "a unique name for the cluster (required)")
	createCluster.MarkFlagRequired("name")

	createCluster.Flags().String("app-instance-type", "", "the app instance type (required)")
	createCluster.MarkFlagRequired("app-instance-type")

	createCluster.Flags().Int("app-instance-count", 1, "the number of app instances")

	createCluster.Flags().String("db-instance-type", "", "the db instance type (required)")
	createCluster.MarkFlagRequired("db-instance-type")

	createCluster.Flags().Int("db-instance-count", 1, "the number of db instances")

	createCluster.Flags().Int("loadtest-instance-count", 1, "the number of loadtest instances")

	rootCmd.AddCommand(createCluster)
}
