package main

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           os.Args[0],
	SilenceErrors: true,
	SilenceUsage:  true,
	Long:          "Use ltops to easily spin up and load test a cluster of Mattermost servers with all the trimmings. Currently supports AWS and Kubernetes. For AWS, you must have your aws-cli configured and terraform installed. For Kubernetes, you must have a Kubernetes cluster configured in your kubeconfig and helm installed. See https://github.com/mattermost/mattermost-load-test/blob/master/README.md for more information.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if verbose, _ := cmd.Flags().GetBool("verbose"); verbose {
			logrus.SetLevel(logrus.DebugLevel)
		}
	},
}

func init() {
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "make output more verbose")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logrus.Fatal(err)
	}
}
