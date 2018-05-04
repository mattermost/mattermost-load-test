package terraform

import "github.com/mattermost/mattermost-load-test/ltops"

type terraformParameters struct {
	ClusterName           string `json:"cluster_name"`
	AppInstanceType       string `json:"app_instance_type"`
	AppInstanceCount      int    `json:"app_instance_count"`
	DBInstanceType        string `json:"db_instance_type"`
	DBInstanceCount       int    `json:"db_instance_count"`
	DBPassword            string `json:"db_password"`
	LoadtestInstanceCount int    `json:"loadtest_instance_count"`
	SSHPublicKey          string `json:"ssh_public_key"`
	SSHPrivateKey         string `json:"ssh_private_key"`
}

func terraformParametersFromClusterConfig(config *ltops.ClusterConfig, dbPassword string, sshPublicKey string, sshPrivateKey string) *terraformParameters {
	return &terraformParameters{
		ClusterName:           config.Name,
		AppInstanceType:       config.AppInstanceType,
		AppInstanceCount:      config.AppInstanceCount,
		DBInstanceCount:       config.DBInstanceCount,
		DBInstanceType:        config.DBInstanceType,
		LoadtestInstanceCount: config.LoadtestInstanceCount,
		DBPassword:            dbPassword,
		SSHPublicKey:          sshPublicKey,
		SSHPrivateKey:         sshPrivateKey,
	}
}

type terraformOutputParameters struct {
	InstanceIp struct {
		Value []string
	}
	LoadtestInstanceIp struct {
		Value []string
	}
	ProxyIp struct {
		Value []string
	}
	DBEndpoint struct {
		Value string
	}
	DBReaderEndpoint struct {
		Value string
	}
	S3bucket struct {
		Value string
	}
	S3bucketRegion struct {
		Value string
	}
	S3AccessKeyId struct {
		Value string
	}
	S3AccessKeySecret struct {
		Value string
	}
	MetricsIp struct {
		Value string
	}
}
