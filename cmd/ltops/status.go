package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/mattermost/mattermost-load-test/kubernetes"
	"github.com/mattermost/mattermost-load-test/ltops"
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

		if cluster, err := LoadCluster(path); err != nil {
			logrus.Error(errors.Wrap(err, "Unable to load cluster "+file.Name()))
		} else {
			printStatusForCluster(cluster)
		}
	}

	return nil
}

const statusFormatString = `
--------------------------------------
Name: %v
Type: %v%v
SiteURL: %v
Metrics: %v
DBConnectionString: %v
RR0ConnectionString: %v
Load Test Profile: %v w/ %v users
Bulk Loaded Data: %v
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
	dbConnectionString := cluster.DBConnectionString()
	rrConnectionStrings := cluster.DBReaderConnectionStrings()
	rrConnnectionString := ""
	if len(rrConnectionStrings) > 0 {
		rrConnnectionString = rrConnectionStrings[0]
	}
	metrics, _ := cluster.GetMetricsAddr()

	release := ""
	if cluster.Type() == kubernetes.CLUSTER_TYPE {
		release = "\nRelease: " + cluster.(*kubernetes.Cluster).Release()
	}

	fmt.Printf(statusFormatString,
		cluster.Name(),
		cluster.Type(),
		release,
		cluster.SiteURL(),
		metrics,
		dbConnectionString,
		rrConnnectionString,
		cluster.Configuration().Profile,
		cluster.Configuration().Users,
		cluster.Configuration().BulkLoadComplete,
		len(app),
		cluster.DBInstanceCount(),
		len(proxy),
		len(loadtest),
	)
}

func init() {
	rootCmd.AddCommand(status)
}
