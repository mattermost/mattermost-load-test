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
	params, err := c.Env.getOuptutParams()
	if err != nil {
		return nil, err
	}
	instanceIps := params.InstanceIp.Value

	return instanceIps, nil
}

func (c *Cluster) GetLoadtestInstancesAddrs() ([]string, error) {
	params, err := c.Env.getOuptutParams()
	if err != nil {
		return nil, err
	}
	instanceIps := params.LoadtestInstanceIp.Value
	return instanceIps, nil
}

func (c *Cluster) GetProxyInstancesAddrs() ([]string, error) {
	params, err := c.Env.getOuptutParams()
	if err != nil {
		return nil, err
	}
	instanceIps := params.ProxyIp.Value

	return instanceIps, nil
}

func (c *Cluster) GetMetricsAddr() (string, error) {
	params, err := c.Env.getOuptutParams()
	if err != nil {
		return "", err
	}

	return params.MetricsIp.Value, nil
}

func (c *Cluster) DBConnectionString() string {
	params, err := c.Env.getOuptutParams()
	if err != nil {
		logrus.Error("Unable to get output parameters for DBConnectionString")
		return ""
	}
	databaseEndpoint := params.DBEndpoint.Value
	return "mmuser:" + c.DBPassword + "@tcp(" + databaseEndpoint + ":3306)/mattermost?charset=utf8mb4,utf8&readTimeout=20s&writeTimeout=20s&timeout=20s"
}

func (c *Cluster) DBReaderConnectionStrings() []string {
	params, err := c.Env.getOuptutParams()
	if err != nil {
		logrus.Error("Unable to get output parameters for DBConnectionString")
		return nil
	}
	databaseEndpoint := params.DBReaderEndpoint.Value
	return []string{"mmuser:" + c.DBPassword + "@tcp(" + databaseEndpoint + ":3306)/mattermost?charset=utf8mb4,utf8&readTimeout=20s&writeTimeout=20s&timeout=20s"}
}

func (c *Cluster) DBInstanceCount() int {
	return c.Config.DBInstanceCount
}

func (c *Cluster) Destroy() error {
	logrus.Info("Destroying cluster...")
	if err := c.Env.destroy(); err != nil {
		return errors.Wrap(err, "Unable to destroy terraform cluster.")
	}

	logrus.Info("Cleaning up files...")
	return os.RemoveAll(c.Env.WorkingDirectory)
}
