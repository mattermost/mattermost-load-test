package main

import (
	"path/filepath"

	"github.com/mattermost/mattermost-load-test/terraform"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var deleteCluster = &cobra.Command{
	Use:   "delete [name]",
	Args:  cobra.ExactArgs(1),
	Short: "Deletes a cluster previously created by create",
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		workingDir, err := defaultWorkingDirectory()
		if err != nil {
			return err
		}

		cluster, err := terraform.LoadCluster(filepath.Join(workingDir, name))
		if err != nil {
			return errors.Wrap(err, "Couldn't load cluster")
		}

		return cluster.Destroy()
	},
}

func init() {
	rootCmd.AddCommand(deleteCluster)
}
