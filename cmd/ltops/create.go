package main

import (
	"path/filepath"

	ltops "github.com/mattermost/mattermost-load-test-ops"
	"github.com/mattermost/mattermost-load-test-ops/terraform"
	"github.com/spf13/cobra"
)

var createCluster = &cobra.Command{
	Use:   "create",
	Short: "Creates a cluster to run Mattermost on for load testing",
	RunE:  createClusterCmd,
}

func createClusterCmd(cmd *cobra.Command, args []string) error {
	var config ltops.ClusterConfig
	config.Name, _ = cmd.Flags().GetString("name")
	config.AppInstanceType, _ = cmd.Flags().GetString("app-type")
	config.AppInstanceCount, _ = cmd.Flags().GetInt("app-count")
	config.DBInstanceType, _ = cmd.Flags().GetString("db-type")
	config.DBInstanceCount, _ = cmd.Flags().GetInt("db-count")
	config.LoadtestInstanceCount, _ = cmd.Flags().GetInt("loadtest-count")

	workingDir, err := defaultWorkingDirectory()
	if err != nil {
		return err
	}
	config.WorkingDirectory = filepath.Join(workingDir, config.Name)

	_, err = terraform.CreateCluster(&config)
	if err != nil {
		return err
	}

	return nil
}

func init() {
	createCluster.Flags().StringP("name", "", "c", "a unique name for the cluster (required)")
	createCluster.MarkFlagRequired("name")

	createCluster.Flags().String("app-type", "", "the app instance type (required)")
	createCluster.MarkFlagRequired("app-type")

	createCluster.Flags().Int("app-count", 1, "the number of app instances")

	createCluster.Flags().String("db-type", "", "the db instance type (required)")
	createCluster.MarkFlagRequired("db-type")

	createCluster.Flags().Int("db-count", 1, "the number of db instances")

	createCluster.Flags().Int("loadtest-count", 1, "the number of loadtest instances")

	rootCmd.AddCommand(createCluster)
}
