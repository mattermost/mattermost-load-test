package ops

import (
	"github.com/nu7hatch/gouuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
)

func DeleteCluster(name string) error {
	clusterInfo, err := LoadClusterInfo(name)
	if err != nil {
		return errors.Wrap(err, "unable to load cluster info")
	}

	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return errors.Wrap(err, "unable to load AWS config")
	}

	requestUUID, err := uuid.NewV4()
	if err != nil {
		return errors.Wrap(err, "unable to generate request UUID")
	}
	requestToken := "mattermost-load-test-ops-" + requestUUID.String()

	cf := cloudformation.New(cfg)
	req := cf.DeleteStackRequest(&cloudformation.DeleteStackInput{
		ClientRequestToken: aws.String(requestToken),
		StackName:          aws.String(clusterInfo.CloudFormationStackId),
	})

	_, err = req.Send()
	if err != nil {
		return errors.Wrap(err, "unable to delete stack")
	}

	logrus.Info("deleting cluster...")

	if stack, err := monitorCloudFormationStack(cf, clusterInfo.CloudFormationStackId, requestToken); err != nil || stack.StackStatus != cloudformation.StackStatusDeleteComplete {
		return errors.Wrap(err, "stack deletion failed")
	}

	return DeleteClusterInfo(name)
}
