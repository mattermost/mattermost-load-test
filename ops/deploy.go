package ops

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

func Deploy(distributionPath, clusterName string) error {
	clusterInfo, err := LoadClusterInfo(clusterName)
	if err != nil {
		return errors.Wrap(err, "unable to load cluster info")
	}

	sshSigner, err := ssh.ParsePrivateKey(clusterInfo.SSHKey)
	if err != nil {
		return errors.Wrap(err, "unable to parse ssh private key")
	}

	instances, err := clusterAppInstances(clusterInfo)
	if err != nil {
		return errors.Wrap(err, "unable to query cluster instances")
	}

	var wg sync.WaitGroup
	wg.Add(len(instances))

	failed := new(int32)

	for _, instance := range instances {
		instance := instance
		go func() {
			logrus.Infof("deploying to %v...", aws.StringValue(instance.InstanceId))
			if err := deployToAppInstance(distributionPath, instance, sshSigner); err != nil {
				wrapped := errors.Wrap(err, "unable to deploy to "+aws.StringValue(instance.InstanceId))
				logrus.Error(wrapped)
				atomic.AddInt32(failed, 1)
			} else {
				logrus.Infof("successfully deployed to %v", aws.StringValue(instance.InstanceId))
			}
			wg.Done()
		}()
	}

	wg.Wait()

	if *failed == 1 {
		return fmt.Errorf("failed to deploy to 1 instance")
	} else if *failed > 0 {
		return fmt.Errorf("failed to deploy to %v instances", *failed)
	}
	return nil
}

func deployToAppInstance(distributionPath string, instance *ec2.Instance, sshSigner ssh.Signer) error {
	client, err := ssh.Dial("tcp", aws.StringValue(instance.PublicIpAddress)+":22", &ssh.ClientConfig{
		User: "ec2-user",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(sshSigner),
		},
		// TODO: get and save host key from console output after instance creation
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
	if err != nil {
		return errors.Wrap(err, "unable to connect to server via ssh")
	}
	defer client.Close()

	remoteDistributionPath := "/tmp/mattermost-load-test-ops-" + filepath.Base(distributionPath)
	if err := uploadFile(client, distributionPath, remoteDistributionPath); err != nil {
		return errors.Wrap(err, "unable to upload distribution")
	}

	return nil
}

func shellQuote(s string) string {
	if strings.ContainsAny(s, `'\`) {
		// TODO
		panic("shell quoting not actually implemented. don't use weird paths")
	}
	return "'" + s + "'"
}

func uploadFile(client *ssh.Client, source, destination string) error {
	f, err := os.Open(source)
	if err != nil {
		return errors.Wrap(err, "unable to open source file")
	}
	defer f.Close()

	session, err := client.NewSession()
	if err != nil {
		return errors.Wrap(err, "unable to create ssh session")
	}
	defer session.Close()

	session.Stdin = f
	if err := session.Run(fmt.Sprintf("cat > %v", shellQuote(destination))); err != nil {
		return err
	}

	return nil
}

func clusterAppInstances(clusterInfo *ClusterInfo) ([]*ec2.Instance, error) {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return nil, errors.Wrap(err, "unable to load AWS config")
	}

	ec2svc := ec2.New(cfg)
	req := ec2svc.DescribeInstancesRequest(&ec2.DescribeInstancesInput{
		Filters: []ec2.Filter{
			{
				Name:   aws.String("tag:aws:cloudformation:stack-id"),
				Values: []string{clusterInfo.CloudFormationStackId},
			},
			{
				Name:   aws.String("tag:aws:cloudformation:logical-id"),
				Values: []string{"AppAutoScalingGroup"},
			},
		},
	})

	resp, err := req.Send()
	if err != nil {
		return nil, errors.Wrap(err, "unable to desribe ec2 instances")
	}

	var instances []*ec2.Instance
	for _, reservation := range resp.Reservations {
		for _, instance := range reservation.Instances {
			instances = append(instances, &instance)
		}
	}

	return instances, nil
}
