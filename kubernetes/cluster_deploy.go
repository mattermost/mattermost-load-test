package kubernetes

import (
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"

	"github.com/mattermost/mattermost-load-test/ltops"
)

type ChartConfig struct {
	Global   *GlobalConfig   `yaml:"global"`
	MySQLHA  *MySQLHAConfig  `yaml:"mysqlha"`
	App      *AppConfig      `yaml:"mattermost-app"`
	Loadtest *LoadtestConfig `yaml:"mattermost-loadtest"`
}

type GlobalConfig struct {
	SiteURL           string          `yaml:"siteUrl"`
	MattermostLicense string          `yaml:"mattermostLicense"`
	Features          *FeaturesConfig `yaml:"features"`
}

type FeaturesConfig struct {
	LoadTest *LoadTestFeature `yaml:"loadTest"`
	Grafana  *GrafanFeature   `yaml:"grafana"`
}

type LoadTestFeature struct {
	Enabled bool `yaml:"enabled"`
}

type GrafanFeature struct {
	Enabled bool `yaml:"enabled"`
}

type MySQLHAConfig struct {
	Enabled bool            `yaml:"enabled"`
	Options *MySQLHAOptions `yaml:"mysqlha"`
}

type MySQLHAOptions struct {
	ReplicaCount int               `yaml:"replicaCount"`
	ConfigFiles  *MySQLConfigFiles `yaml:"configFiles"`
}

type MySQLConfigFiles struct {
	Master string `yaml:"master.cnf"`
	Slave  string `yaml:"slave.cnf"`
}

type AppConfig struct {
	ReplicaCount int           `yaml:"replicaCount"`
	Image        *ImageSetting `yaml:"image"`
}

type LoadtestConfig struct {
	ReplicaCount                      int           `yaml:"replicaCount"`
	Image                             *ImageSetting `yaml:"image"`
	NumTeams                          int           `yaml:"numTeams"`
	NumChannelsPerTeam                int           `yaml:"numChannelsPerTeam"`
	NumUsers                          int           `yaml:"numUsers"`
	SkipBulkLoad                      bool          `yaml:"skipBulkLoad"`
	TestLengthMinutes                 int           `yaml:"testLengthMinutes"`
	NumActiveEntities                 int           `yaml:"numActiveEntities"`
	ActionRateMilliseconds            int           `yaml:"actionRateMilliseconds"`
	ActionRateMaxVarianceMilliseconds int           `yaml:"actionRateMaxVarianceMilliseconds"`
}

type ImageSetting struct {
	Tag string `yaml:"tag"`
}

// TODO: Replace with an argument or config option when load test profiles are added
const NUM_USERS = 30000

const masterMySQLConfig = `
[mysqld]
log_bin
skip_name_resolve
max_connections = 300
`

const slaveMySQLConfig = `
[mysqld]
super_read_only
skip_name_resolve
slave_parallel_workers = 100
slave_parallel_type = LOGICAL_CLOCK
max_connections = 300
`

func (c *Cluster) DeployMattermost(mattermostFile string, licenceFileLocation string) error {
	log.Info("installing mattermost helm chart...")

	if len(c.ReleaseName) > 0 {
		log.Info("already installed as release '" + c.ReleaseName + "'")
		return nil
	}

	license, err := ltops.GetFileOrURL(licenceFileLocation)
	if err != nil {
		return err
	}

	config := &ChartConfig{
		Global: &GlobalConfig{
			SiteURL:           "http://localhost:8065",
			MattermostLicense: string(license),
			Features: &FeaturesConfig{
				&LoadTestFeature{Enabled: true},
				&GrafanFeature{Enabled: true},
			},
		},
		MySQLHA: &MySQLHAConfig{
			Enabled: true,
			Options: &MySQLHAOptions{
				ReplicaCount: c.Configuration().DBInstanceCount,
				ConfigFiles: &MySQLConfigFiles{
					Master: masterMySQLConfig,
					Slave:  slaveMySQLConfig,
				},
			},
		},
		App: &AppConfig{
			ReplicaCount: c.Configuration().AppInstanceCount,
			Image: &ImageSetting{
				Tag: "4.10.1",
			},
		},
		Loadtest: &LoadtestConfig{
			ReplicaCount: c.Configuration().LoadtestInstanceCount,
			Image: &ImageSetting{
				Tag: "4.10.1",
			},
			NumTeams:                          1,
			NumChannelsPerTeam:                400,
			NumUsers:                          NUM_USERS,
			SkipBulkLoad:                      true,
			TestLengthMinutes:                 20,
			NumActiveEntities:                 NUM_USERS / c.Configuration().LoadtestInstanceCount,
			ActionRateMilliseconds:            240000,
			ActionRateMaxVarianceMilliseconds: 15000,
		},
	}

	err = saveChartConfig(config, c.Config.WorkingDirectory)
	if err != nil {
		return err
	}

	cmd := exec.Command("helm", "install", "-f", filepath.Join(c.Config.WorkingDirectory, chartFilename), "mattermost/mattermost-helm")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrap(err, "unable to install mattermost chart, error from helm: "+string(out))
	}

	fields := strings.Fields(strings.Split(string(out), "\n")[0])
	c.ReleaseName = fields[1]

	log.Info("created release '" + c.ReleaseName + "'")

	err = saveCluster(c, c.Config.WorkingDirectory)
	if err != nil {
		return err
	}

	log.Info("...done")

	return nil
}

const chartFilename = "chartconfig.yaml"

func saveChartConfig(config *ChartConfig, dir string) error {
	b, err := yaml.Marshal(config)
	if err != nil {
		return errors.Wrap(err, "unable to marshal chart config")
	}

	path := filepath.Join(dir, chartFilename)
	if err := ioutil.WriteFile(path, b, 0600); err != nil {
		return errors.Wrap(err, "unable to write chart config")
	}

	return nil
}

// Not applicable to kubernetes
func (c *Cluster) DeployLoadtests(loadtestsDistLocation string) error {
	return nil
}
