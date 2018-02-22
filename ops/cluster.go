package ops

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"

	"github.com/pkg/errors"
)

type ClusterInfo struct {
	CloudFormationStackId      string
	CloudFormationStackOutputs map[string]string
	DatabasePassword           string
	SSHKey                     []byte
}

func ClusterInfoDirectory() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}

	if u.HomeDir == "" {
		return "", fmt.Errorf("no home directory to place cluster info in")
	}

	dir := filepath.Join(u.HomeDir, ".mattermost-load-test-ops", "cluster-info")
	if err := os.MkdirAll(dir, 0700); err != nil && !os.IsExist(err) {
		return "", errors.Wrap(err, "unable to create cluster info directory")
	}

	return dir, nil
}

func SaveClusterInfo(name string, info *ClusterInfo) error {
	dir, err := ClusterInfoDirectory()
	if err != nil {
		return err
	}

	b, err := json.Marshal(info)
	if err != nil {
		return errors.Wrap(err, "unable to marshal cluster info")
	}

	path := filepath.Join(dir, name)
	if err := ioutil.WriteFile(path, b, 0600); err != nil {
		return errors.Wrap(err, "unable to write cluster info")
	}

	return nil
}

func LoadClusterInfo(name string) (*ClusterInfo, error) {
	dir, err := ClusterInfoDirectory()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(dir, name)
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read cluster info")
	}

	var info *ClusterInfo
	if err := json.Unmarshal(b, &info); err != nil {
		return nil, errors.Wrap(err, "unable to unmarshal cluster info")
	}

	return info, nil
}

func DeleteClusterInfo(name string) error {
	dir, err := ClusterInfoDirectory()
	if err != nil {
		return err
	}

	path := filepath.Join(dir, name)
	return os.RemoveAll(path)
}
