package terraform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	ltops "github.com/mattermost/mattermost-load-test-ops"
	"github.com/mattermost/mattermost-load-test-ops/sshtools"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

func getFileOrURL(fileOrUrl string) ([]byte, error) {
	buffer := bytes.NewBuffer(nil)
	if strings.HasPrefix(fileOrUrl, "http") {
		response, err := http.Get(fileOrUrl)
		if err != nil {
			return nil, errors.Wrap(err, "Can't get file at URL: "+fileOrUrl)
		}
		defer response.Body.Close()

		io.Copy(buffer, response.Body)

		return buffer.Bytes(), nil
	} else {
		f, err := os.Open(fileOrUrl)
		if err != nil {
			return nil, errors.Wrap(err, "unable to open file "+fileOrUrl)
		}
		defer f.Close()

		io.Copy(buffer, f)

		return buffer.Bytes(), nil
	}
}

func (c *Cluster) DeployMattermost(mattermostDistLocation string, licenceFileLocation string) error {
	appInstanceAddrs, err := c.GetAppInstancesAddrs()
	if err != nil || len(appInstanceAddrs) <= 0 {
		return errors.Wrap(err, "Unable to get app instance addresses")
	}

	proxyInstanceAddrs, err := c.GetProxyInstancesAddrs()
	if err != nil || len(proxyInstanceAddrs) <= 0 {
		return errors.Wrap(err, "Unable to get app instance addresses")
	}

	mattermostDist, err := getFileOrURL(mattermostDistLocation)
	if err != nil {
		return err
	}

	licenseFile, err := getFileOrURL(licenceFileLocation)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(len(appInstanceAddrs) + len(proxyInstanceAddrs))

	failed := new(int32)

	doDeploy(&wg, failed, appInstanceAddrs, "app", func(instanceNum int, addr string, logger logrus.FieldLogger) error {
		return deployToAppInstance(bytes.NewReader(mattermostDist), bytes.NewReader(licenseFile), addr, c, logrus.WithField("instance", addr))
	})

	doDeploy(&wg, failed, proxyInstanceAddrs, "proxy", func(instanceNum int, addr string, logger logrus.FieldLogger) error {
		return deployToProxyInstance(addr, c, logrus.WithField("instance", addr))
	})

	wg.Wait()

	if *failed == 1 {
		return fmt.Errorf("failed to deploy to 1 instance")
	} else if *failed > 0 {
		return fmt.Errorf("failed to deploy to %v instances", *failed)
	} else {
		// This is here because the commands above do not wait for the servers to come back up after they restart them.
		//TODO: Actually wait for them instead of just sleeping
		logrus.Info("Deploy successful. Restarting servers...")
		time.Sleep(time.Second * 5)
		logrus.Info("Done")
	}
	return nil
}

func (c *Cluster) DeployLoadtests(loadtestsDistLocation string) error {
	loadtestInstanceAddrs, err := c.GetLoadtestInstancesAddrs()
	if err != nil || len(loadtestInstanceAddrs) <= 0 {
		return errors.Wrap(err, "Unable to get loadtest instance addresses")
	}

	loadtestsDist, err := getFileOrURL(loadtestsDistLocation)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(len(loadtestInstanceAddrs))

	failed := new(int32)

	doDeploy(&wg, failed, loadtestInstanceAddrs, "loadtest", func(instanceNum int, addr string, logger logrus.FieldLogger) error {
		return deployToLoadtestInstance(instanceNum, addr, bytes.NewReader(loadtestsDist), c, logrus.WithField("instance", addr))
	})

	wg.Wait()

	if *failed == 1 {
		return fmt.Errorf("failed to deploy to 1 instance")
	} else if *failed > 0 {
		return fmt.Errorf("failed to deploy to %v instances", *failed)
	}

	return nil
}

func doDeploy(wg *sync.WaitGroup, failed *int32, addresses []string, addressesName string, deployFunc func(instanceNum int, addr string, logger logrus.FieldLogger) error) error {
	for instanceNum, instanceAddr := range addresses {
		instanceAddr := instanceAddr
		instanceNum := instanceNum
		go func() {
			logrus.Infof("deploying to %v%v : %v...", addressesName, instanceNum, instanceAddr)
			if err := deployFunc(instanceNum, instanceAddr, logrus.WithField("instance", strconv.Itoa(instanceNum)+":"+instanceAddr)); err != nil {
				wrapped := errors.Wrap(err, "unable to deploy to "+addressesName+strconv.Itoa(instanceNum)+" : "+instanceAddr)
				logrus.Error(wrapped)
				atomic.AddInt32(failed, 1)
			} else {
				logrus.Infof("successfully deployed to %v%v : %v...", addressesName, instanceNum, instanceAddr)
			}
			wg.Done()
		}()
	}
	return nil
}

func deployToLoadtestInstance(instanceNum int, instanceAddr string, loadtestDistribution io.Reader, cluster ltops.Cluster, logger logrus.FieldLogger) error {
	client, err := sshtools.SSHClient(cluster.SSHKey(), instanceAddr)
	if err != nil {
		return errors.Wrap(err, "unable to connect to server via ssh")
	}
	defer client.Close()

	logger.Debug("uploading distribution...")
	remoteDistributionPath := "/home/ubuntu/mattermost-load-test.tar.gz"
	if err := sshtools.UploadReader(client, loadtestDistribution, remoteDistributionPath); err != nil {
		return errors.Wrap(err, "unable to upload loadtest distribution.")
	}

	remoteSSHKeyPath := "/home/ubuntu/key.pem"
	if err := sshtools.UploadBytes(client, cluster.SSHKey(), remoteSSHKeyPath); err != nil {
		return errors.Wrap(err, "unable to upload ssh key")
	}

	for _, cmd := range []string{
		"sudo apt-get update",
		"sudo apt-get install -y jq",
		"sudo rm -rf /home/ubuntu/mattermost-load-test",
		"tar -xvzf /home/ubuntu/mattermost-load-test.tar.gz",
		"sudo chmod 600 /home/ubuntu/key.pem",
	} {
		logger.Debug("+ " + cmd)
		if err := sshtools.RemoteCommand(client, cmd); err != nil {
			return errors.Wrap(err, "error running command: "+cmd)
		}
	}

	logger.Debug("uploading limits config...")
	if err := uploadLimitsConfig(client); err != nil {
		return errors.Wrap(err, "Unable to upload limits config")
	}

	proxyURLs, err := cluster.GetProxyInstancesAddrs()
	if err != nil || len(proxyURLs) < 1 {
		return errors.Wrap(err, "Couldn't get app instance addresses.")
	}

	appURLs, err := cluster.GetAppInstancesAddrs()
	if err != nil || len(appURLs) < 1 {
		return errors.Wrap(err, "Couldn't get app instance addresses.")
	}

	appURL, err := url.Parse(appURLs[0])
	if err != nil {
		return errors.Wrap(err, "Couldn't parse app url.")
	}

	siteURL, err := url.Parse(proxyURLs[instanceNum])
	if err != nil {
		return errors.Wrap(err, "Can't parse site URL")
	}

	serverURL := *siteURL
	serverURL.Scheme = "http"

	websocketURL := *siteURL
	websocketURL.Scheme = "ws"

	pprofURL := *appURL
	pprofURL.Host = pprofURL.Host + ":8067"
	pprofURL.Path = "/debug/pprof"

	for k, v := range map[string]interface{}{
		".ConnectionConfiguration.ServerURL":            serverURL.String(),
		".ConnectionConfiguration.WebsocketURL":         websocketURL.String(),
		".ConnectionConfiguration.PProfURL":             pprofURL.String(),
		".ConnectionConfiguration.DataSource":           cluster.DBConnectionString(),
		".ConnectionConfiguration.LocalCommands":        false,
		".ConnectionConfiguration.SSHHostnamePort":      appURL.String() + ":22",
		".ConnectionConfiguration.SSHUsername":          "ubuntu",
		".ConnectionConfiguration.SSHKey":               remoteSSHKeyPath,
		".ConnectionConfiguration.MattermostInstallDir": "/opt/mattermost",
		".ConnectionConfiguration.WaitForServerStart":   false,
	} {
		logger.Debug("updating config: " + k)
		jsonValue, err := json.Marshal(v)
		if err != nil {
			return errors.Wrap(err, "invalid config value for key: "+k)
		}
		if err := sshtools.RemoteCommand(client, fmt.Sprintf(`jq '%s = %s' /home/ubuntu/mattermost-load-test/loadtestconfig.json > /tmp/ltconfig.json && mv /tmp/ltconfig.json /home/ubuntu/mattermost-load-test/loadtestconfig.json`, k, string(jsonValue))); err != nil {
			return errors.Wrap(err, "error updating config: "+k)
		}
	}

	for _, cmd := range []string{
		"sudo shutdown -r now &",
	} {
		logger.Debug("+ " + cmd)
		if err := sshtools.RemoteCommand(client, cmd); err != nil {
			return errors.Wrap(err, "error running command: "+cmd)
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
		if err := sshtools.RemoteCommand(client, cmd); err != nil {
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
		"sudo shutdown -r now &",
	} {
		logger.Debug("+ " + cmd)
		if err := sshtools.RemoteCommand(client, cmd); err != nil {
			return errors.Wrap(err, "error running command: "+cmd)
		}
	}

	return nil
}

func deployToAppInstance(mattermostDistribution, license io.Reader, instanceAddr string, clust *Cluster, logger logrus.FieldLogger) error {
	client, err := sshtools.SSHClient(clust.SSHKey(), instanceAddr)
	if err != nil {
		return errors.Wrap(err, "unable to connect to server via ssh")
	}
	defer client.Close()

	logger.Debug("uploading distribution...")
	remoteDistributionPath := "/tmp/mattermost.tar.gz"
	if err := sshtools.UploadReader(client, mattermostDistribution, remoteDistributionPath); err != nil {
		return errors.Wrap(err, "unable to upload distribution.")
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
		if err := sshtools.RemoteCommand(client, cmd); err != nil {
			return errors.Wrap(err, "error running command: "+cmd)
		}
	}

	logger.Debug("uploading license file...")
	remoteLicenseFilePath := "/opt/mattermost/config/mattermost.mattermost-license"
	if err := sshtools.UploadReader(client, license, remoteLicenseFilePath); err != nil {
		return errors.Wrap(err, "unable to upload license file")
	}

	logger.Debug("uploading limits config...")
	if err := uploadLimitsConfig(client); err != nil {
		return errors.Wrap(err, "Unable to upload limits config")
	}

	outputParams, err := clust.Env.getOuptutParams()
	if err != nil {
		return errors.Wrap(err, "Can't get output parameters")
	}

	s3AccessKeyId := outputParams.S3AccessKeyId.Value
	s3AccessKeySecret := outputParams.S3AccessKeySecret.Value
	s3Bucket := outputParams.S3bucket.Value
	s3Region := outputParams.S3bucketRegion.Value

	for k, v := range map[string]interface{}{
		".ServiceSettings.ListenAddress":               ":80",
		".ServiceSettings.LicenseFileLocation":         remoteLicenseFilePath,
		".ServiceSettings.SiteURL":                     clust.SiteURL(),
		".ServiceSettings.EnableAPIv3":                 true,
		".SqlSettings.DriverName":                      "mysql",
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
	} {
		logger.Debug("updating config: " + k)
		jsonValue, err := json.Marshal(v)
		if err != nil {
			return errors.Wrap(err, "invalid config value for key: "+k)
		}
		if err := sshtools.RemoteCommand(client, fmt.Sprintf(`jq '%s = %s' /opt/mattermost/config/config.json > /tmp/mmcfg.json && mv /tmp/mmcfg.json /opt/mattermost/config/config.json`, k, string(jsonValue))); err != nil {
			return errors.Wrap(err, "error updating config: "+k)
		}
	}

	for _, cmd := range []string{
		"sudo setcap cap_net_bind_service=+ep /opt/mattermost/bin/platform",
		"sudo systemctl daemon-reload",
		"sudo systemctl restart mattermost.service",
		"sudo systemctl enable mattermost.service",
		"sudo shutdown -r now &",
	} {
		logger.Debug("+ " + cmd)
		if err := sshtools.RemoteCommand(client, cmd); err != nil {
			return errors.Wrap(err, "error running command: "+cmd)
		}
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
