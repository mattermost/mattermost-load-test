package kubernetes

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/mattermost/mattermost-load-test/ltops"
)

type Cluster struct {
	Config      *ltops.ClusterConfig
	ReleaseName string
	Kubernetes  *kubernetes.Clientset `json:"-"`
}

func (c *Cluster) Name() string {
	return c.Config.Name
}

func (c *Cluster) Type() string {
	return c.Config.Type
}

func (c *Cluster) Release() string {
	return c.ReleaseName
}

func (c *Cluster) Configuration() *ltops.ClusterConfig {
	return c.Config
}

func (c *Cluster) SSHKey() []byte {
	return []byte{}
}

func (c *Cluster) SiteURL() string {
	if len(c.Release()) == 0 {
		return ""
	}

	svc, err := c.Kubernetes.CoreV1().Services("default").Get(c.Release()+"-nginx-ingress-controller", metav1.GetOptions{})
	if err != nil {
		return ""
	}

	ingressInstances := svc.Status.LoadBalancer.Ingress
	if len(ingressInstances) == 0 || ingressInstances[0].IP == "" {
		return "pending"
	}

	return ingressInstances[0].IP
}

func (c *Cluster) GetAppInstancesAddrs() ([]string, error) {
	if len(c.Release()) == 0 {
		return []string{}, nil
	}

	pods, err := c.Kubernetes.CoreV1().Pods("").List(metav1.ListOptions{LabelSelector: "release=" + c.Release() + ",app=mattermost-helm"})
	if err != nil {
		return nil, err
	}

	podNames := make([]string, len(pods.Items))
	for i, p := range pods.Items {
		podNames[i] = p.Name
	}

	return podNames, nil
}

func (c *Cluster) GetLoadtestInstancesAddrs() ([]string, error) {
	if len(c.Release()) == 0 {
		return []string{}, nil
	}

	pods, err := c.Kubernetes.CoreV1().Pods("").List(metav1.ListOptions{LabelSelector: "app=mattermost-helm-loadtest,release=" + c.Release()})
	if err != nil {
		return nil, err
	}

	podNames := make([]string, len(pods.Items))
	for i, p := range pods.Items {
		podNames[i] = p.Name
	}

	return podNames, nil
}

func (c *Cluster) GetProxyInstancesAddrs() ([]string, error) {
	if len(c.Release()) == 0 {
		return []string{}, nil
	}

	pods, err := c.Kubernetes.CoreV1().Pods("").List(metav1.ListOptions{LabelSelector: "release=" + c.Release() + ",app=nginx-ingress,component=controller"})
	if err != nil {
		return nil, err
	}

	podNames := make([]string, len(pods.Items))
	for i, p := range pods.Items {
		podNames[i] = p.Name
	}

	return podNames, nil
}

func (c *Cluster) GetMetricsAddr() (string, error) {
	if len(c.Release()) == 0 {
		return "", nil
	}

	svc, err := c.Kubernetes.CoreV1().Services("default").Get(c.Release()+"-grafana", metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	ingressInstances := svc.Status.LoadBalancer.Ingress
	if len(ingressInstances) == 0 || ingressInstances[0].IP == "" {
		return "pending", nil
	}

	return ingressInstances[0].IP, nil
}

func (c *Cluster) GetMetricsPodName() (string, error) {
	if len(c.Release()) == 0 {
		return "", nil
	}

	pods, err := c.Kubernetes.CoreV1().Pods("").List(metav1.ListOptions{LabelSelector: "app=" + c.Release() + "-grafana"})
	if err != nil {
		return "", err
	}

	if len(pods.Items) == 0 {
		return "", errors.New("no grafana pods running")
	}

	return pods.Items[0].Name, nil
}

func (c *Cluster) DBConnectionString() string {
	if len(c.Release()) == 0 {
		return ""
	}
	return fmt.Sprintf("mmuser:passwd@tcp(%v-mysqlha-0.%v-mysqlha:3306)/mattermost?charset=utf8mb4,utf8&readTimeout=20s&writeTimeout=20s&timeout=20s", c.Release(), c.Release())
}

func (c *Cluster) DBReaderConnectionStrings() []string {
	if len(c.Release()) == 0 {
		return []string{}
	}
	return []string{fmt.Sprintf("mmuser:passwd@tcp(%v-mysqlha-readonly:3306)/mattermost?charset=utf8mb4,utf8&readTimeout=20s&writeTimeout=20s&timeout=20s", c.Release())}
}

func (c *Cluster) DBInstanceCount() int {
	if len(c.Release()) == 0 {
		return 0
	}

	pods, err := c.Kubernetes.CoreV1().Pods("").List(metav1.ListOptions{LabelSelector: "app=" + c.Release() + "-mysqlha"})
	if err != nil {
		return 0
	}

	return len(pods.Items)
}

func (c *Cluster) Destroy() error {
	log.Info("Destroying cluster...")

	cmd := exec.Command("helm", "del", "--purge", c.Release())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrap(err, "unable to delete release, error from helm: "+string(out))
	}

	return os.RemoveAll(c.Configuration().WorkingDirectory)
}
