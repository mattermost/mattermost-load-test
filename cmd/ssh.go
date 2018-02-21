package cmd

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"github.com/mattermost/mattermost-load-test-ops/ops"
)

var ssh = &cobra.Command{
	Use:   "ssh [instance-id]",
	Short: "Connects to an app instance via SSH",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		clusterName, _ := cmd.Flags().GetString("cluster-name")

		clusterInfo, err := ops.LoadClusterInfo(clusterName)
		if err != nil {
			return errors.Wrap(err, "unable to load cluster info")
		}

		instances, err := ops.ClusterAppInstances(clusterInfo)
		if err != nil {
			return errors.Wrap(err, "unable to query cluster instances")
		}

		var instance *ec2.Instance

		if len(instances) > 1 {
			if len(args) > 0 {
				for _, candidate := range instances {
					if aws.StringValue(candidate.InstanceId) == args[0] {
						instance = candidate
						break
					}
				}
			} else {
				fmt.Printf("An instance id must be provided:\n\n")
				for _, instance := range instances {
					fmt.Printf("%s\n", aws.StringValue(instance.InstanceId))
				}
			}
		}

		if instance == nil {
			return fmt.Errorf("invalid selection")
		}

		return ops.SSH(clusterInfo, instance)
	},
}

func init() {
	ssh.Flags().String("cluster-name", "", "the name of the cluster to ssh to (required)")
	ssh.MarkFlagRequired("cluster-name")

	rootCmd.AddCommand(ssh)
}
