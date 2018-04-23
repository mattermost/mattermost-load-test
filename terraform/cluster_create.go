package terraform

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	ltops "github.com/mattermost/mattermost-load-test-ops"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func CreateCluster(cfg *ltops.ClusterConfig) (ltops.Cluster, error) {
	dbPassword, err := generatePassword()
	if err != nil {
		return nil, errors.Wrap(err, "unable to generate database password")
	}

	sshPrivateKeyPEM, sshAuthorizedKey, err := generateSSHKey()
	if err != nil {
		return nil, errors.Wrap(err, "unable to generate ssh key")
	}

	terraformParameters := terraformParametersFromClusterConfig(cfg, dbPassword, string(sshAuthorizedKey), string(sshPrivateKeyPEM))
	env, err := newTerraformEnvironment(cfg.WorkingDirectory, terraformParameters)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to create terrafrom environment.")
	}

	logrus.Info("creating cluster...")

	if err := env.apply(); err != nil {
		return nil, errors.Wrap(err, "Unable to run apply for create cluster")
	}

	cluster := &Cluster{
		Config:           cfg,
		SSHPrivateKeyPEM: sshPrivateKeyPEM,
		DBPassword:       dbPassword,
		Env:              env,
	}

	err = saveCluster(cluster, env.WorkingDirectory)
	if err != nil {
		return nil, err
	}

	return cluster, nil
}

const infoFilename = "clusterinfo.json"

func saveCluster(cluster *Cluster, dir string) error {
	b, err := json.Marshal(cluster)
	if err != nil {
		return errors.Wrap(err, "unable to marshal cluster")
	}

	path := filepath.Join(dir, infoFilename)
	if err := ioutil.WriteFile(path, b, 0600); err != nil {
		return errors.Wrap(err, "unable to write cluster")
	}

	return nil
}
