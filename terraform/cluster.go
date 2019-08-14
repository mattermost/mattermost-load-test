package terraform

import (
	"os"

	"github.com/mattermost/mattermost-load-test/ltops"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Cluster struct {
	Config           *ltops.ClusterConfig
	SSHPrivateKeyPEM []byte
	DBPassword       string
	Env              *TerraformEnvironment
}

func (c *Cluster) Name() string {
	return c.Config.Name
}

func (c *Cluster) Type() string {
	return c.Config.Type
}

func (c *Cluster) Configuration() *ltops.ClusterConfig {
	return c.Config
}

func (c *Cluster) SSHKey() []byte {
	return c.SSHPrivateKeyPEM
}

func (c *Cluster) SiteURL() string {
	addrs, err := c.GetProxyInstancesAddrs()
	if err != nil || len(addrs) < 1 {
		logrus.Error("SiteURL: Unable to get proxy instance addresses.")
		return ""
	}
	return "http://" + addrs[0]
}

func (c *Cluster) GetAppInstancesAddrs() ([]string, error) {
	params, err := c.Env.getOutputParams()
	if err != nil {
		return nil, err
	}
	instanceIps := params.InstanceIp.Value

	return instanceIps, nil
}

func (c *Cluster) GetLoadtestInstancesAddrs() ([]string, error) {
	params, err := c.Env.getOutputParams()
	if err != nil {
		return nil, err
	}
	instanceIps := params.LoadtestInstanceIp.Value
	return instanceIps, nil
}

func (c *Cluster) GetProxyInstancesAddrs() ([]string, error) {
	params, err := c.Env.getOutputParams()
	if err != nil {
		return nil, err
	}
	instanceIps := params.ProxyIp.Value

	return instanceIps, nil
}

func (c *Cluster) GetMetricsAddr() (string, error) {
	params, err := c.Env.getOutputParams()
	if err != nil {
		return "", err
	}

	return params.MetricsIp.Value, nil
}

func (c *Cluster) DBDriverName() string {
	switch c.Config.DBEngineType {
	case "aurora-postgresql":
		return "postgres"
	case "aurora", "aurora-mysql":
		return "mysql"
	default:
		logrus.Errorf("Unable to get db driver name, invalid db-engine-type %v", c.Config.DBEngineType)
		return ""
	}
}

func (c *Cluster) DBConnectionString() string {
	params, err := c.Env.getOutputParams()
	if err != nil {
		logrus.Error("Unable to get output parameters for DBConnectionString")
		return ""
	}
	databaseEndpoint := params.DBEndpoint.Value
	switch c.Config.DBEngineType {
	case "aurora-postgresql":
		return "postgres://mmuser:" + c.DBPassword + "@" + databaseEndpoint + ":5432/mattermost?sslmode=disable\u0026connect_timeout=30"
	case "aurora", "aurora-mysql":
		return "mmuser:" + c.DBPassword + "@tcp(" + databaseEndpoint + ":3306)/mattermost?charset=utf8mb4,utf8&readTimeout=20s&writeTimeout=20s&timeout=20s"
	default:
		logrus.Errorf("Unable to get endpoint, invalid db-engine-type %v", c.Config.DBEngineType)
		return ""
	}
}

func (c *Cluster) DBReaderConnectionStrings() []string {
	params, err := c.Env.getOutputParams()
	if err != nil {
		logrus.Error("Unable to get output parameters for DBConnectionString")
		return nil
	}
	databaseEndpoint := params.DBReaderEndpoint.Value
	switch c.Config.DBEngineType {
	case "aurora-postgresql":
		return []string{"postgres://mmuser:" + c.DBPassword + "@" + databaseEndpoint + ":5432/mattermost?sslmode=disable\u0026connect_timeout=30"}
	case "aurora", "aurora-mysql":
		return []string{"mmuser:" + c.DBPassword + "@tcp(" + databaseEndpoint + ":3306)/mattermost?charset=utf8mb4,utf8&readTimeout=20s&writeTimeout=20s&timeout=20s"}
	default:
		logrus.Errorf("Unable to get endpoint, invalid db-engine-type %v", c.Config.DBEngineType)
		return nil
	}
}

func (c *Cluster) DBInstanceCount() int {
	return c.Config.DBInstanceCount
}

func (c *Cluster) DBSettings() (*ltops.DBSettings, error) {
	params, err := c.Env.getOutputParams()
	if err != nil {
		return nil, errors.Wrap(err, "unable to get output parameters for DBConnectionString")
	}

	var port int

	switch c.Config.DBEngineType {
	case "aurora-postgresql":
		port = 5432
	default:
		port = 3306
	}

	return &ltops.DBSettings{
		Username: "mmuser",
		Password: c.DBPassword,
		Endpoint: params.DBEndpoint.Value,
		Port:     port,
		Database: "mattermost",
	}, nil
}

func (c *Cluster) Destroy() error {
	logrus.Info("Destroying cluster...")
	if err := c.Env.destroy(); err != nil {
		return errors.Wrap(err, "Unable to destroy terraform cluster.")
	}

	logrus.Info("Cleaning up files...")
	return os.RemoveAll(c.Env.WorkingDirectory)
}
