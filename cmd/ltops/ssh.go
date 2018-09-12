package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattermost/mattermost-load-test/kubernetes"
	"github.com/mattermost/mattermost-load-test/ltops"
	"github.com/mattermost/mattermost-load-test/sshtools"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var sshCommand = &cobra.Command{
	Use:              "ssh",
	Short:            "Connects to an instance via SSH",
	TraverseChildren: true,
}

var sshAppCommand = &cobra.Command{
	Use:     "app",
	Short:   "Connect to app instance via SSH",
	PreRunE: showHelpIfNoFlags,
	RunE: func(cmd *cobra.Command, args []string) error {
		clusterName, _ := cmd.Flags().GetString("cluster")
		instanceNumber, _ := cmd.Flags().GetInt("instance")

		workingDir, err := defaultWorkingDirectory()
		if err != nil {
			return err
		}

		cluster, err := LoadCluster(filepath.Join(workingDir, clusterName))
		if err != nil {
			return errors.Wrap(err, "couldn't load cluster")
		}

		addrs, err := cluster.GetAppInstancesAddrs()
		if err != nil {
			return errors.Wrap(err, "unable to get app instances")
		}

		if len(addrs) <= instanceNumber {
			return fmt.Errorf("invalid instance number: %d", instanceNumber)
		}

		return ssh(cluster, "app instance", addrs[instanceNumber], strings.Join(args, " "))
	},
}

var sshProxyCommand = &cobra.Command{
	Use:     "proxy",
	Short:   "Connect to proxy instance via SSH",
	PreRunE: showHelpIfNoFlags,
	RunE: func(cmd *cobra.Command, args []string) error {
		clusterName, _ := cmd.Flags().GetString("cluster")
		instanceNumber, _ := cmd.Flags().GetInt("instance")

		workingDir, err := defaultWorkingDirectory()
		if err != nil {
			return err
		}

		cluster, err := LoadCluster(filepath.Join(workingDir, clusterName))
		if err != nil {
			return errors.Wrap(err, "couldn't load cluster")
		}

		addrs, err := cluster.GetProxyInstancesAddrs()
		if err != nil {
			return errors.Wrap(err, "unable to get proxy instances")
		}

		if len(addrs) <= instanceNumber {
			return fmt.Errorf("invalid instance number: %d", instanceNumber)
		}

		return ssh(cluster, "proxy instance", addrs[instanceNumber], strings.Join(args, " "))
	},
}

var sshLoadtestCommand = &cobra.Command{
	Use:     "loadtest",
	Short:   "Connect to loadtest instance via SSH",
	PreRunE: showHelpIfNoFlags,
	RunE: func(cmd *cobra.Command, args []string) error {
		clusterName, _ := cmd.Flags().GetString("cluster")
		instanceNumber, _ := cmd.Flags().GetInt("instance")

		workingDir, err := defaultWorkingDirectory()
		if err != nil {
			return err
		}

		cluster, err := LoadCluster(filepath.Join(workingDir, clusterName))
		if err != nil {
			return errors.Wrap(err, "couldn't load cluster")
		}

		addrs, err := cluster.GetLoadtestInstancesAddrs()
		if err != nil {
			return errors.Wrap(err, "unable to get loadtest instances")
		}

		if len(addrs) <= instanceNumber {
			return fmt.Errorf("invalid instance number: %d", instanceNumber)
		}

		return ssh(cluster, "loadtest instance", addrs[instanceNumber], strings.Join(args, " "))
	},
}

var sshMetricsCommand = &cobra.Command{
	Use:     "metrics",
	Short:   "Connect to metrics instance",
	PreRunE: showHelpIfNoFlags,
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

		var addr string
		if cluster.Type() == kubernetes.CLUSTER_TYPE {
			addr, err = cluster.(*kubernetes.Cluster).GetMetricsPodName()
		} else {
			addr, err = cluster.GetMetricsAddr()
		}

		if err != nil {
			return errors.Wrap(err, "could not get metrics server address")
		}

		return ssh(cluster, "metrics instance", addr, strings.Join(args, " "))
	},
}

func ssh(cluster ltops.Cluster, description, addr, cmd string) error {
	logrus.Infof("Connecting to %s at %s", description, addr)

	if cluster.Type() == kubernetes.CLUSTER_TYPE {
		return sshtools.SSHInteractiveKubesPod(addr)
	}

	if cmd == "" {
		return sshtools.SSHInteractiveTerminal(cluster.SSHKey(), addr)
	} else {
		logrus.Debugf("Invoking: %s", cmd)

		client, err := sshtools.SSHClient(cluster.SSHKey(), addr)
		if err != nil {
			return err
		}
		defer client.Close()

		return sshtools.RemoteCommand(client, cmd, os.Stdout)
	}
}

func init() {
	sshCommand.PersistentFlags().StringP("cluster", "c", "", "the name of the cluster (required)")
	sshCommand.MarkPersistentFlagRequired("cluster")

	sshCommand.PersistentFlags().IntP("instance", "i", 0, "the instance number (default 0)")

	sshCommand.AddCommand(sshAppCommand, sshLoadtestCommand, sshProxyCommand, sshMetricsCommand)

	sshCommand.Flags().SortFlags = false

	rootCmd.AddCommand(sshCommand)
}
