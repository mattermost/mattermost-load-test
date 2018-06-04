package main

import (
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/mattermost/mattermost-load-test/terraform"
)

var deploy = &cobra.Command{
	Use:   "deploy",
	Short: "Deploys an app distribution to a load test cluster",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		licenseFile, _ := cmd.Flags().GetString("license")
		mattermostFile, _ := cmd.Flags().GetString("mattermost")
		clusterName, _ := cmd.Flags().GetString("cluster")
		loadtestsFile, _ := cmd.Flags().GetString("loadtests")

		workingDir, err := defaultWorkingDirectory()
		if err != nil {
			return err
		}

		cluster, err := LoadCluster(filepath.Join(workingDir, clusterName))
		if err != nil {
			return errors.Wrap(err, "Couldn't load cluster")
		}

		if cluster.Type() == terraform.CLUSTER_TYPE {
			if len(mattermostFile) == 0 {
				return errors.New("required flag \"mattermost\" not set")
			}
			if len(loadtestsFile) == 0 {
				return errors.New("required flag \"loadtests\" not set")
			}
		}

		err = cluster.DeployMattermost(mattermostFile, licenseFile)
		if err != nil {
			return errors.Wrap(err, "Couldn't deploy mattermost")
		}

		err = cluster.DeployLoadtests(loadtestsFile)
		if err != nil {
			return errors.Wrap(err, "Couldn't deploy loadtests")
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

	rootCmd.AddCommand(deploy)
}
