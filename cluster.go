package ltops

type ClusterConfig struct {
	Name                  string
	AppInstanceType       string
	AppInstanceCount      int
	DBInstanceType        string
	DBInstanceCount       int
	LoadtestInstanceCount int
}

type Cluster interface {
	Name() string
	DBConnectionString() string
	DBReaderConnectionStrings() []string
	SSHKey() []byte
	SiteURL() string
	GetAppInstancesAddrs() ([]string, error)
	GetLoadtestInstancesAddrs() ([]string, error)
	GetProxyInstancesAddrs() ([]string, error)
}

type ClusterService interface {
	CreateCluster(cfg *ClusterConfig) (Cluster, error)
	DeleteCluster(cluster Cluster) error
	LoadCluster(clusterName string) (Cluster, error)
	DeployMattermost(cluster Cluster, mattermostFile string, licenceFile string) error
	//ModifyMattermostConfig(cluster Cluster, mattermostConfig string) error
	DeployLoadtests(cluster Cluster, loadtestsFile string) error
	//ModifyLoadtestConfig(cluster Cluster, loadtestsFile string) error
	//LoadtestCluster(cluster Cluster) error
}
