package main

import (
	"os"
	"path/filepath"

	"github.com/mattermost/mattermost-load-test/kubernetes"
	"github.com/mattermost/mattermost-load-test/ltops"
	"github.com/mattermost/mattermost-load-test/sshtools"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var rebootCommand = &cobra.Command{
	Use:              "reboot",
	Short:            "Reboot servers via SSH",
	TraverseChildren: true,
}

var rebootAppCommand = &cobra.Command{
	Use:     "app",
	Short:   "Reboot all app instances via SSH",
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

		addrs, err := cluster.GetAppInstancesAddrs()
		if err != nil {
			return errors.Wrap(err, "unable to get app instances")
		}

		for _, addr := range addrs {
			if err := reboot(cluster, "app instance", addr); err != nil {
				return err
			}
			logrus.Debugf("Rebooted: %s", addr)
		}

		return nil
	},
}

var rebootProxyCommand = &cobra.Command{
	Use:     "proxy",
	Short:   "Reboot proxy instances via SSH",
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

		addrs, err := cluster.GetProxyInstancesAddrs()
		if err != nil {
			return errors.Wrap(err, "unable to get proxy instances")
		}

		for _, addr := range addrs {
			if err := reboot(cluster, "proxy instance", addr); err != nil {
				return err
			}
			logrus.Debugf("Rebooted: %s", addr)
		}

		return nil
	},
}

var rebootLoadtestCommand = &cobra.Command{
	Use:     "loadtest",
	Short:   "Reboot loadtest instances via SSH",
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

		addrs, err := cluster.GetLoadtestInstancesAddrs()
		if err != nil {
			return errors.Wrap(err, "unable to get loadtest instances")
		}

		for _, addr := range addrs {
			if err := reboot(cluster, "loadtest instance", addr); err != nil {
				return err
			}
			logrus.Debugf("Rebooted: %s", addr)
		}

		return nil
	},
}

func reboot(cluster ltops.Cluster, description, addr string) error {
	logrus.Infof("Rebooting %s at %s", description, addr)

	if cluster.Type() == kubernetes.CLUSTER_TYPE {
		return sshtools.SSHInteractiveKubesPod(addr)
	}

	client, err := sshtools.SSHClient(cluster.SSHKey(), addr)
	if err != nil {
		return err
	}
	defer client.Close()

	_ = sshtools.RemoteCommand(client, "sudo reboot", os.Stdout) // ignore the error since reboot doesn't return
	return nil
}

func init() {
	rebootCommand.PersistentFlags().StringP("cluster", "c", "", "the name of the cluster (required)")
	rebootCommand.MarkPersistentFlagRequired("cluster")

	rebootCommand.AddCommand(rebootAppCommand, rebootLoadtestCommand, rebootProxyCommand)

	rebootCommand.Flags().SortFlags = false

	rootCmd.AddCommand(rebootCommand)
}
