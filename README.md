# Mattermost Load Test

Mattermost Load Test provides infrastructure for simulating real-world usage of the Mattermost Enterprise Edition E20 at scale. 

In setting up this system you will: 

- Deploy Mattermost in a production configuration, potentially in high availability mode
- Optimize the performance of your Mattermost deployment 
- Deploy a Mattermost Load Test server to apply simulated load to your production deployment 
- Log in to your Mattermost deployment to observe it under load

If you have questions about configuration, please contact your Account Manager. An overview of support available to E20 customers is available at https://about.mattermost.com/support/

## Pre-requisites 

SOFTWARE

- **Software required for Mattermost production deployment** - see [Software and Hardware Requirements Guide](https://docs.mattermost.com/install/requirements.html) for software requirements 
- **Git Large File Storage (Git LFS)** - to retrieve the sample databases used in this repository, you need to install Git LFS from https://git-lfs.github.com/

HARDWARE

- **Hardware required for Mattermost production deployment** - see [Software and Hardware Requirements Guide](https://docs.mattermost.com/install/requirements.html) for sizing hardware based on projected needs
- **Load Test Server** - to run Load Tests with hardware similar to Mattermost application server in production setup

## Running the tests

To run the Load Test simulation, complete the following: 

### 1) Set up your Load Test environment 

Follow setup instructions at: https://github.com/mattermost/mattermost-load-test/blob/master/install.rst

### 2) Set up your server that will be running the loadtests

The hardward specifications of the server running Load Test should be similar to the hardware of your application server. 

Install the `loadtest` command on the Load Test server. You can use `make package` to get a `tar.gz` under the `dist` directory.

### 3) Perform the following optimizations to maximize performance

#### a) Update either `upstart` or `systemd`

Depending on your distribution and version, either modify your `upstart` config file or your `systemd` config file: 

For upstart at `/etc/init/mattermost.conf`, add the line `limit nofile 50000 50000`:

```
start on runlevel [2345]
stop on runlevel [016]
respawn
limit nofile 50000 50000
chdir /home/ubuntu/mattermost
setuid ubuntu
exec bin/platform
```

For systemd at `/lib/systemd/system/mattermost.service` under `[Service]` add the line `LimitNOFILE=50000`:

```
Unit]
Description=Mattermost Server
After=network.target

[Service]
Type=simple
ExecStart=/home/ubuntu/mattermost/bin/platform
Restart=always
LimitNOFILE=50000
RestartSec=5
WorkingDirectory=/home/ubuntu/mattermost
User=ubuntu
Group=ubuntu

[Install]
WantedBy=multi-user.target
```
#### b) Modify `/etc/security/limits.conf`

Modify your `/etc/security/limits.conf` on all your machines. 

Change your process limit to 8192 and your max number of open files to 65536.

You can do this by adding the lines:

```
* soft nofile 65536
* hard nofile 65536
* soft nproc 8192
* hard nproc 8192
```

#### b) Modify `/etc/sysctl.conf`

Modify your `/etc/sysctl.conf` on all your machines.

Add the lines:
```
net.ipv4.ip_local_port_range="1024 65000"
net.ipv4.tcp_fin_timeout=30
```

You will need to restart the machines to let the changes in a) and b) take effect.

#### c) Modify NGINX configuration 

Modify your NGINX configuration to be:

```
upstream backend {
        server <HOSTNAME_OF_YOUR_APP_SERVER>;
}

server {
    listen 80;
    server_name    <HOSTNAME_OF_YOUR_PROXY>;

   location /api/v3/users/websocket {
      proxy_set_header Upgrade $http_upgrade;
      proxy_set_header Connection "upgrade";
      #proxy_set_header X-Forwarded-Ssl on;
      client_max_body_size 50M;
      proxy_set_header Host $http_host;
      proxy_set_header X-Real-IP $remote_addr;
      proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
      proxy_set_header X-Forwarded-Proto $scheme;
      proxy_set_header X-Frame-Options SAMEORIGIN;
      proxy_buffers 256 16k;
      proxy_buffer_size 16k;
      proxy_read_timeout 600s;
      proxy_pass http://backend;
   }

   location / {
      proxy_set_header X-Forwarded-Ssl on;
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
      proxy_pass http://backend;
    }
}

```

In addition modify your `/etc/nginx/nginx.conf`:

  - Change or add `worker_connections` to `120000`
  - Change or add `worker_processes` to the number of cores on the nginx machine (eg. `4`)
  - Change or add `keepalive_timeout` to `20`
  - Change or add `worker_rlimit_nofile` to `65536`

TODO: Mention which specific tweaks to perform on nginx. 

#### d) Modify Mattermost configuration 

Modify your Mattermost configuration file `config/config.json`:

  - Change `MaxIdleConns` to `20`
  - Change `MaxOpenConns` to `300`

### Setting up your Database

To set up your database, you can use the pre-created databases under `sample-dbs`. For now the recommended database is `loadtest1-20000-shift.sql` To load them, from the command line use `mysql -u username < file.sql`

After you have loaded the database, you can copy the corresponding configuration and state to your loadtest server. For `loadtest1-20000-shift.sql` those are `loadtest1-20000-shift-state.json` and `loadtest-20000-shift-loadtestconfig.json`. You will need to rename the config to `loadtestconfig.json` so it is recognized by the loadtests.

### Set up your `loadtestconfig.json`

You will need to modify at least two entries in your `loadtestconfig.json`.

1. `ServerURL` needs to be set to the address of your proxy and `WebsocketURL` should be set as well.
2. If your using a sample DB the `AdminEmail` and `AdminPassword` should be set correctly already. 

If you want to know more about other configuration options, see [Load Test Configuration](loadtestconfig.md) documentation. 

### Running the Load Tests

Now you can run the simulator from your Load Test machine by using the command `cat state.json | loadtest listenandpost`. 

A summary of activity will be output to the console so you can monitor the test. More detailed logging is performed in a `status.log` file output to the same directory the tests where run from. 

You can watch it by opening another terminal and running `tail -f status.log`. 

### Interacting with the Mattermost deployment under load 

You can create users to interact with the Mattermost deployment while under load by using the following Command Line Interface commands: 

    - `./bin/platform -create_user -team_name="name" -email="user@example.com" -password="mypassword" -username="user"`
    - `./bin/platform -assign_role -email="user@example.com" -role="system_admin"`

You can also use the System Administrator account set up in the Load Tests to interact with the system, however the account will be joined to every team and channel in the deployment, and may be awkward to use. 

## Configuration Documentation

Please see [Load Test Configuration](loadtestconfig.md) documentation. 

## Compliling for non master branch Mattermost

1. Edit the `glide.yaml` file under `github.com/mattermost/platform` change `version: master` to the branch you want to build against. For a release the branch is called `release-x-x`, eg `release-3.5`
2. run `make clean`
3. run `make package`
