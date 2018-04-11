package main

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var deleteCluster = &cobra.Command{
	Use:   "delete [name]",
	Args:  cobra.ExactArgs(1),
	Short: "Deletes a cluster previously created by create",
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		clusterService, err := createTerraformClusterService()
		if err != nil {
			return err
		}

		cluster, err := clusterService.LoadCluster(name)
		if err != nil {
			return errors.Wrap(err, "Couldn't load cluster")
		}

		return clusterService.DeleteCluster(cluster)
	},
}

func init() {
	rootCmd.AddCommand(deleteCluster)
}
