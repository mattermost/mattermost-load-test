package ops

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/cloudwatchlogsiface"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/ecsiface"
)

func LoadTest(clusterName, configFile string, args []string) error {
	clusterInfo, err := LoadClusterInfo(clusterName)
	if err != nil {
		return errors.Wrap(err, "unable to load cluster info")
	}

	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return errors.Wrap(err, "unable to load AWS config")
	}

	logrus.Info("launching load test...")

	ecsSvc := ecs.New(cfg)

	env := make(map[string]string)

	if configFile != "" {
		var config map[string]interface{}

		f, err := os.Open(configFile)
		if err != nil {
			return errors.Wrap(err, "unable to open config file")
		}
		defer f.Close()

		if err := json.NewDecoder(f).Decode(&config); err != nil {
			return errors.Wrap(err, "unable to parse config file")
		}

		var inspect func(string, interface{})
		inspect = func(prefix string, v interface{}) {
			switch v := v.(type) {
			case map[string]interface{}:
				for k, v := range v {
					inspect(prefix+"_"+strings.ToUpper(k), v)
				}
			case float64:
				env[prefix] = strconv.FormatFloat(v, 'f', -1, 64)
			default:
				env[prefix] = fmt.Sprint(v)
			}
		}
		inspect("MMLOADTEST", config)
	}

	env["MMLOADTEST_CONNECTIONCONFIGURATION_SERVERURL"] = "http://" + clusterInfo.CloudFormationStackOutputs["LoadBalancerDNSName"]
	env["MMLOADTEST_CONNECTIONCONFIGURATION_WEBSOCKETURL"] = "ws://" + clusterInfo.CloudFormationStackOutputs["LoadBalancerDNSName"]
	env["MMLOADTEST_CONNECTIONCONFIGURATION_PPROFURL"] = "http://" + clusterInfo.CloudFormationStackOutputs["LoadBalancerDNSName"] + ":8067/debug/pprof"
	env["MMLOADTEST_CONNECTIONCONFIGURATION_DBENDPOINT"] = clusterInfo.DatabaseConnectionString()
	env["MMLOADTEST_CONNECTIONCONFIGURATION_LOCALCOMMANDS"] = "false"
	env["MMLOADTEST_CONNECTIONCONFIGURATION_SSHHOSTNAMEPORT"] = clusterInfo.CloudFormationStackOutputs["LoadBalancerDNSName"] + ":22"
	env["MMLOADTEST_CONNECTIONCONFIGURATION_SSHUSERNAME"] = "ec2-user"
	env["MMLOADTEST_CONNECTIONCONFIGURATION_SSHKEY"] = string(clusterInfo.SSHKey)
	env["MMLOADTEST_CONNECTIONCONFIGURATION_MATTERMOSTINSTALLDIR"] = "/opt/mattermost"
	env["MMLOADTEST_RESULTSCONFIGURATION_PPROFDELAYMINUTES"] = "0"

	var envPairs []ecs.KeyValuePair
	for k, v := range env {
		envPairs = append(envPairs, ecs.KeyValuePair{
			Name:  aws.String(k),
			Value: aws.String(v),
		})
	}

	req := ecsSvc.RunTaskRequest(&ecs.RunTaskInput{
		Cluster:    aws.String(clusterInfo.CloudFormationStackOutputs["LoadTestCluster"]),
		LaunchType: ecs.LaunchTypeFargate,
		NetworkConfiguration: &ecs.NetworkConfiguration{
			AwsvpcConfiguration: &ecs.AwsVpcConfiguration{
				AssignPublicIp: ecs.AssignPublicIpEnabled,
				SecurityGroups: []string{
					clusterInfo.CloudFormationStackOutputs["LoadTestSecurityGroup"],
				},
				Subnets: []string{
					clusterInfo.CloudFormationStackOutputs["Subnet1"],
				},
			},
		},
		Overrides: &ecs.TaskOverride{
			ContainerOverrides: []ecs.ContainerOverride{
				{
					Command:     args,
					Environment: envPairs,
					Name:        aws.String("loadtest"),
				},
			},
		},
		TaskDefinition: aws.String(clusterInfo.CloudFormationStackOutputs["LoadTestTaskDefinition"]),
	})

	resp, err := req.Send()
	if err != nil {
		return errors.Wrap(err, "unable to run ecs task")
	} else if len(resp.Failures) > 0 {
		return fmt.Errorf("failure running ecs task: " + aws.StringValue(resp.Failures[0].Reason))
	}

	_, err = followECSTask(ecsSvc, cloudwatchlogs.New(cfg), clusterInfo, aws.StringValue(resp.Tasks[0].TaskArn))
	if err != nil {
		return errors.Wrap(err, "unable to follow ecs task")
	}

	return nil
}

func followECSTask(ecsSvc ecsiface.ECSAPI, cwlSvc cloudwatchlogsiface.CloudWatchLogsAPI, clusterInfo *ClusterInfo, taskARN string) (taskOut *ecs.Task, errOut error) {
	c := make(chan os.Signal, 1)
	defer close(c)
	signal.Notify(c, os.Interrupt)
	defer signal.Stop(c)

	go func() {
		for {
			if _, ok := <-c; ok {
				ecsSvc.StopTaskRequest(&ecs.StopTaskInput{
					Cluster: aws.String(clusterInfo.CloudFormationStackOutputs["LoadTestCluster"]),
					Task:    aws.String(taskARN),
				}).Send()
			} else {
				break
			}
		}
	}()

	var nextToken *string

	waitDuration := time.Millisecond * 500

	for {
		descTaskReq := ecsSvc.DescribeTasksRequest(&ecs.DescribeTasksInput{
			Cluster: aws.String(clusterInfo.CloudFormationStackOutputs["LoadTestCluster"]),
			Tasks:   []string{taskARN},
		})

		descTasksResp, err := descTaskReq.Send()
		if err != nil {
			return nil, errors.Wrap(err, "unable to query taskARN status")
		}

		if len(descTasksResp.Failures) > 0 {
			return nil, fmt.Errorf("failed to describe task: " + aws.StringValue(descTasksResp.Failures[0].Reason))
		}

		getLogEventsReq := cwlSvc.GetLogEventsRequest(&cloudwatchlogs.GetLogEventsInput{
			LogGroupName:  aws.String(clusterInfo.CloudFormationStackOutputs["LoadTestLogGroup"]),
			LogStreamName: aws.String(clusterInfo.CloudFormationStackName() + "/loadtest/" + strings.Split(taskARN, "/")[1]),
			NextToken:     nextToken,
			StartFromHead: aws.Bool(true),
		})

		getLogEventsResp, err := getLogEventsReq.Send()

		if err != nil {
			if err, ok := err.(awserr.Error); !ok || err.Code() != "ResourceNotFoundException" {
				return nil, errors.Wrap(err, "unable to get log events")
			}
		} else {
			for _, event := range getLogEventsResp.Events {
				fmt.Println(aws.StringValue(event.Message))
			}
			nextToken = getLogEventsResp.NextForwardToken
		}

		if task := &descTasksResp.Tasks[0]; task.StoppedAt != nil {
			return task, nil
		}

		if getLogEventsResp != nil && len(getLogEventsResp.Events) > 0 {
			waitDuration = time.Millisecond * 500
		} else if waitDuration < 5*time.Second {
			waitDuration += time.Second
		}

		time.Sleep(waitDuration)

		if waitDuration < time.Second*10 {
			waitDuration += time.Second
		}
	}
}
