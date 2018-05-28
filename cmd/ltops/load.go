package main

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-load-test/kubernetes"
	"github.com/mattermost/mattermost-load-test/ltops"
	"github.com/mattermost/mattermost-load-test/terraform"
)

type ClusterJson struct {
	Config *ltops.ClusterConfig
	Bytes  []byte
}

func (c *ClusterJson) UnmarshalJSON(b []byte) error {
	var data map[string]interface{}
	err := json.Unmarshal(b, &data)
	if err != nil {
		return err
	}

	config, ok := data["Config"]
	if !ok {
		return errors.New("missing cluster config")
	}

	configBytes, err := json.Marshal(config)
	if err != nil {
		return err
	}

	err = json.Unmarshal(configBytes, &c.Config)
	if err != nil {
		return err
	}

	c.Bytes = b

	return nil
}

func (c *ClusterJson) GetCluster() (ltops.Cluster, error) {
	clusterType := c.Config.Type

	if clusterType == terraform.CLUSTER_TYPE {
		var cluster *terraform.Cluster
		err := json.Unmarshal(c.Bytes, &cluster)
		if err != nil {
			return nil, err
		}

		return cluster, nil
	} else if clusterType == kubernetes.CLUSTER_TYPE {
		var cluster *kubernetes.Cluster
		err := json.Unmarshal(c.Bytes, &cluster)
		if err != nil {
			return nil, err
		}

		if len(cluster.Release()) > 0 {
			err = cluster.Connect()
			if err != nil {
				return nil, err
			}
		}

		return cluster, nil
	}

	return nil, errors.New("unrecognized cluster type: " + clusterType)
}

const infoFilename = "clusterinfo.json"

// Loads a cluster from a specific directory
func LoadCluster(dir string) (ltops.Cluster, error) {
	path := filepath.Join(dir, infoFilename)
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read cluster info")
	}

	var cluster *ClusterJson
	if err := json.Unmarshal(b, &cluster); err != nil {
		return nil, errors.Wrap(err, "unable to unmarshal cluster info")
	}

	return cluster.GetCluster()
}
