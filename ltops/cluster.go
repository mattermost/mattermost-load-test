package ltops

type ClusterConfig struct {
	Name                  string
	Type                  string
	AppInstanceType       string
	AppInstanceCount      int
	DBInstanceType        string
	DBInstanceCount       int
	LoadtestInstanceCount int
	WorkingDirectory      string
	Profile               string
	Users                 int
	BulkLoadComplete      bool
}

// Cluster represents an active cluster
type Cluster interface {
	// Returns the name of the cluster
	Name() string

	// Returns the type of the cluster
	Type() string

	// Returns the current configuration of the cluster
	Configuration() *ClusterConfig

	// Returns the SSH private key to connect to the cluster's instances
	SSHKey() []byte

	// Returns the siteURL to connect to the cluster
	SiteURL() string

	// Retuns a slice of the IP addresses of the app server instances in this cluster
	GetAppInstancesAddrs() ([]string, error)

	// Retuns a slice of the IP addresses of the loadtest instances in this cluster
	GetLoadtestInstancesAddrs() ([]string, error)

	// Retuns a slice of the IP addresses of the proxy instances in this cluster
	GetProxyInstancesAddrs() ([]string, error)

	// Retuns the ip address of the metrics server
	GetMetricsAddr() (string, error)

	// Returns the master databame connection string
	DBConnectionString() string

	// Returns a list of all the read-replica database connection strings
	DBReaderConnectionStrings() []string

	// Returns a count of DB instances
	DBInstanceCount() int

	// Deploys a load test cluster
	Deploy(options *DeployOptions) error

	// Runs a loadtest
	Loadtest(options *LoadTestOptions) error

	// Destroys the cluster
	Destroy() error

	// Modifies the configuration of an active Mattermost deployment
	//ModifyMattermostConfig(cluster Cluster, mattermostConfig string) error

	// Runs loadtests against the cluster. Must have deployed mattermost and loadtests
	//ModifyMattermostConfig(cluster Cluster, mattermostConfig string) error
}
