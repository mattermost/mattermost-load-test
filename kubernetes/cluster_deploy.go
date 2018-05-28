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
	ReplicaCount int `yaml:"replicaCount"`
}

type AppConfig struct {
	ReplicaCount int           `yaml:"replicaCount"`
	Image        *ImageSetting `yaml:"image"`
}

type LoadtestConfig struct {
	ReplicaCount int `yaml:"replicaCount"`
}

type ImageSetting struct {
	Tag string `yaml:"tag"`
}

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
			},
		},
		App: &AppConfig{
			ReplicaCount: c.Configuration().AppInstanceCount,
			Image: &ImageSetting{
				Tag: "4.9.2",
			},
		},
		Loadtest: &LoadtestConfig{
			ReplicaCount: c.Configuration().LoadtestInstanceCount,
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
