package kubernetes

import (
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func (c *Cluster) Connect() error {
	if c.Kubernetes != nil {
		return nil
	}

	kubeconfig := filepath.Join(homeDir(), ".kube", "config")

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return err
	}

	c.Kubernetes, err = kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	return nil
}
