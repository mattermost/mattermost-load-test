package main

import (
	"path/filepath"

	"github.com/mattermost/mattermost-server/v5/mlog"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var destroyCluster = &cobra.Command{
	Use:     "destroy",
	Short:   "Destroys a previously created cluster",
	PreRunE: showHelpIfNoFlags,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("cluster")

		workingDir, err := defaultWorkingDirectory()
		if err != nil {
			return err
		}

		cluster, err := LoadCluster(filepath.Join(workingDir, name))
		if err != nil {
			return errors.Wrap(err, "Couldn't load cluster")
		}
		mlog.Info("Destroying cluster...")
		if err = cluster.Destroy(); err != nil {
			return err
		}
		mlog.Info("Cluster destroyed successfully")
		return nil
	},
}

func init() {
	destroyCluster.Flags().StringP("cluster", "c", "", "the name of the cluster (required)")
	destroyCluster.MarkFlagRequired("cluster")

	destroyCluster.Flags().SortFlags = false
	destroyCluster.Aliases = append(destroyCluster.Aliases, "delete")

	rootCmd.AddCommand(destroyCluster)
}
