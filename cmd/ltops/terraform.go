package main

import (
	ltops "github.com/mattermost/mattermost-load-test-ops"
	"github.com/mattermost/mattermost-load-test-ops/terraform"
)

func createTerraformClusterService() (ltops.ClusterService, error) {
	defaultDir, err := defaultWorkingDirectory()
	if err != nil {
		return nil, err
	}

	clusterService, err := terraform.NewClusterService(
		&terraform.ClusterServiceConfig{
			WorkingDirectory: defaultDir,
		},
	)
	if err != nil {
		return nil, err
	}

	return clusterService, nil
}
