package ltops

// DeployOptions defines the possible options when deploying a Mattermost load test cluster.
type DeployOptions struct {
	MattermostBinaryFile string // file path or URL to Mattermost binary to use
	LicenseFile          string // file path or URL to Mattermost enterprise license
	LoadTestBinaryFile   string // file path or URL to load test agent binary
	Profile              string // the profile to load test
	Users                int    // the number of active users to simulate
	HelmConfigFile       string // path to helm chart config file
}
