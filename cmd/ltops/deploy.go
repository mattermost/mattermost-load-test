package main

import (
	"path/filepath"

	"github.com/mattermost/mattermost-server/v5/mlog"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/mattermost/mattermost-load-test/kubernetes"
	"github.com/mattermost/mattermost-load-test/ltops"
	"github.com/mattermost/mattermost-load-test/terraform"
)

var deploy = &cobra.Command{
	Use:     "deploy",
	Short:   "Deploys an app distribution to a load test cluster",
	Args:    cobra.ExactArgs(0),
	PreRunE: showHelpIfNoFlags,
	RunE: func(cmd *cobra.Command, args []string) error {
		deployOptions := &ltops.DeployOptions{}

		clusterName, _ := cmd.Flags().GetString("cluster")
		deployOptions.LicenseFile, _ = cmd.Flags().GetString("license")
		deployOptions.MattermostBinaryFile, _ = cmd.Flags().GetString("mattermost")
		deployOptions.LoadTestBinaryFile, _ = cmd.Flags().GetString("loadtests")
		deployOptions.Users, _ = cmd.Flags().GetInt("users")
		deployOptions.HelmConfigFile, _ = cmd.Flags().GetString("helm-config")

		workingDir, err := defaultWorkingDirectory()
		if err != nil {
			return err
		}

		cluster, err := LoadCluster(filepath.Join(workingDir, clusterName))
		if err != nil {
			return errors.Wrap(err, "Couldn't load cluster")
		}
		mlog.Info("Deploying to cluster...")
		if cluster.Type() == terraform.CLUSTER_TYPE {
			if len(deployOptions.MattermostBinaryFile) != 0 && len(deployOptions.LicenseFile) == 0 {
				return errors.New("required flag \"license\" not set")
			}
			if len(deployOptions.MattermostBinaryFile) == 0 && len(deployOptions.LoadTestBinaryFile) == 0 {
				return errors.New("one of \"mattermost\" or \"loadtest\" must be set for a terraform cluster")
			}
			if deployOptions.Users > 0 {
				return errors.New("flag \"users\" not supported for type " + cluster.Type())
			}
		} else if cluster.Type() == kubernetes.CLUSTER_TYPE {
			if len(deployOptions.MattermostBinaryFile) > 0 {
				return errors.New("flag \"mattermost\" not supported for type " + cluster.Type())
			}
			if len(deployOptions.LoadTestBinaryFile) > 0 {
				return errors.New("flag \"loadtests\" not supported for type " + cluster.Type())
			}
			if deployOptions.Users == 0 && len(deployOptions.HelmConfigFile) == 0 {
				return errors.New("one of flags \"users\" or \"helm-config\" must be set for a kubernetes cluster")
			}
		}

		// TODO: stop hard-coding and add flag when we have multiple profiles
		deployOptions.Profile = ltops.PROFILE_STANDARD

		err = cluster.Deploy(deployOptions)
		if err != nil {
			return errors.Wrap(err, "Couldn't deploy load test cluster")
		}
		mlog.Info("Deployed to cluster successfully!")
		return nil
	},
}

func init() {
	deploy.Flags().StringP("cluster", "c", "", "cluster name (required)")
	deploy.MarkFlagRequired("cluster")

	deploy.Flags().StringP("mattermost", "m", "", "mattermost distribution: local file, URL, 'master', branch or PR# (terraform)")
	deploy.Flags().StringP("license", "l", "", "the license file: local file or URL (required with --mattermost)")
	deploy.Flags().StringP("loadtests", "t", "", "the loadtests package: local file, URL, or 'master' (terraform)")

	deploy.Flags().IntP("users", "u", 0, "number of active users in the load test (kubernetes)")
	deploy.Flags().StringP("helm-config", "f", "", "custom helm configuration to use (kubernetes)")

	deploy.Flags().SortFlags = false

	rootCmd.AddCommand(deploy)
}
