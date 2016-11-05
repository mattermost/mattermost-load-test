# Mattermost Load Test

Mattermost Load Test provides infrastructure for simulating real-world usage of the [Mattermost enterprise messaging server](https://about.mattermost.com/) at scale.

## Pre-requisites 

### Git LFS

This repository uses Git LFS: https://git-lfs.github.com/

To retrieve the sample databases, you will need to install it. 

## Running the tests

### Setting up your Load Test environment

1. Setup your Load Test environment using the following instructions: https://github.com/mattermost/mattermost-load-test/blob/master/install.rst

2. Setup your server that will be running the loadtests. It should be similar in power to the application server. Install the `loadtest` command on it. You can use `make package` to get a `tar.gz` under the `dist` directory.

3. Perform these tweaks to maximize performance:

Depending on your distribution and version, either modify your `upstart` config file or your `systemd` config file.

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


Modify your `/etc/security/limits.conf` on all your machines. 

Change your process limit to 8192 and your max number of open files to 65536.

You can do this by adding the lines:

```
* soft nofile 65536
* hard nofile 65536
* soft nproc 8192
* hard nproc 8192
```


Modify your `/etc/sysctl.conf` on all your machines.

Add the lines:
```
net.ipv4.ip_local_port_range="1024 65000"
net.ipv4.tcp_fin_timeout=30
```

You will need to restart the machines to let these changes take effect.


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

In addtion modify your `/etc/nginx/nginx.conf`:

  - Change or add `worker_connections` to `120000`
  - Change or add `worker_processes` to the number of cores on the nginx machine (eg. `4`)
  - Change or add `keepalive_timeout` to `20`
  - Change or add `worker_rlimit_nofile` to `65536`

TODO: Mention which specific tweaks to perform to nginx. 

Modify your Mattermost configuration file `config/config.json`:

  - Change `MaxIdleConns` to `20`
  - Change `MaxOpenConns` to `300`

### Setting up your Database

To setup your database, you can use the pre-created databases under `sample-dbs`. For now the recommended database is `loadtest1-20000-shift.sql` To load them, from the command line use `mysql -u username < file.sql`

After you have loaded the database, you can copy the corresponding configuration and state to your loadtest server. For `loadtest1-20000-shift.sql` those are `loadtest1-20000-shift-state.json` and `loadtest-20000-shift-loadtestconfig.json`. You will need to rename the config to `loadtestconfig.json` so it is recognized by the loadtests.

### Setup your loadtestconfig.json

You will need to modify at least two entries in your `loadtestconfig.json`.

1. `ServerURL` needs to be set to the address of your proxy and `WebsocketURL` should be set as well.
2. If your using a sample DB the `AdminEmail` and `AdminPassword` should be set correctly already. 

If you want to know more about other configuration options, see [loadtestconfig.json documentation](loadtestconfig.md)

### Running the loadtests. 

Now you can run the loadtests from your loadtest machine by using the command `cat state.json | loadtest listenandpost`. This will run the loadtest. 

A summary of activity will be output to the console so you can monitor the test. More detailed logging is performed in a status.log file output to the same directory the tests where run from. You can watch it by opening another terminal and running `tail -f status.log`. 

### Interacting with the loaded environment

The system admin user you used to setup the loadtests is now joined to every team and channel on the system. This might not be the best way to interact with your loaded environment. You can run the following commands from your mattermost installation directory to create a second system admin user:

    - `./bin/platform -create_user -team_name="name" -email="user@example.com" -password="mypassword" -username="user"`
    - `./bin/platform -assign_role -email="user@example.com" -role="system_admin"`


## Configuration Documentation

Please see [loadtestconfig.json documentation](loadtestconfig.md).
