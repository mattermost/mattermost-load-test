package kubernetes

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/mattermost/mattermost-load-test/ltops"
)

const (
	CLUSTER_TYPE = "kubernetes"
)

func CreateCluster(cfg *ltops.ClusterConfig) (ltops.Cluster, error) {
	log.Info("checking kubernetes cluster...")
	log.Info("note you must already have an existing kubernetes cluster configured in your kubeconfig")

	cmd := exec.Command("kubectl", "get", "nodes")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.New("error running 'kubectl get nodes', make sure your kubeconfig is correct. error from kubectl: " + string(out))
	}

	log.Info("kubectl working and cluster exists")

	cmd = exec.Command("helm", "ls")
	err = cmd.Run()
	if err != nil {
		return nil, errors.Wrap(err, "unable to run helm")
	}

	cmd = exec.Command("helm", "repo", "add", "incubator", "https://kubernetes-charts-incubator.storage.googleapis.com/")
	err = cmd.Run()
	if err != nil {
		return nil, errors.Wrap(err, "unable to add incubator helm repo")
	}

	cmd = exec.Command("helm", "repo", "add", "mattermost", "https://helm.mattermost.com")
	err = cmd.Run()
	if err != nil {
		return nil, errors.Wrap(err, "unable to add mattermost helm repo")
	}

	log.Info("helm working and repos added")

	cfg.Type = CLUSTER_TYPE

	cluster := &Cluster{
		Config: cfg,
	}

	err = saveCluster(cluster, cfg.WorkingDirectory)
	if err != nil {
		return nil, err
	}

	log.Info("...done")

	return cluster, nil
}

const infoFilename = "clusterinfo.json"

func saveCluster(cluster *Cluster, dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

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

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
