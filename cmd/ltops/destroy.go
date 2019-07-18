package main

import (
	"path/filepath"

	"github.com/mattermost/mattermost-server/mlog"

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
		if err = cluster.Destroy(); err == nil {
			mlog.Info("Custer destroyed successfully")
		}
		return err
	},
}

func init() {
	destroyCluster.Flags().StringP("cluster", "c", "", "the name of the cluster (required)")
	destroyCluster.MarkFlagRequired("cluster")

	destroyCluster.Flags().SortFlags = false
	destroyCluster.Aliases = append(destroyCluster.Aliases, "delete")

	rootCmd.AddCommand(destroyCluster)
}
