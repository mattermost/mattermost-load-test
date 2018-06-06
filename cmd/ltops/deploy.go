package main

import (
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/mattermost/mattermost-load-test/kubernetes"
	"github.com/mattermost/mattermost-load-test/ltops"
	"github.com/mattermost/mattermost-load-test/terraform"
)

var deploy = &cobra.Command{
	Use:   "deploy",
	Short: "Deploys an app distribution to a load test cluster",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		deployOptions := &ltops.DeployOptions{}

		clusterName, _ := cmd.Flags().GetString("cluster")
		deployOptions.LicenseFile, _ = cmd.Flags().GetString("license")
		deployOptions.MattermostBinaryFile, _ = cmd.Flags().GetString("mattermost")
		deployOptions.LoadTestBinaryFile, _ = cmd.Flags().GetString("loadtests")
		deployOptions.Users, _ = cmd.Flags().GetInt("users")

		workingDir, err := defaultWorkingDirectory()
		if err != nil {
			return err
		}

		cluster, err := LoadCluster(filepath.Join(workingDir, clusterName))
		if err != nil {
			return errors.Wrap(err, "Couldn't load cluster")
		}

		if cluster.Type() == terraform.CLUSTER_TYPE {
			if len(deployOptions.MattermostBinaryFile) == 0 {
				return errors.New("required flag \"mattermost\" not set")
			}
			if len(deployOptions.LoadTestBinaryFile) == 0 {
				return errors.New("required flag \"loadtests\" not set")
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
			if deployOptions.Users == 0 {
				return errors.New("required flag \"users\" not set")
			}
		}

		// TODO: stop hard-coding and add flag when we have multiple profiles
		deployOptions.Profile = kubernetes.PROFILE_STANDARD

		err = cluster.Deploy(deployOptions)
		if err != nil {
			return errors.Wrap(err, "Couldn't deploy load test cluster")
		}

		return nil
	},
}

func init() {
	deploy.Flags().StringP("cluster", "c", "", "cluster name (required)")
	deploy.MarkFlagRequired("cluster")

	deploy.Flags().StringP("mattermost", "m", "", "mattermost distribution to deploy. Can be local file or URL. (required for terraform)")

	deploy.Flags().StringP("license", "l", "", "the license file to use (required)")
	deploy.MarkFlagRequired("license")

	deploy.Flags().StringP("loadtests", "t", "", "the loadtests package to use (required for terraform)")

	deploy.Flags().IntP("users", "u", 0, "the number of active users to configure the load test to run with (required for kubernetes)")

	rootCmd.AddCommand(deploy)
}
