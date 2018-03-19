package ops

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

func Deploy(distributionPath, clusterName, licenseFile string) error {
	clusterInfo, err := LoadClusterInfo(clusterName)
	if err != nil {
		return errors.Wrap(err, "unable to load cluster info")
	}

	instances, err := ClusterAppInstances(clusterInfo)
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
			if err := deployToAppInstance(distributionPath, licenseFile, clusterInfo, instance, logrus.WithField("instance-id", aws.StringValue(instance.InstanceId))); err != nil {
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

func deployToAppInstance(distributionPath, licenseFile string, clusterInfo *ClusterInfo, instance *ec2.Instance, logger logrus.FieldLogger) error {
	client, err := sshClient(clusterInfo, instance)
	if err != nil {
		return errors.Wrap(err, "unable to connect to server via ssh")
	}
	defer client.Close()

	logger.Debug("uploading distribution...")
	remoteDistributionPath := "/tmp/mattermost.tar.gz"
	if err := uploadFile(client, distributionPath, remoteDistributionPath); err != nil {
		return errors.Wrap(err, "unable to upload distribution")
	}

	if err := uploadSystemdFile(client); err != nil {
		return errors.Wrap(err, "unable to upload systemd file")
	}

	for _, cmd := range []string{
		"sudo rm -rf mattermost /opt/mattermost",
		"tar -xvzf /tmp/mattermost.tar.gz",
		"sudo mv mattermost /opt",
		"mkdir -p /opt/mattermost/data",
		"sudo yum install -y jq",
	} {
		logger.Debug("+ " + cmd)
		if err := remoteCommand(client, cmd); err != nil {
			return errors.Wrap(err, "error running command: "+cmd)
		}
	}

	logger.Debug("uploading license file...")
	remoteLicenseFilePath := "/opt/mattermost/config/mattermost.mattermost-license"
	if err := uploadFile(client, licenseFile, remoteLicenseFilePath); err != nil {
		return errors.Wrap(err, "unable to upload license file")
	}

	for k, v := range map[string]interface{}{
		".ServiceSettings.ListenAddress":       ":80",
		".ServiceSettings.LicenseFileLocation": remoteLicenseFilePath,
		".ServiceSettings.SiteURL":             "http://" + clusterInfo.CloudFormationStackOutputs["LoadBalancerDNSName"],
		".ServiceSettings.EnableAPIv3":         true,
		".SqlSettings.DriverName":              "mysql",
		".SqlSettings.DataSource":              clusterInfo.DatabaseConnectionString(),
		".ClusterSettings.Enable":              true,
		".ClusterSettings.ClusterName":         "load-test",
		".ClusterSettings.ReadOnlyConfig":      false,
		".MetricsSettings.Enable":              true,
		".MetricsSettings.BlockProfileRate":    1,
	} {
		logger.Debug("updating config: " + k)
		jsonValue, err := json.Marshal(v)
		if err != nil {
			return errors.Wrap(err, "invalid config value for key: "+k)
		}
		if err := remoteCommand(client, fmt.Sprintf(`jq '%s = %s' /opt/mattermost/config/config.json > /tmp/mmcfg.json && mv /tmp/mmcfg.json /opt/mattermost/config/config.json`, k, string(jsonValue))); err != nil {
			return errors.Wrap(err, "error updating config: "+k)
		}
	}

	for _, cmd := range []string{
		"sudo setcap cap_net_bind_service=+ep /opt/mattermost/bin/platform",
		"sudo systemctl daemon-reload",
		"sudo systemctl restart mattermost.service",
		"sudo systemctl enable mattermost.service",
	} {
		logger.Debug("+ " + cmd)
		if err := remoteCommand(client, cmd); err != nil {
			return errors.Wrap(err, "error running command: "+cmd)
		}
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

func remoteCommand(client *ssh.Client, cmd string) error {
	session, err := client.NewSession()
	if err != nil {
		return errors.Wrap(err, "unable to create ssh session")
	}
	defer session.Close()

	if err := session.Run(cmd); err != nil {
		return err
	}

	return nil
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
	if err := session.Run("cat > " + shellQuote(destination)); err != nil {
		return err
	}

	return nil
}

func uploadSystemdFile(client *ssh.Client) error {
	session, err := client.NewSession()
	if err != nil {
		return errors.Wrap(err, "unable to create ssh session")
	}
	defer session.Close()

	service := `
[Unit]
Description=Mattermost
After=network.target

[Service]
Type=simple
ExecStart=/opt/mattermost/bin/platform
Restart=always
RestartSec=10
WorkingDirectory=/opt/mattermost
User=ec2-user
Group=ec2-user
LimitNOFILE=49152

[Install]
WantedBy=multi-user.target
`

	session.Stdin = strings.NewReader(strings.TrimSpace(service))
	if err := session.Run("cat | sudo tee /lib/systemd/system/mattermost.service"); err != nil {
		return err
	}

	return nil
}

func ClusterAppInstances(clusterInfo *ClusterInfo) ([]*ec2.Instance, error) {
	cfg, err := LoadAWSConfig()
	if err != nil {
		return nil, errors.Wrap(err, "unable to load AWS config")
	}

	ec2svc := ec2.New(cfg)
	req := ec2svc.DescribeInstancesRequest(&ec2.DescribeInstancesInput{
		Filters: []ec2.Filter{
			{
				Name:   aws.String("instance-state-name"),
				Values: []string{"running"},
			},
			{
				Name:   aws.String("tag:aws:cloudformation:stack-id"),
				Values: []string{clusterInfo.CloudFormationStackId},
			},
			{
				Name:   aws.String("tag:mattermost-load-test-app-instance"),
				Values: []string{"true"},
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
