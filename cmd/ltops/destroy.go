package main

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var destroyCluster = &cobra.Command{
	Use:   "destroy",
	Short: "Destroys a previously created cluster",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().NFlag() == 0 {
			cmd.Help()
			os.Exit(0)
		}

		return nil
	},
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

		return cluster.Destroy()
	},
}

func init() {
	destroyCluster.Flags().StringP("cluster", "c", "", "the name of the cluster (required)")
	destroyCluster.MarkFlagRequired("cluster")

	destroyCluster.Flags().SortFlags = false

	rootCmd.AddCommand(destroyCluster)
}
