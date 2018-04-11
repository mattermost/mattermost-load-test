package main

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
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

		clusterService, err := createTerraformClusterService()
		if err != nil {
			return err
		}

		cluster, err := clusterService.LoadCluster(clusterName)
		if err != nil {
			return errors.Wrap(err, "Couldn't load cluster")
		}

		err = clusterService.DeployMattermost(cluster, mattermostFile, licenseFile)
		if err != nil {
			return errors.Wrap(err, "Couldn't deploy mattermost")
		}

		err = clusterService.DeployLoadtests(cluster, loadtestsFile)
		if err != nil {
			return errors.Wrap(err, "Couldn't deploy loadtests")
		}

		return nil
	},
}

func init() {
	deploy.Flags().StringP("cluster", "c", "", "cluster name (required)")
	deploy.MarkFlagRequired("cluster")

	deploy.Flags().StringP("mattermost", "m", "", "mattermost distribution to deploy. Can be local file or URL. (required)")
	deploy.MarkFlagRequired("mattermost")

	deploy.Flags().StringP("license", "l", "", "the license file to use (required)")
	deploy.MarkFlagRequired("license")

	deploy.Flags().StringP("loadtests", "t", "", "the loadtests package to use (required)")
	deploy.MarkFlagRequired("loadtests")

	rootCmd.AddCommand(deploy)
}
