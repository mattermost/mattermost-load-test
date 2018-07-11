package kubernetes

import (
	"k8s.io/apimachinery/pkg/api/resource"
)

type ChartConfig struct {
	Global     *GlobalConfig     `yaml:"global"`
	Tags       *TagsConfig       `yaml:"tags"`
	MySQLHA    *MySQLHAConfig    `yaml:"mysqlha"`
	App        *AppConfig        `yaml:"mattermost-app"`
	Loadtest   *LoadtestConfig   `yaml:"mattermost-loadtest"`
	Proxy      *ProxyConfig      `yaml:"nginx-ingress"`
	Prometheus *PrometheusConfig `yaml:"prometheus"`
}

type TagsConfig struct {
	Core    bool `yaml:"core"`
	Metrics bool `yaml:"metrics"`
	Ingress bool `yaml:"ingress"`
	Storage bool `yaml:"storage"`
}

type GlobalConfig struct {
	SiteURL           string          `yaml:"siteUrl"`
	MattermostLicense string          `yaml:"mattermostLicense"`
	Features          *FeaturesConfig `yaml:"features"`
}

type FeaturesConfig struct {
	LoadTest     *LoadTestFeature    `yaml:"loadTest"`
	Grafana      *GrafanaFeature     `yaml:"grafana"`
	LinkPreviews *LinkPreviewFeature `yaml:"linkPreviews"`
	CustomEmoji  *CustomEmojiFeature `yaml:"customEmoji"`
}

type LoadTestFeature struct {
	Enabled bool `yaml:"enabled"`
}

type GrafanaFeature struct {
	Enabled bool `yaml:"enabled"`
}

type LinkPreviewFeature struct {
	Enabled bool `yaml:"enabled"`
}

type CustomEmojiFeature struct {
	Enabled bool `yaml:"enabled"`
}

type MySQLHAConfig struct {
	Enabled   bool              `yaml:"enabled"`
	Options   *MySQLHAOptions   `yaml:"mysqlha"`
	Resources *ResourcesSetting `yaml:"resources"`
}

type MySQLHAOptions struct {
	ReplicaCount int               `yaml:"replicaCount"`
	ConfigFiles  *MySQLConfigFiles `yaml:"configFiles"`
}

type MySQLConfigFiles struct {
	Master string `yaml:"master.cnf"`
	Slave  string `yaml:"slave.cnf"`
}

type AppConfig struct {
	ReplicaCount int               `yaml:"replicaCount"`
	Image        *ImageSetting     `yaml:"image"`
	Resources    *ResourcesSetting `yaml:"resources"`
}

type LoadtestConfig struct {
	ReplicaCount                      int               `yaml:"replicaCount"`
	Image                             *ImageSetting     `yaml:"image"`
	Resources                         *ResourcesSetting `yaml:"resources"`
	NumTeams                          int               `yaml:"numTeams"`
	NumChannelsPerTeam                int               `yaml:"numChannelsPerTeam"`
	NumUsers                          int               `yaml:"numUsers"`
	NumPosts                          int               `yaml:"numPosts"`
	ReplyChance                       float32           `yaml:"replyChance"`
	LinkPreviewChance                 float32           `yaml:"linkPreviewChance"`
	SkipBulkLoad                      bool              `yaml:"skipBulkLoad"`
	TestLengthMinutes                 int               `yaml:"testLengthMinutes"`
	NumActiveEntities                 int               `yaml:"numActiveEntities"`
	ActionRateMilliseconds            int               `yaml:"actionRateMilliseconds"`
	ActionRateMaxVarianceMilliseconds int               `yaml:"actionRateMaxVarianceMilliseconds"`
}

type ProxyConfig struct {
	Enabled    bool             `yaml:"enabled"`
	Controller *ProxyController `yaml:"controller"`
}

type ProxyController struct {
	ReplicaCount int               `yaml:"replicaCount"`
	Resources    *ResourcesSetting `yaml:"resources"`
}

type PrometheusConfig struct {
	Enabled bool `yaml:"enabled"`
}

type ImageSetting struct {
	Tag string `yaml:"tag"`
}

type ResourcesSetting struct {
	Limits   *ResourceSetting `yaml:"limits,omitempty"`
	Requests *ResourceSetting `yaml:"requests"`
}

type ResourceSetting struct {
	CPU    *Quantity `yaml:"cpu"`
	Memory *Quantity `yaml:"memory"`
}

type Quantity struct {
	*resource.Quantity
}

func (q *Quantity) MarshalYAML() (interface{}, error) {
	return q.String(), nil
}

func (c *ChartConfig) TotalCPURequests() *Quantity {
	total := cpu(0)
	for i := 0; i < c.App.ReplicaCount; i++ {
		total.Add(*c.App.Resources.Requests.CPU.Quantity)
	}
	for i := 0; i < c.MySQLHA.Options.ReplicaCount; i++ {
		total.Add(*c.MySQLHA.Resources.Requests.CPU.Quantity)
	}
	for i := 0; i < c.Proxy.Controller.ReplicaCount; i++ {
		total.Add(*c.Proxy.Controller.Resources.Requests.CPU.Quantity)
	}
	for i := 0; i < c.Loadtest.ReplicaCount; i++ {
		total.Add(*c.Loadtest.Resources.Requests.CPU.Quantity)
	}

	// Add two cores as buffer for other pods
	total.Add(*cpu(2).Quantity)
	return total
}

func (c *ChartConfig) TotalMemoryRequests() *Quantity {
	total := memory(0)
	for i := 0; i < c.App.ReplicaCount; i++ {
		total.Add(*c.App.Resources.Requests.Memory.Quantity)
	}
	for i := 0; i < c.MySQLHA.Options.ReplicaCount; i++ {
		total.Add(*c.MySQLHA.Resources.Requests.Memory.Quantity)
	}
	for i := 0; i < c.Proxy.Controller.ReplicaCount; i++ {
		total.Add(*c.Proxy.Controller.Resources.Requests.Memory.Quantity)
	}
	for i := 0; i < c.Loadtest.ReplicaCount; i++ {
		total.Add(*c.Loadtest.Resources.Requests.Memory.Quantity)
	}

	// Add two 2 GiB of memory as buffer for other pods
	total.Add(*memory(2).Quantity)
	return total
}
