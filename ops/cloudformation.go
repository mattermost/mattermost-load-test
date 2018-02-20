package ops

import (
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/cloudformationiface"
)

func monitorCloudFormationStack(cf cloudformationiface.CloudFormationAPI, stackId, requestToken string) (cloudformation.StackStatus, error) {
	seenEventIds := make(map[string]bool)

	waitDuration := time.Second

	for {
		descStackReq := cf.DescribeStacksRequest(&cloudformation.DescribeStacksInput{
			StackName: aws.String(stackId),
		})

		descStacksResp, err := descStackReq.Send()
		if err != nil {
			return "", errors.Wrap(err, "unable to query stack status")
		}

		if len(descStacksResp.Stacks) != 1 {
			return "", fmt.Errorf("expected single stack in describe response")
		}

		status := descStacksResp.Stacks[0].StackStatus

		descEventsReq := cf.DescribeStackEventsRequest(&cloudformation.DescribeStackEventsInput{
			StackName: aws.String(stackId),
		})

		descEventsResp, err := descEventsReq.Send()
		if err != nil {
			return "", errors.Wrap(err, "unable to query stack events")
		}

		for i := len(descEventsResp.StackEvents) - 1; i >= 0; i-- {
			event := descEventsResp.StackEvents[i]
			if aws.StringValue(event.ClientRequestToken) != requestToken || seenEventIds[*event.EventId] {
				continue
			}
			if reason := aws.StringValue(event.ResourceStatusReason); reason != "" {
				logrus.Debugf("%v %v: %v", aws.StringValue(event.LogicalResourceId), event.ResourceStatus, reason)
			} else {
				logrus.Debugf("%v %v", aws.StringValue(event.LogicalResourceId), event.ResourceStatus)
			}
			seenEventIds[*event.EventId] = true
			waitDuration = time.Second
		}

		if !strings.HasSuffix(string(status), "_IN_PROGRESS") {
			return status, nil
		}

		time.Sleep(waitDuration)

		if waitDuration < time.Second*10 {
			waitDuration += time.Second
		}
	}
}

const clusterCloudFormationTemplate = `
Description: Manages a Mattermost loadtest cluster
Mappings:
  Regions:
    us-east-1:
      AppImage: ami-97785bed
    us-east-2:
      AppImage: ami-f63b1193
    us-west-2:
      AppImage: ami-f2d3638a
    us-west-1:
      AppImage: ami-824c4ee2
    ca-central-1:
      AppImage: ami-a954d1cd
    eu-west-1:
      AppImage: ami-d834aba1
    eu-west-2:
      AppImage: ami-403e2524
    eu-west-3:
      AppImage: ami-8ee056f3
    eu-central-1:
      AppImage: ami-5652ce39
    ap-southeast-1:
      AppImage: ami-68097514
    ap-northeast-2:
      AppImage: ami-863090e8
    ap-northeast-1:
      AppImage: ami-ceafcba8
    ap-southeast-2:
      AppImage: ami-942dd1f6
    ap-south-1:
      AppImage: ami-531a4c3c
    sa-east-1:
      AppImage: ami-84175ae8
Parameters:
  AppInstanceCount:
    Type: Number
  AppInstanceType:
    Type: String
  DBInstanceType:
    Type: String
  DBPassword:
    Type: String
    NoEcho: true
  SSHAuthorizedKey:
    Type: String
Resources:
  AppAutoScalingGroup:
    Type: AWS::AutoScaling::AutoScalingGroup
    DependsOn:
      - InternetGatewayAttachment
      - Subnet1RouteTableAssociation
      - Subnet2RouteTableAssociation
    CreationPolicy:
      ResourceSignal:
        Count: !Ref AppInstanceCount
        Timeout: PT10M
    UpdatePolicy:
      AutoScalingReplacingUpdate:
        WillReplace: true
    Properties:
      DesiredCapacity: !Ref AppInstanceCount
      HealthCheckType: EC2
      LaunchConfigurationName: !Ref AppLaunchConfiguration
      MaxSize: !Ref AppInstanceCount
      MinSize: !Ref AppInstanceCount
      VPCZoneIdentifier:
        - !Ref Subnet1
        - !Ref Subnet2
  AppInstanceSecurityGroup:
    Type: AWS::EC2::SecurityGroup
    Properties:
      GroupDescription: app instance security group
      VpcId: !Ref VPC
  AppLaunchConfiguration:
    Type: AWS::AutoScaling::LaunchConfiguration
    Properties:
      AssociatePublicIpAddress: true
      ImageId: !FindInMap [Regions, !Ref 'AWS::Region', AppImage]
      InstanceType: !Ref AppInstanceType
      SecurityGroups:
        - !Ref AppInstanceSecurityGroup
      UserData:
        Fn::Base64: !Sub |
          #!/bin/bash -xe
          yum install -y aws-cfn-bootstrap
          mkdir -p /home/ec2-user/.ssh
          echo '${SSHAuthorizedKey}' > /home/ec2-user/.ssh/authorized_keys
          /opt/aws/bin/cfn-signal -e $? --stack ${AWS::StackName} --resource AppAutoScalingGroup --region ${AWS::Region}
  Database:
    Type: AWS::RDS::DBInstance
    Properties:
      AllocatedStorage: '100'
      AutoMinorVersionUpgrade: false
      BackupRetentionPeriod: '0'
      DBInstanceClass: !Ref DBInstanceType
      DBName: loadtest
      DBSubnetGroupName: !Ref DatabaseSubnetGroup
      Engine: MySQL
      EngineVersion: '5.7'
      MasterUsername: loadtest
      MasterUserPassword: !Ref DBPassword
      PubliclyAccessible: true
      PreferredMaintenanceWindow: Sun:14:00-Sun:14:30
      Tags:
        - Key: Name
          Value: !Ref AWS::StackName
      VPCSecurityGroups:
        - !Ref DatabaseSecurityGroup
  DatabaseSecurityGroup:
    Type: AWS::EC2::SecurityGroup
    Properties:
      GroupDescription: database security group
      VpcId: !Ref VPC
  DatabaseSubnetGroup:
    Type: AWS::RDS::DBSubnetGroup
    Properties:
      DBSubnetGroupDescription: database subnet group
      SubnetIds:
        - Ref: Subnet1
        - Ref: Subnet2
  InternetGateway:
    Type: AWS::EC2::InternetGateway
  InternetGatewayAttachment:
    Type: AWS::EC2::VPCGatewayAttachment
    Properties:
      InternetGatewayId: !Ref InternetGateway
      VpcId: !Ref VPC
  RouteTable:
    Type: AWS::EC2::RouteTable
    Properties:
      VpcId: !Ref VPC
  RouteToInternet:
    Type: AWS::EC2::Route
    Properties:
      RouteTableId: !Ref RouteTable
      DestinationCidrBlock: 0.0.0.0/0
      GatewayId: !Ref InternetGateway
  Subnet1:
    Type: AWS::EC2::Subnet
    Properties:
      AvailabilityZone: !Select 
        - 0
        - Fn::GetAZs: !Ref AWS::Region
      CidrBlock: 10.0.0.0/20
      VpcId: !Ref VPC
  Subnet1RouteTableAssociation:
    Type: AWS::EC2::SubnetRouteTableAssociation
    Properties: 
      RouteTableId: !Ref RouteTable
      SubnetId: !Ref Subnet1
  Subnet2:
    Type: AWS::EC2::Subnet
    Properties:
      AvailabilityZone: !Select 
        - 1
        - Fn::GetAZs: !Ref AWS::Region
      CidrBlock: 10.0.16.0/20
      VpcId: !Ref VPC
  Subnet2RouteTableAssociation:
    Type: AWS::EC2::SubnetRouteTableAssociation
    Properties: 
      RouteTableId: !Ref RouteTable
      SubnetId: !Ref Subnet2
  VPC:
    Type: AWS::EC2::VPC
    Properties:
      CidrBlock: 10.0.0.0/16
      EnableDnsHostnames: true
      InstanceTenancy: dedicated
      Tags:
        - Key: Name
          Value: !Ref AWS::StackName
`
