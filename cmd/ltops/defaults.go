package main

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/pkg/errors"
)

const subdirName = ".mattermost-load-test-ops"

func defaultWorkingDirectory() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}

	if u.HomeDir == "" {
		return "", fmt.Errorf("no home directory to place cluster info in")
	}

	dir := filepath.Join(u.HomeDir, subdirName)
	if err := os.MkdirAll(dir, 0700); err != nil && !os.IsExist(err) {
		return "", errors.Wrap(err, "unable to create cluster info directory")
	}

	return dir, nil
}
