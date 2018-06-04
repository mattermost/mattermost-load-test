package main

import (
	"path/filepath"
	"strconv"

	"github.com/mattermost/mattermost-load-test/kubernetes"
	"github.com/mattermost/mattermost-load-test/ltops"
	"github.com/mattermost/mattermost-load-test/sshtools"
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
			return errors.Wrap(err, "instance number must be a number")
		}

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
			return errors.Wrap(err, "unable to get app instances.")
		}

		if len(addrs) <= instanceNumber {
			return errors.New("invalid instance number.")
		}

		return ssh(cluster, addrs[instanceNumber])
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

		cluster, err := LoadCluster(filepath.Join(workingDir, clusterName))
		if err != nil {
			return errors.Wrap(err, "couldn't load cluster")
		}

		addrs, err := cluster.GetProxyInstancesAddrs()
		if err != nil {
			return errors.Wrap(err, "unable to get proxy instances.")
		}

		if len(addrs) <= instanceNumber {
			return errors.New("invalid instance number.")
		}

		return ssh(cluster, addrs[instanceNumber])
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
			return errors.Wrap(err, "instance number must be a number")
		}

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
			return errors.Wrap(err, "unable to get loadtest instances.")
		}

		if len(addrs) <= instanceNumber {
			return errors.New("invalid instance number.")
		}

		return ssh(cluster, addrs[instanceNumber])
	},
}

var sshMetricsCommand = &cobra.Command{
	Use:   "metrics [cluster]",
	Short: "Connect to metrics instance",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		clusterName := args[0]

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
			return errors.Wrap(err, "could not get metrics server address.")
		}

		return ssh(cluster, addr)
	},
}

func ssh(cluster ltops.Cluster, addr string) error {
	logrus.Info("Connecting to " + addr)

	if cluster.Type() == kubernetes.CLUSTER_TYPE {
		return sshtools.SSHInteractiveKubesPod(addr)
	}

	return sshtools.SSHInteractiveTerminal(cluster.SSHKey(), addr)
}

func init() {
	sshCommand.AddCommand(sshAppCommand, sshLoadtestCommand, sshProxyCommand, sshMetricsCommand)

	rootCmd.AddCommand(sshCommand)
}
