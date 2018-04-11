package main

/*var loadTest = &cobra.Command{
	Use:   "loadtest -- [args...]",
	Short: "Runs a mattermost-load-test command againt the given cluster",
	RunE: func(cmd *cobra.Command, args []string) error {
		clusterName, _ := cmd.Flags().GetString("cluster-name")
		config, _ := cmd.Flags().GetString("config")

		clusterService, err := createTerraformClusterService()
		if err != nil {
			return err
		}

		cluster, err := clusterService.LoadCluster(clusterName)
		if err != nil {
			return errors.Wrap(err, "Couldn't load cluster")
		}

		return clusterService.LoadtestCluster(cluster, config, args)
	},
}

func init() {
	deploy.Flags().StringP("cluster", "c", "", "cluster name (required)")
	deploy.MarkFlagRequired("cluster")

	loadTest.Flags().StringP("config", "f", "", "a config file to use instead of the default (the ConnectionConfiguration section is mostly ignored)")

	rootCmd.AddCommand(loadTest)
}*/
