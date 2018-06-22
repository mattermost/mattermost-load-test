package kubernetes

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func (c *Cluster) Connect() error {
	if c.Kubernetes != nil {
		return nil
	}

	loader := clientcmd.NewDefaultClientConfigLoadingRules()

	config, err := clientcmd.BuildConfigFromKubeconfigGetter("", loader.Load)
	if err != nil {
		return err
	}

	c.Kubernetes, err = kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	return nil
}
