package terraform

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

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

func loadCluster(dir string) (*Cluster, error) {
	path := filepath.Join(dir, infoFilename)
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read cluster info")
	}

	var cluster *Cluster
	if err := json.Unmarshal(b, &cluster); err != nil {
		return nil, errors.Wrap(err, "unable to unmarshal cluster info")
	}

	return cluster, nil
}

func deleteCluster(name string, dir string) error {
	path := filepath.Join(dir, name)
	return os.RemoveAll(path)
}
