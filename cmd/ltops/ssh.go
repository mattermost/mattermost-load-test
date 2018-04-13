package main

import (
	"path/filepath"
	"strconv"

	"github.com/mattermost/mattermost-load-test-ops/sshtools"
	"github.com/mattermost/mattermost-load-test-ops/terraform"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var sshCommand = &cobra.Command{
	Use:   "ssh [cluster] [instance-id]",
	Short: "Connects to an instance via SSH",
}

var sshAppCommand = &cobra.Command{
	Use:   "app [cluster] [instance number]",
	Short: "Connect to app instance via SSH",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		clusterName := args[0]
		instanceNumber, err := strconv.Atoi(args[1])
		if err != nil {
			return errors.Wrap(err, "Instance number must be a number")
		}

		workingDir, err := defaultWorkingDirectory()
		if err != nil {
			return err
		}

		cluster, err := terraform.LoadCluster(filepath.Join(workingDir, clusterName))
		if err != nil {
			return errors.Wrap(err, "Couldn't load cluster")
		}

		addrs, err := cluster.GetAppInstancesAddrs()
		if err != nil {
			return errors.Wrap(err, "Unable to get app instances.")
		}

		if len(addrs) <= instanceNumber {
			return errors.New("Invalid instance number.")
		}

		addr := addrs[instanceNumber]
		logrus.Info("Connecting to " + addr)

		return sshtools.SSHInteractiveTerminal(cluster.SSHKey(), addr)
	},
}

var sshProxyCommand = &cobra.Command{
	Use:   "proxy [cluster] [instance number]",
	Short: "Connect to proxy instance via SSH",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		clusterName := args[0]
		instanceNumber, err := strconv.Atoi(args[1])
		if err != nil {
			return errors.Wrap(err, "Instance number must be a number")
		}

		workingDir, err := defaultWorkingDirectory()
		if err != nil {
			return err
		}

		cluster, err := terraform.LoadCluster(filepath.Join(workingDir, clusterName))
		if err != nil {
			return errors.Wrap(err, "Couldn't load cluster")
		}

		addrs, err := cluster.GetProxyInstancesAddrs()
		if err != nil {
			return errors.Wrap(err, "Unable to get proxy instances.")
		}

		if len(addrs) <= instanceNumber {
			return errors.New("Invalid instance number.")
		}

		addr := addrs[instanceNumber]
		logrus.Info("Connecting to " + addr)

		return sshtools.SSHInteractiveTerminal(cluster.SSHKey(), addr)
	},
}

var sshLoadtestCommand = &cobra.Command{
	Use:   "loadtest [cluster] [instance number]",
	Short: "Connect to loadtest instance via SSH",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		clusterName := args[0]
		instanceNumber, err := strconv.Atoi(args[1])
		if err != nil {
			return errors.Wrap(err, "Instance number must be a number")
		}

		workingDir, err := defaultWorkingDirectory()
		if err != nil {
			return err
		}

		cluster, err := terraform.LoadCluster(filepath.Join(workingDir, clusterName))
		if err != nil {
			return errors.Wrap(err, "Couldn't load cluster")
		}

		addrs, err := cluster.GetLoadtestInstancesAddrs()
		if err != nil {
			return errors.Wrap(err, "Unable to get loadtest instances.")
		}

		if len(addrs) <= instanceNumber {
			return errors.New("Invalid instance number.")
		}

		addr := addrs[instanceNumber]
		logrus.Info("Connecting to " + addr)

		return sshtools.SSHInteractiveTerminal(cluster.SSHKey(), addr)
	},
}

func init() {
	sshCommand.AddCommand(sshAppCommand, sshLoadtestCommand, sshProxyCommand)

	rootCmd.AddCommand(sshCommand)
}
