package kubernetes

import (
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"

	"github.com/mattermost/mattermost-load-test/ltops"
)

func (c *Cluster) Deploy(options *ltops.DeployOptions) error {
	log.Info("installing mattermost helm chart...")

	if len(c.ReleaseName) > 0 {
		log.Info("already installed as release '" + c.ReleaseName + "'")
		return nil
	}

	license, err := ltops.GetFileOrURL(options.LicenseFile)
	if err != nil {
		return err
	}

	config, err := c.GetHelmConfigFromProfile(options.Profile, options.Users, string(license))
	if err != nil {
		return err
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
