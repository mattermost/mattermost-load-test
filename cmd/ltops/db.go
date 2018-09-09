package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/mattermost/mattermost-load-test/ltops"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var dbCommand = &cobra.Command{
	Use:   "db",
	Short: "Launches mysql connected to the cluster database",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().NFlag() == 0 {
			cmd.Help()
			os.Exit(0)
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		clusterName, _ := cmd.Flags().GetString("cluster")

		workingDir, err := defaultWorkingDirectory()
		if err != nil {
			return err
		}

		cluster, err := LoadCluster(filepath.Join(workingDir, clusterName))
		if err != nil {
			return errors.Wrap(err, "couldn't load cluster")
		}

		settings, err := cluster.DBSettings()
		if err != nil {
			return errors.Wrap(err, "failed to get database settings")
		}

		// TODO: support psql
		return mysql(settings)
	},
}

func mysql(settings *ltops.DBSettings) error {
	logrus.Infof("Connecting to %s:%d", settings.Endpoint, settings.Port)

	cmd := exec.Command("mysql", "-u", settings.Username, fmt.Sprintf("-p%s", settings.Password), "-h", settings.Endpoint, "-P", strconv.Itoa(settings.Port), settings.Database)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func init() {
	dbCommand.Flags().StringP("cluster", "c", "", "the name of the cluster (required)")
	dbCommand.MarkFlagRequired("cluster")

	dbCommand.Flags().SortFlags = false

	rootCmd.AddCommand(dbCommand)
}
