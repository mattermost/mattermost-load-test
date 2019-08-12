package terraform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/mattermost/mattermost-load-test/ltops"
	"github.com/mattermost/mattermost-load-test/sshtools"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

func (c *Cluster) Deploy(options *ltops.DeployOptions) error {
	var wg sync.WaitGroup

	failed := make(chan bool)

	if len(options.MattermostBinaryFile) > 0 {
		c.DeployMattermost(&wg, failed, options.MattermostBinaryFile, options.LicenseFile)
	}

	if len(options.LoadTestBinaryFile) > 0 {
		c.DeployLoadtests(&wg, failed, options.LoadTestBinaryFile)
	}

	done := make(chan bool)
	failedCount := 0
	go func() {
		for range failed {
			failedCount++
		}

		close(done)
	}()

	wg.Wait()
	close(failed)
	<-done

	if failedCount > 0 {
		return fmt.Errorf("failed to deploy %d resources", failedCount)
	}

	return nil
}

func do(wg *sync.WaitGroup, failed chan bool, logger logrus.FieldLogger, action func() error) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		err := action()
		if err != nil {
			logger.Error(err.Error())
			failed <- true
		}
	}()
}

func (c *Cluster) DeployMattermost(wg *sync.WaitGroup, failed chan bool, mattermostDistLocation string, licenceFileLocation string) {
	do(wg, failed, logrus.StandardLogger(), func() error {
		appInstanceAddrs, err := c.GetAppInstancesAddrs()
		if err != nil || len(appInstanceAddrs) <= 0 {
			return errors.Wrap(err, "unable to get app instance addresses")
		}

		mattermostDist, err := ltops.GetMattermostFileOrURL(mattermostDistLocation)
		if err != nil {
			return errors.Wrap(err, "unable to load mattermost distribution")
		}

		licenseFile, err := ltops.GetFileOrURL(licenceFileLocation)
		if err != nil {
			return errors.Wrap(err, "unable to load mattermost license")
		}

		doDeploy(wg, failed, appInstanceAddrs, "app", func(instanceNum int, addr string, logger logrus.FieldLogger) error {
			return deployToAppInstance(bytes.NewReader(mattermostDist), bytes.NewReader(licenseFile), addr, c, logger)
		})

		return nil
	})

	do(wg, failed, logrus.StandardLogger(), func() error {
		proxyInstanceAddrs, err := c.GetProxyInstancesAddrs()
		if err != nil || len(proxyInstanceAddrs) <= 0 {
			return errors.Wrap(err, "unable to get app instance addresses")
		}

		doDeploy(wg, failed, proxyInstanceAddrs, "proxy", func(instanceNum int, addr string, logger logrus.FieldLogger) error {
			return deployToProxyInstance(addr, c, logger)
		})

		return nil
	})
}

func (c *Cluster) DeployLoadtests(wg *sync.WaitGroup, failed chan bool, loadtestsDistLocation string) {
	do(wg, failed, logrus.StandardLogger(), func() error {
		loadtestInstanceAddrs, err := c.GetLoadtestInstancesAddrs()
		if err != nil || len(loadtestInstanceAddrs) <= 0 {
			return errors.Wrap(err, "unable to get loadtest instance addresses")
		}

		loadtestsDist, err := ltops.GetLoadtestFileOrURL(loadtestsDistLocation)
		if err != nil {
			return errors.Wrap(err, "unable to load loadtests distribution")
		}

		doDeploy(wg, failed, loadtestInstanceAddrs, "loadtest", func(instanceNum int, addr string, logger logrus.FieldLogger) error {
			return deployToLoadtestInstance(instanceNum, addr, bytes.NewReader(loadtestsDist), c, logger)
		})

		return nil
	})
}

func doDeploy(wg *sync.WaitGroup, failed chan bool, addresses []string, addressesName string, deployFunc func(instanceNum int, addr string, logger logrus.FieldLogger) error) error {
	for instanceNum, instanceAddr := range addresses {
		instanceAddr := instanceAddr
		instanceNum := instanceNum

		logger := logrus.WithField("instance", instanceAddr)
		do(wg, failed, logger, func() error {
			logger.Infof("deploying to %s%d", addressesName, instanceNum)
			if err := deployFunc(instanceNum, instanceAddr, logger); err != nil {
				return errors.Wrapf(err, "unable to deploy to %s%d", addressesName, instanceNum)
			} else {
				logger.Infof("successfully deployed to %v%v", addressesName, instanceNum)
			}

			return nil
		})
	}
	return nil
}

var remoteSSHKeyPath = "/home/ubuntu/key.pem"

func deployToLoadtestInstance(instanceNum int, instanceAddr string, loadtestDistribution io.Reader, cluster ltops.Cluster, logger logrus.FieldLogger) error {
	debugLogWriter := newLogrusWriter(logger, logrus.DebugLevel)
	defer debugLogWriter.Close()

	client, err := sshtools.SSHClient(cluster.SSHKey(), instanceAddr)
	if err != nil {
		return errors.Wrap(err, "unable to connect to server via ssh")
	}
	defer client.Close()

	logger.Debug("uploading distribution...")
	remoteDistributionPath := "/home/ubuntu/mattermost-load-test.tar.gz"
	if err := sshtools.UploadReader(client, loadtestDistribution, remoteDistributionPath, debugLogWriter); err != nil {
		return errors.Wrap(err, "unable to upload loadtest distribution.")
	}

	if err := sshtools.UploadBytes(client, cluster.SSHKey(), remoteSSHKeyPath, debugLogWriter); err != nil {
		return errors.Wrap(err, "unable to upload ssh key")
	}

	for _, cmd := range []string{
		"sudo apt-get update",
		"sudo apt-get install -y jq",
		"curl -C - https://dl.google.com/go/go1.11.2.linux-amd64.tar.gz -o go1.11.2.linux-amd64.tar.gz",
		"sudo tar -C /usr/local -xzf go1.11.2.linux-amd64.tar.gz",
		"sudo ln -sf /usr/local/go/bin/go /usr/bin/go",
		"sudo apt-get install -y python-pydot python-pydot-ng graphviz",
		"sudo rm -rf /home/ubuntu/mattermost-load-test",
		"tar -xvzf /home/ubuntu/mattermost-load-test.tar.gz",
		"sudo chmod 600 /home/ubuntu/key.pem",
	} {
		logger.Debug("+ " + cmd)
		if err := sshtools.RemoteCommand(client, cmd, ioutil.Discard); err != nil {
			return errors.Wrap(err, "error running command: "+cmd)
		}
	}

	logger.Debug("uploading limits config")
	if err := uploadLimitsConfig(client); err != nil {
		return errors.Wrap(err, "Unable to upload limits config")
	}

	if err := configureLoadtestInstance(instanceNum, client, cluster, logger); err != nil {
		return errors.Wrap(err, "failed to configure loadtest instance")
	}

	if err := reboot(cluster, client, logger); err != nil {
		return errors.Wrap(err, "failed to reboot loadtest instance")
	}

	return nil
}

func configureLoadtestInstance(instanceNum int, client *ssh.Client, cluster ltops.Cluster, logger logrus.FieldLogger) error {
	proxyURLs, err := cluster.GetProxyInstancesAddrs()
	if err != nil || len(proxyURLs) < 1 {
		return errors.Wrap(err, "Couldn't get app instance addresses.")
	}

	appURLs, err := cluster.GetAppInstancesAddrs()
	if err != nil || len(appURLs) < 1 {
		return errors.Wrap(err, "Couldn't get app instance addresses.")
	}

	appURL := appURLs[0]

	siteURL, err := url.Parse(proxyURLs[instanceNum%len(proxyURLs)])
	if err != nil {
		return errors.Wrap(err, "Can't parse site URL")
	}

	serverURL := *siteURL
	serverURL.Scheme = "http"

	websocketURL := *siteURL
	websocketURL.Scheme = "ws"
	driverName := "mysql"
	if cluster.Configuration().DBEngineType == "aurora-postgresql" {
		driverName = "postgres"
	}

	for k, v := range map[string]interface{}{
		".ConnectionConfiguration.ServerURL":            serverURL.String(),
		".ConnectionConfiguration.WebsocketURL":         websocketURL.String(),
		".ConnectionConfiguration.PProfURL":             "http://" + appURL + ":8067/debug/pprof",
		".ConnectionConfiguration.DataSource":           cluster.DBConnectionString(),
		".ConnectionConfiguration.DriverName":           driverName,
		".ConnectionConfiguration.LocalCommands":        false,
		".ConnectionConfiguration.SSHHostnamePort":      appURL + ":22",
		".ConnectionConfiguration.SSHUsername":          "ubuntu",
		".ConnectionConfiguration.SSHKey":               remoteSSHKeyPath,
		".ConnectionConfiguration.MattermostInstallDir": "/opt/mattermost",
		".LoadtestEnviromentConfig.NumEmoji":            0,
		".LoadtestEnviromentConfig.NumPlugins":          0,
	} {
		logger.Debugf("updating config %s=%v", k, v)
		jsonValue, err := json.Marshal(v)
		if err != nil {
			return errors.Wrap(err, "invalid config value for key: "+k)
		}
		if err := sshtools.RemoteCommand(client, fmt.Sprintf(`jq '%s = %s' /home/ubuntu/mattermost-load-test/loadtestconfig.json > /tmp/ltconfig.json && mv /tmp/ltconfig.json /home/ubuntu/mattermost-load-test/loadtestconfig.json`, k, string(jsonValue)), ioutil.Discard); err != nil {
			return errors.Wrap(err, "error updating config: "+k)
		}
	}

	return nil
}

func deployToProxyInstance(instanceAddr string, clust ltops.Cluster, logger logrus.FieldLogger) error {
	client, err := sshtools.SSHClient(clust.SSHKey(), instanceAddr)
	if err != nil {
		return errors.Wrap(err, "unable to connect to server via ssh")
	}
	defer client.Close()

	appInstances, err := clust.GetAppInstancesAddrs()
	if err != nil {
		return errors.Wrap(err, "Unable to get app instance addresses.")
	}

	for _, cmd := range []string{
		"sudo apt-get update",
		"sudo apt-get install -y nginx",
	} {
		logger.Debug("+ " + cmd)
		if err := sshtools.RemoteCommand(client, cmd, ioutil.Discard); err != nil {
			return errors.Wrap(err, "error running command: "+cmd)
		}
	}

	if err := uploadNginxConfig(client, appInstances); err != nil {
		return errors.Wrap(err, "Unable to upload nginx config")
	}

	if err := uploadLimitsConfig(client); err != nil {
		return errors.Wrap(err, "Unable to upload limits config")
	}

	for _, cmd := range []string{
		"sudo ln -fs /etc/nginx/sites-available/mattermost /etc/nginx/sites-enabled/mattermost",
		"sudo rm -f /etc/nginx/sites-enabled/default",
		"sudo grep -q -F 'worker_rlimit_nofile' /etc/nginx/nginx.conf || echo 'worker_rlimit_nofile 65536;' | sudo tee -a /etc/nginx/nginx.conf",
		"sudo sed -i 's/worker_connections.*/worker_connections 200000;/g' /etc/nginx/nginx.conf",
		"sudo systemctl daemon-reload",
		"sudo systemctl restart nginx",
		"sudo systemctl enable nginx",
	} {
		logger.Debug("+ " + cmd)
		if err := sshtools.RemoteCommand(client, cmd, ioutil.Discard); err != nil {
			return errors.Wrap(err, "error running command: "+cmd)
		}
	}

	if err := reboot(clust, client, logger); err != nil {
		return errors.Wrap(err, "failed to reboot proxy instance")
	}

	return nil
}

func deployToAppInstance(mattermostDistribution, license io.Reader, instanceAddr string, clust *Cluster, logger logrus.FieldLogger) error {
	debugLogWriter := newLogrusWriter(logger, logrus.DebugLevel)
	defer debugLogWriter.Close()

	client, err := sshtools.SSHClient(clust.SSHKey(), instanceAddr)
	if err != nil {
		return errors.Wrap(err, "unable to connect to server via ssh")
	}
	defer client.Close()

	logger.Debug("uploading distribution")
	remoteDistributionPath := "/tmp/mattermost.tar.gz"
	if err := sshtools.UploadReader(client, mattermostDistribution, remoteDistributionPath, debugLogWriter); err != nil {
		return errors.Wrap(err, "unable to upload distribution")
	}

	if err := uploadSystemdFile(client); err != nil {
		return errors.Wrap(err, "unable to upload systemd file")
	}

	for _, cmd := range []string{
		"sudo rm -rf mattermost /opt/mattermost",
		"tar -xvzf /tmp/mattermost.tar.gz",
		"sudo mv mattermost /opt",
		"mkdir -p /opt/mattermost/data",
		"sudo apt-get update",
		"sudo apt-get install -y jq",
	} {
		logger.Debug("+ " + cmd)
		if err := sshtools.RemoteCommand(client, cmd, ioutil.Discard); err != nil {
			return errors.Wrapf(err, "error running command: %s", cmd)
		}
	}

	logger.Debug("uploading license file...")
	remoteLicenseFilePath := "/opt/mattermost/config/mattermost.mattermost-license"
	if err := sshtools.UploadReader(client, license, remoteLicenseFilePath, debugLogWriter); err != nil {
		return errors.Wrap(err, "unable to upload license file")
	}

	logger.Debug("uploading limits config...")
	if err := uploadLimitsConfig(client); err != nil {
		return errors.Wrap(err, "unable to upload limits config")
	}

	outputParams, err := clust.Env.getOutputParams()
	if err != nil {
		return errors.Wrap(err, "failed to get output parameters")
	}

	s3AccessKeyId := outputParams.S3AccessKeyId.Value
	s3AccessKeySecret := outputParams.S3AccessKeySecret.Value
	s3Bucket := outputParams.S3bucket.Value
	s3Region := outputParams.S3bucketRegion.Value
	driverName := "mysql"
	if clust.Configuration().DBEngineType == "aurora-postgresql" {
		driverName = "postgres"
	}
	for k, v := range map[string]interface{}{
		".ServiceSettings.ListenAddress":               ":80",
		".ServiceSettings.LicenseFileLocation":         remoteLicenseFilePath,
		".ServiceSettings.SiteURL":                     clust.SiteURL(),
		".ServiceSettings.EnableAPIv3":                 true,
		".ServiceSettings.EnableLinkPreviews":          true,
		".ServiceSettings.EnableSecurityFixAlert":      false,
		".SqlSettings.DriverName":                      driverName,
		".SqlSettings.DataSource":                      clust.DBConnectionString(),
		".SqlSettings.DataSourceReplicas":              clust.DBReaderConnectionStrings(),
		".SqlSettings.MaxOpenConns":                    3000,
		".SqlSettings.MaxIdleConns":                    200,
		".ClusterSettings.Enable":                      true,
		".ClusterSettings.ClusterName":                 "load-test",
		".ClusterSettings.ReadOnlyConfig":              false,
		".MetricsSettings.Enable":                      true,
		".MetricsSettings.BlockProfileRate":            1,
		".FileSettings.DriverName":                     "amazons3",
		".FileSettings.AmazonS3AccessKeyId":            s3AccessKeyId,
		".FileSettings.AmazonS3SecretAccessKey":        s3AccessKeySecret,
		".FileSettings.AmazonS3Bucket":                 s3Bucket,
		".FileSettings.AmazonS3Region":                 s3Region,
		".TeamSettings.MaxUsersPerTeam":                10000000,
		".TeamSettings.EnableOpenServer":               true,
		".TeamSettings.MaxChannelsPerTeam":             10000000,
		".ServiceSettings.EnableIncomingWehbooks":      true,
		".ServiceSettings.EnableOnlyAdminIntegrations": false,
		".PluginSettings.Enable":                       true,
		".PluginSettings.EnableUploads":                true,
		".LogSettings.EnableDiagnostics":               false,
	} {
		logger.Debug("updating config: " + k)
		jsonValue, err := json.Marshal(v)
		if err != nil {
			return errors.Wrap(err, "invalid config value for key: "+k)
		}
		if err := sshtools.RemoteCommand(client, fmt.Sprintf(`jq '%s = %s' /opt/mattermost/config/config.json > /tmp/mmcfg.json && mv /tmp/mmcfg.json /opt/mattermost/config/config.json`, k, string(jsonValue)), ioutil.Discard); err != nil {
			return errors.Wrap(err, "error updating config: "+k)
		}
	}

	for _, cmd := range []string{
		"sudo setcap cap_net_bind_service=+ep /opt/mattermost/bin/platform",
		"[[ -f /opt/mattermost/bin/mattermost ]] && sudo setcap cap_net_bind_service=+ep /opt/mattermost/bin/mattermost",
		"sudo systemctl daemon-reload",
		"sudo systemctl restart mattermost.service",
		"sudo systemctl enable mattermost.service",
	} {
		logger.Debug("+ " + cmd)
		if err := sshtools.RemoteCommand(client, cmd, ioutil.Discard); err != nil {
			return errors.Wrapf(err, "error running command: %s", cmd)
		}
	}

	if err := reboot(clust, client, logger); err != nil {
		return errors.Wrap(err, "failed to reboot app instance")
	}

	return nil
}

func uploadSystemdFile(client *ssh.Client) error {
	session, err := client.NewSession()
	if err != nil {
		return errors.Wrap(err, "unable to create ssh session")
	}
	defer session.Close()

	service := `
[Unit]
Description=Mattermost
After=network.target

[Service]
Type=simple
ExecStart=/opt/mattermost/bin/platform
Restart=always
RestartSec=10
WorkingDirectory=/opt/mattermost
User=ubuntu
Group=ubuntu
LimitNOFILE=49152

[Install]
WantedBy=multi-user.target
`

	session.Stdin = strings.NewReader(strings.TrimSpace(service))
	if err := session.Run("cat | sudo tee /lib/systemd/system/mattermost.service"); err != nil {
		return err
	}

	return nil
}

func uploadNginxConfig(client *ssh.Client, appInstanceAddrs []string) error {
	session, err := client.NewSession()
	if err != nil {
		return errors.Wrap(err, "unable to create ssh session")
	}
	defer session.Close()

	config := `
upstream backend {
%s
        keepalive 32;
}

proxy_cache_path /var/cache/nginx levels=1:2 keys_zone=mattermost_cache:10m max_size=3g inactive=120m use_temp_path=off;

server {
    listen 80;
    server_name _;

   location ~ /api/v[0-9]+/(users/)?websocket$ {
           proxy_set_header Upgrade $http_upgrade;
           proxy_set_header Connection "upgrade";
           client_max_body_size 50M;
           proxy_set_header Host $http_host;
           proxy_set_header X-Real-IP $remote_addr;
           proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
           proxy_set_header X-Forwarded-Proto $scheme;
           proxy_set_header X-Frame-Options SAMEORIGIN;
           proxy_buffers 256 16k;
           proxy_buffer_size 16k;
           client_body_timeout 60;
           send_timeout        300;
           lingering_timeout   5;
           proxy_connect_timeout   90;
           proxy_send_timeout      300;
           proxy_read_timeout      90s;
           proxy_pass http://backend;
   }

   location / {
           client_max_body_size 50M;
           proxy_set_header Connection "";
           proxy_set_header Host $http_host;
           proxy_set_header X-Real-IP $remote_addr;
           proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
           proxy_set_header X-Forwarded-Proto $scheme;
           proxy_set_header X-Frame-Options SAMEORIGIN;
           proxy_buffers 256 16k;
           proxy_buffer_size 16k;
           proxy_read_timeout 600s;
           proxy_cache mattermost_cache;
           proxy_cache_revalidate on;
           proxy_cache_min_uses 2;
           proxy_cache_use_stale timeout;
           proxy_cache_lock on;
           proxy_http_version 1.1;
           proxy_pass http://backend;
   }
}
`
	backends := ""
	for _, addr := range appInstanceAddrs {
		backends += "server " + addr + ";\n"
	}

	session.Stdin = strings.NewReader(strings.TrimSpace(fmt.Sprintf(config, backends)))
	if err := session.Run("cat | sudo tee /etc/nginx/sites-available/mattermost"); err != nil {
		return err
	}

	return nil
}

func uploadLimitsConfig(client *ssh.Client) error {
	limits := `
* soft nofile 65536
* hard nofile 65536
* soft nproc 8192
* hard nproc 8192
`

	sysctl := `
net.ipv4.ip_local_port_range="1024 65000"
net.ipv4.tcp_fin_timeout=30
`

	session, err := client.NewSession()
	if err != nil {
		return errors.Wrap(err, "unable to create ssh session")
	}
	session.Stdin = strings.NewReader(strings.TrimSpace(limits))
	if err := session.Run("cat | sudo tee /etc/security/limits.conf"); err != nil {
		return err
	}
	session.Close()

	session, err = client.NewSession()
	if err != nil {
		return errors.Wrap(err, "unable to create ssh session")
	}
	session.Stdin = strings.NewReader(strings.TrimSpace(sysctl))
	if err := session.Run("cat | sudo tee /etc/sysctl.conf"); err != nil {
		return err
	}
	session.Close()

	return nil
}

func reboot(clust ltops.Cluster, client *ssh.Client, logger logrus.FieldLogger) error {
	for _, cmd := range []string{
		"sudo shutdown -r now &",
	} {
		logger.Debug("+ " + cmd)
		if err := sshtools.RemoteCommand(client, cmd, ioutil.Discard); err != nil {
			return errors.Wrap(err, "error running command: "+cmd)
		}
	}

	connected := make(chan bool)
	cancel := make(chan bool)
	go func() {
		for {
			select {
			case <-time.After(5 * time.Second):
				logger.Debug("attempting to establish ssh session")

				if newClient, err := sshtools.SSHClient(
					clust.SSHKey(),
					strings.Split(client.RemoteAddr().String(), ":")[0],
				); err != nil {
					logger.Debug(err.Error())
				} else {
					newClient.Close()
					close(connected)
					return
				}
			case <-cancel:
				return
			}
		}
	}()

	select {
	case <-time.After(60 * time.Second):
		close(cancel)
		return errors.New("failed to reestablish ssh session after 60 seconds")
	case <-connected:
		logger.Debug("reestablished ssh session")
	}

	return nil
}
