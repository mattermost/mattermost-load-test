package kubernetes

import (
	"fmt"

	"github.com/mattermost/mattermost-load-test/ltops"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const masterMySQLConfig = `[mysqld]
log_bin
skip_name_resolve
max_connections = 300
innodb_log_buffer_size = 128M
`

const slaveMySQLConfig = `[mysqld]
super_read_only
skip_name_resolve
slave_parallel_workers = 100
slave_parallel_type = LOGICAL_CLOCK
max_connections = 300
`

func cpu(value int64) *Quantity {
	return &Quantity{resource.NewQuantity(value, resource.DecimalSI)}
}

func memory(valueInGiB int64) *Quantity {
	return &Quantity{resource.NewQuantity(valueInGiB*1024*1024*1024, resource.BinarySI)}
}

func getStandardConfig(users int) *ChartConfig {
	config := &ChartConfig{
		Global: &GlobalConfig{
			Features: &FeaturesConfig{
				Ingress: &IngressFeature{Enabled: true},
				LoadTest: &LoadTestFeature{
					Enabled: true,
					Image: &ImageSetting{
						Tag: "4.10.1",
					},
					Resources:                         &ResourcesSetting{Requests: &ResourceSetting{}},
					NumTeams:                          1,
					NumChannelsPerTeam:                400,
					NumUsers:                          users,
					ReplyChance:                       0.3,
					LinkPreviewChance:                 0.2,
					SkipBulkLoad:                      true,
					TestLengthMinutes:                 20,
					ActionRateMilliseconds:            240000,
					ActionRateMaxVarianceMilliseconds: 15000,
				},
				Database: &DatabaseFeature{
					UseInternal: true,
					Internal: &DatabaseInternal{
						DBUser:     "mmuser",
						DBPassword: "passwd",
						DBName:     "mattermost",
					},
				},
				Grafana:      &GrafanaFeature{Enabled: true},
				LinkPreviews: &LinkPreviewFeature{Enabled: true},
				CustomEmoji:  &CustomEmojiFeature{Enabled: true},
				Storage:      &StorageFeature{Enabled: true},
			},
		},
		MySQLHA: &MySQLHAConfig{
			Enabled: true,
			Options: &MySQLHAOptions{
				ConfigFiles: &MySQLConfigFiles{
					Master: masterMySQLConfig,
					Slave:  slaveMySQLConfig,
				},
			},
			Resources: &ResourcesSetting{Requests: &ResourceSetting{}},
		},
		App: &AppConfig{
			Image: &ImageSetting{
				Tag: "4.10.1",
			},
			Resources: &ResourcesSetting{Requests: &ResourceSetting{}},
		},
		Proxy: &ProxyConfig{
			Enabled: true,
			Controller: &ProxyController{
				Resources: &ResourcesSetting{Requests: &ResourceSetting{}},
			},
		},
		Prometheus: &PrometheusConfig{
			Enabled: true,
		},
	}

	config.Global.Features.LoadTest.NumUsers = users

	// TODO: replace with non-flubbed numbers
	if users <= 5000 {
		config.App.ReplicaCount = 2
		config.App.Resources.Requests.CPU = cpu(2)
		config.App.Resources.Requests.Memory = memory(4)
		config.MySQLHA.Options.ReplicaCount = 2
		config.MySQLHA.Resources.Requests.CPU = cpu(2)
		config.MySQLHA.Resources.Requests.Memory = memory(4)
		config.Proxy.Controller.ReplicaCount = 1
		config.Proxy.Controller.Resources.Requests.CPU = cpu(2)
		config.Proxy.Controller.Resources.Requests.Memory = memory(4)
		config.Global.Features.LoadTest.ReplicaCount = 1
		config.Global.Features.LoadTest.Resources.Requests.CPU = cpu(2)
		config.Global.Features.LoadTest.Resources.Requests.Memory = memory(4)
		config.Global.Features.LoadTest.NumPosts = 5000000
	} else if users <= 10000 {
		config.App.ReplicaCount = 2
		config.App.Resources.Requests.CPU = cpu(4)
		config.App.Resources.Requests.Memory = memory(8)
		config.MySQLHA.Options.ReplicaCount = 2
		config.MySQLHA.Resources.Requests.CPU = cpu(4)
		config.MySQLHA.Resources.Requests.Memory = memory(8)
		config.Proxy.Controller.ReplicaCount = 2
		config.Proxy.Controller.Resources.Requests.CPU = cpu(2)
		config.Proxy.Controller.Resources.Requests.Memory = memory(4)
		config.Global.Features.LoadTest.ReplicaCount = 2
		config.Global.Features.LoadTest.Resources.Requests.CPU = cpu(2)
		config.Global.Features.LoadTest.Resources.Requests.Memory = memory(4)
		config.Global.Features.LoadTest.NumPosts = 10000000
	} else if users <= 20000 {
		config.App.ReplicaCount = 4
		config.App.Resources.Requests.CPU = cpu(4)
		config.App.Resources.Requests.Memory = memory(8)
		config.MySQLHA.Options.ReplicaCount = 4
		config.MySQLHA.Resources.Requests.CPU = cpu(4)
		config.MySQLHA.Resources.Requests.Memory = memory(16)
		config.Proxy.Controller.ReplicaCount = 3
		config.Proxy.Controller.Resources.Requests.CPU = cpu(2)
		config.Proxy.Controller.Resources.Requests.Memory = memory(4)
		config.Global.Features.LoadTest.ReplicaCount = 3
		config.Global.Features.LoadTest.Resources.Requests.CPU = cpu(2)
		config.Global.Features.LoadTest.Resources.Requests.Memory = memory(4)
		config.Global.Features.LoadTest.NumPosts = 20000000
	} else if users <= 30000 {
		config.App.ReplicaCount = 4
		config.App.Resources.Requests.CPU = cpu(4)
		config.App.Resources.Requests.Memory = memory(8)
		config.MySQLHA.Options.ReplicaCount = 4
		config.MySQLHA.Resources.Requests.CPU = cpu(4)
		config.MySQLHA.Resources.Requests.Memory = memory(32)
		config.Proxy.Controller.ReplicaCount = 4
		config.Proxy.Controller.Resources.Requests.CPU = cpu(2)
		config.Proxy.Controller.Resources.Requests.Memory = memory(4)
		config.Global.Features.LoadTest.ReplicaCount = 4
		config.Global.Features.LoadTest.Resources.Requests.CPU = cpu(2)
		config.Global.Features.LoadTest.Resources.Requests.Memory = memory(4)
		config.Global.Features.LoadTest.NumPosts = 30000000
	} else {
		config.App.ReplicaCount = 5
		config.App.Resources.Requests.CPU = cpu(4)
		config.App.Resources.Requests.Memory = memory(16)
		config.MySQLHA.Options.ReplicaCount = 6
		config.MySQLHA.Resources.Requests.CPU = cpu(4)
		config.MySQLHA.Resources.Requests.Memory = memory(64)
		config.Proxy.Controller.ReplicaCount = 6
		config.Proxy.Controller.Resources.Requests.CPU = cpu(2)
		config.Proxy.Controller.Resources.Requests.Memory = memory(8)
		config.Global.Features.LoadTest.ReplicaCount = 6
		config.Global.Features.LoadTest.Resources.Requests.CPU = cpu(4)
		config.Global.Features.LoadTest.Resources.Requests.Memory = memory(8)
		config.Global.Features.LoadTest.NumPosts = 60000000
	}

	config.Global.Features.LoadTest.NumActiveEntities = users / config.Global.Features.LoadTest.ReplicaCount

	return config
}

func (c *Cluster) GetHelmConfigFromProfile(profile string, users int, license string) (*ChartConfig, error) {
	var getConfigFunc func(int) *ChartConfig

	switch profile {
	case ltops.PROFILE_STANDARD:
		getConfigFunc = getStandardConfig
		break
	default:
		return nil, errors.New("unrecognized profile " + profile)
	}

	config := getConfigFunc(users)
	config.Global.MattermostLicense = license

	nodes, err := c.Kubernetes.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	totalCPUCapacity := cpu(0).Quantity
	totalMemoryCapacity := memory(0).Quantity
	for _, n := range nodes.Items {
		totalCPUCapacity.Add(*n.Status.Capacity.Cpu())
		totalMemoryCapacity.Add(*n.Status.Capacity.Memory())
	}

	totalCPURequests := config.TotalCPURequests()
	if totalCPUCapacity.Cmp(*totalCPURequests.Quantity) == -1 {
		return nil, errors.New(fmt.Sprintf("not enough cpu capacity in kubernetes cluster, capacity=%v cores, required=%v cores", totalCPUCapacity, totalCPURequests))
	}

	totalMemoryRequests := config.TotalMemoryRequests()
	if totalMemoryCapacity.Cmp(*totalMemoryRequests.Quantity) == -1 {
		return nil, errors.New(fmt.Sprintf("not enough memory capacity in kubernetes cluster, capacity=%v, required=%v", totalMemoryCapacity, totalMemoryRequests))
	}

	log.Info(fmt.Sprintf("%v profile with %v users requests %v cores and %v memory on the cluster", profile, users, totalCPURequests, totalMemoryRequests))

	return config, nil
}
