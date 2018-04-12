package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	ltops "github.com/mattermost/mattermost-load-test-ops"
	"github.com/mattermost/mattermost-load-test-ops/terraform"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var status = &cobra.Command{
	Use:   "status",
	Short: "Prints some status infomation on clusters you have running.",
	RunE:  statusCmd,
}

func statusCmd(cmd *cobra.Command, args []string) error {
	workingDir, err := defaultWorkingDirectory()
	if err != nil {
		return err
	}

	files, err := ioutil.ReadDir(workingDir)
	if err != nil {
		return errors.Wrap(err, "Can't read directory where cluster info stored.")
	}

	for _, file := range files {
		path := filepath.Join(workingDir, file.Name())

		cluster, err := terraform.LoadCluster(path)
		if err != nil {
			logrus.Error(errors.Wrap(err, "Unable to load cluster "+file.Name()))
		}

		printStatusForCluster(cluster)
	}

	return nil
}

const statusFormatString = `
--------------------------------------
Name: %v
SiteURL: %v
Instances:
	APP:    %v
	DB:     %v
	PROXY:  %v
	LT:     %v
--------------------------------------

`

func printStatusForCluster(cluster ltops.Cluster) {
	app, _ := cluster.GetAppInstancesAddrs()
	proxy, _ := cluster.GetProxyInstancesAddrs()
	loadtest, _ := cluster.GetLoadtestInstancesAddrs()

	fmt.Printf(statusFormatString,
		cluster.Name(),
		cluster.SiteURL(),
		len(app),
		cluster.Configuration().DBInstanceCount,
		len(proxy),
		len(loadtest),
	)
}

func init() {
	rootCmd.AddCommand(status)
}
