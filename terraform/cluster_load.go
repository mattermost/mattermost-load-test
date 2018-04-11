package terraform

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	ltops "github.com/mattermost/mattermost-load-test-ops"
	"github.com/pkg/errors"
)

// Loads a cluster from a specific directory
func LoadCluster(dir string) (ltops.Cluster, error) {
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
