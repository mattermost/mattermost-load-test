package kubernetes

import (
	"fmt"
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
	previouslyDeployed := len(c.Release()) > 0
	if previouslyDeployed {
		log.Info("upgrading mattermost helm chart...")
	} else {
		log.Info("installing mattermost helm chart...")
	}

	license, err := ltops.GetFileOrURL(options.LicenseFile)
	if err != nil {
		return err
	}

	configFileLocation := ""

	if len(options.HelmConfigFile) == 0 {
		config, err := c.GetHelmConfigFromProfile(options.Profile, options.Users, string(license))
		if err != nil {
			return err
		}

		err = saveChartConfig(config, c.Config.WorkingDirectory)
		if err != nil {
			return err
		}

		configFileLocation = filepath.Join(c.Config.WorkingDirectory, chartFilename)
	} else {
		configFileLocation = options.HelmConfigFile
	}

	if previouslyDeployed {
		cmd := exec.Command("helm", "upgrade", "-f", configFileLocation, c.Release(), "mattermost/mattermost-helm")
		if out, err := cmd.CombinedOutput(); err != nil {
			return errors.Wrap(err, "unable to install mattermost chart, error from helm: "+string(out))
		}

		// Delete the pods so they are recreated with any config changes
		cmd = exec.Command("kubectl", "delete", "po", "-l", fmt.Sprintf("release=%v", c.Release()))
		if out, err := cmd.CombinedOutput(); err != nil {
			return errors.Wrap(err, "unable to restart release pods, error from kubectl: "+string(out))
		}

		cmd = exec.Command("kubectl", "delete", "po", "-l", fmt.Sprintf("app=%v-mysqlha", c.Release()))
		if out, err := cmd.CombinedOutput(); err != nil {
			return errors.Wrap(err, "unable to restart mysqlha pods, error from kubectl: "+string(out))
		}

		log.Info("upgraded release '" + c.ReleaseName + "'")
	} else {
		cmd := exec.Command("helm", "install", "-f", configFileLocation, "mattermost/mattermost-helm")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return errors.Wrap(err, "unable to install mattermost chart, error from helm: "+string(out))
		}

		fields := strings.Fields(strings.Split(string(out), "\n")[0])
		c.ReleaseName = fields[1]

		log.Info("created release '" + c.Release() + "'")
	}

	if c.Configuration().Profile != options.Profile || c.Configuration().Users != options.Users {
		c.Config.BulkLoadComplete = false
	}

	if len(options.HelmConfigFile) == 0 {
		c.Config.Profile = options.Profile
	} else {
		c.Config.Profile = "custom helm config"
	}
	c.Config.Users = options.Users

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
