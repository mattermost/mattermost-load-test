package cmd

import (
	"github.com/spf13/cobra"

	"github.com/mattermost/mattermost-load-test-ops/ops"
)

var deleteCluster = &cobra.Command{
	Use:   "delete-cluster",
	Short: "Deletes a cluster previously created by create-cluster",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		return ops.DeleteCluster(name)
	},
}

func init() {
	deleteCluster.Flags().String("name", "", "the name of the cluster to delete (required)")
	deleteCluster.MarkFlagRequired("name")

	rootCmd.AddCommand(deleteCluster)
}
