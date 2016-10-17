..  _prod-ubuntu:

===============================================
Mattermost Load Test Install Guide (WIP) 
===============================================

This install guide sets up Mattermost Load Test to replicate a set of consistent results.

1. `Confirm Benchmark Hardware`_
2. `Production Install on Ubuntu 14.04 LTS with MySQL on Amazon RDS`_
3. `Installing Mattermost Load Test`_
4. `Varying Mattermost Load Test`_

Confirm Benchmark Hardware
============================================

Confirm you will be able to provision the following hardware to replicate the results of Mattermost Load Tests: 

- One (1) Load Test Server - Amazon EC2 instance of size `m4.xlarge`
- One (1) Mattermost Application Server - Amazon EC2 instance of size `m4.xlarge`
- One (1) Mattermost Database Server - Amazon MySQL RDS instance of size `m4.xlarge`
- One (1) Mattermost Proxy - Amazon EC2 instance of size `m4.xlarge`

See `Amazon EC2 Instance Types <https://aws.amazon.com/ec2/instance-types/>`_ for details on hardware used. 

Production Install on Ubuntu 14.04 LTS with MySQL on Amazon RDS
========================================================================

Install Mattermost in production mode on one, two or three machines, using the following steps: 

- `Install Ubuntu Server (x64) 14.04 LTS <#install-ubuntu-server-x64-1404-lts>`_
- `Set up Database Server <#set-up-database-server>`_
- `Set up Mattermost Server <#set-up-mattermost-server>`_
- `Set up NGINX Server <#set-up-nginx-server>`_
- `Test setup and configure Mattermost Server <#test-setup-and-configure-mattermost-server>`_


Install Ubuntu Server (x64) 14.04 LTS
-------------------------------------

1. Set up 3 machines with Ubuntu 14.04 with 2GB of RAM or more. The
   servers will be used for the Proxy, Mattermost (must be
   x64), and Database.

   -  **Optional:** You can also use a **1 machine setup** (Proxy, Mattermost and Database on one machine) or a **2 machine setup** (Proxy and Mattermost on one machine, Database on another) depending on your data center standards. 

2. Make sure the system is up to date with the most recent security
   patches.

   -  ``sudo apt-get update``
   -  ``sudo apt-get upgrade``

Set up Database Server
----------------------

The Mattermost Load Test is benchmarked using MySQL on a pre-optimized Amazon RDS instance. 

To set up an Amazon Relational Database Server with MySQL (use default MySQL 5.6.27), from the EC2 main menu in Amazon Web Services click: 

   - Launch Database using RDS: 
   - Get Started Now > Select MySQL > Select "Production" then Next Step 
   - Select DB Instance Class `m4.xlarge` > Select 100GB to enable SSD storage 
   - Launch DB instance 

Final steps: 

- Confirm `max_connecton` is not less than setting in Mattermost config

Set up Mattermost Server
------------------------

1. For the purposes of this guide we will assume this server has an IP
   address of ``10.10.10.2``
2. For the sake of making this guide simple we located the files at
   ``/home/ubuntu/mattermost``. In the future we will give guidance for
   storing under ``/opt``.
3. We have also elected to run the Mattermost Server as the ``ubuntu``
   account for simplicity. We recommend setting up and running the
   service under a ``mattermost`` user account with limited permissions.
4. Download `any version of the Mattermost Server <https://docs.mattermost.com/administration/upgrade.html#version-archive>`_ by typing:

   -  ``wget https://releases.mattermost.com/X.X.X/mattermost-X.X.X-linux-amd64.tar.gz``
   -  Where ``vX.X.X`` is typically the latest Mattermost release version, which is currently ``v3.3.0``. 
   
5. Unzip the Mattermost Server by typing:

   -  ``tar -xvzf mattermost-X.X.X-linux-amd64.tar.gz``

6. Create the storage directory for files. We assume you will have
   attached a large drive for storage of images and files. For this
   setup we will assume the directory is located at
   ``/mattermost/data``.

   -  Create the directory by typing:
   -  ``sudo mkdir -p /mattermost/data``
   -  Set the ubuntu account as the directory owner by typing:
   -  ``sudo chown -R ubuntu /mattermost``

7. Configure Mattermost Server by editing the config.json file at
   ``/home/ubuntu/mattermost/config``

   -  ``cd ~/mattermost/config``
   -  Edit the file by typing:
   -  ``vi config.json``
   -  replace ``DriverName": "mysql"`` with ``DriverName": "postgres"``
   -  replace
      ``"DataSource": "mmuser:mostest@tcp(dockerhost:3306)/mattermost_test?charset=utf8mb4,utf8"``
      with
      ``"DataSource": "postgres://mmuser:mmuser_password@10.10.10.1:5432/mattermost?sslmode=disable&connect_timeout=10"``
   -  Optionally you may continue to edit configuration settings in
      ``config.json`` or use the System Console described in a later
      section to finish the configuration.

8. Test the Mattermost Server

   -  ``cd ~/mattermost/bin``
   -  Run the Mattermost Server by typing:
   -  ``./platform``
   -  You should see a console log like ``Server is listening on :8065``
      letting you know the service is running.
   -  Stop the server for now by typing ``ctrl-c``

9. Setup Mattermost to use the Upstart daemon which handles supervision
   of the Mattermost process.

   -  ``sudo touch /etc/init/mattermost.conf``
   -  ``sudo vi /etc/init/mattermost.conf``
   -  Copy the following lines into ``/etc/init/mattermost.conf``

      ::

          start on runlevel [2345]
          stop on runlevel [016]
          respawn
          limit nofile 50000 50000
          chdir /home/ubuntu/mattermost
          setuid ubuntu
          exec bin/platform

   -  You can manage the process by typing:
   -  ``sudo start mattermost``
   -  Verify the service is running by typing:
   -  ``curl http://10.10.10.2:8065``
   -  You should see a page titles *Mattermost - Signup*
   -  You can also stop the process by running the command
      ``sudo stop mattermost``, but we will skip this step for now.

Set up NGINX Server
-------------------

1. For the purposes of this guide we will assume this server has an IP
   address of ``10.10.10.3``

2. We use NGINX for proxying request to the Mattermost Server. The main
   benefits are:

   -  SSL termination
   -  http to https redirect
   -  Port mapping ``:80`` to ``:8065``
   -  Standard request logs

3. Install the latest version of NGINX on Ubuntu from `https://www.nginx.com/resources/wiki/start/topics/tutorials/install/ <https://www.nginx.com/resources/wiki/start/topics/tutorials/install/>`_

   -  NGINX version 1.10.1 or 1.11.4 are recommended to replicate results
   -  Note: ``sudo apt-get install nginx`` wasn't used, because it installs NGINX version 1.4.

4. Verify NGINX is running

   -  ``curl http://10.10.10.3``
   -  You should see a *Welcome to NGINX!* page

5. You can manage NGINX with the following commands

   -  ``sudo service nginx stop``
   -  ``sudo service nginx start``
   -  ``sudo service nginx restart``

6. Map a FQDN (fully qualified domain name) like
   ``mattermost.example.com`` to point to the NGINX server.

7. Make sure resource limits are set correctly - we want `open files` to be at least 50,000

   - You can verify the NGINX process has the correct amounts by running

    ::

        ps -aux | grep nginx
        cat /proc/<worker process ID>/limits

   - You can modify the resource limits by

   - ``sudo vi /etc/nginx/nginx.conf``
   - Add the line ``worker_rlimit_nofile 50000;``
   - ``sudo vi /etc/security/limits.conf``
   - Add the lines 
      
        ::
        
            root hard nofile 500000
            www-data hard nofile 500000   

8. Modify nginx.conf settings

   - ``sudo vi /etc/nginx/nginx.conf``
   - Modify ``worker_connections`` and change to ``20000``
   - Modify ``worker_processes`` and change to ``4`` or the number of cores on your box
   - Modify ``keepalive_timeout`` and change to ``20``
   - Your nginx.conf settings should look something like

        ::
        
            user  nginx;
            worker_processes  4;
            worker_rlimit_nofile 50000;

            error_log  /var/log/nginx/error.log warn;
            pid        /var/run/nginx.pid;


            events {
                worker_connections  20000;
            }


            http {
                include       /etc/nginx/mime.types;
                default_type  application/octet-stream;

                log_format  main  '$remote_addr - $remote_user [$time_local] "$request" '
                                  '$status $body_bytes_sent "$http_referer" '
                                  '"$http_user_agent" "$http_x_forwarded_for"';

                access_log  /var/log/nginx/access.log  main;

                sendfile        on;
                #tcp_nopush     on;

                keepalive_timeout  20;

                #gzip  on;

                include /etc/nginx/conf.d/*.conf;
            }

9. Configure NGINX to proxy connections from the internet to the
   Mattermost Server

   -  Create a configuration for Mattermost
   -  ``sudo touch /etc/nginx/sites-available/mattermost``
   -  Update the Mattermost NGINX config
   -  ``sudo vi /etc/nginx/conf.d/default.conf``
   -  Below is a sample configuration with the minimum settings required
      to configure Mattermost

    ::

        server {
            server_name mattermost.example.com;

        location / {
            sendfile on;
            proxy_connect_timeout 300;
            proxy_send_timeout 300;
            proxy_read_timeout 300;
            send_timeout 300;
            proxy_buffers 256 16k;
            proxy_buffer_size 32k;
            client_max_body_size 50M;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
            proxy_set_header Host $http_host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
            proxy_set_header X-Frame-Options SAMEORIGIN;
            proxy_pass http://10.10.10.2:8065;
            proxy_http_version 1.1;
            proxy_set_header Connection "";
         }
      }

   - Restart NGINX by typing:
   - `` sudo service nginx restart```
   - Verify you can see Mattermost thru the proxy by typing:
   - `` curl http://localhost``
   - You should see a page titles *Mattermost - Signup*

Set up NGINX with SSL (Recommended)
-----------------------------------

1. You can use a free and an open certificate security like let's
   encrypt, this is how to proceed

-  ``sudo apt-get install git``
-  ``git clone https://github.com/letsencrypt/letsencrypt``
-  ``cd letsencrypt``

2. Be sure that the port 80 is not used by stopping NGINX

-  ``sudo service nginx stop``
-  ``netstat -na | grep ':80.*LISTEN'``
-  ``./letsencrypt-auto certonly --standalone``

3. This command will download packages and run the instance, after that
   you will have to give your domain name
4. You can find your certificate in ``/etc/letsencrypt/live``
5. Modify the file at ``/etc/nginx/sites-available/mattermost`` and add
   the following lines:

  ::

      server {
         listen         80;
         server_name    mattermost.example.com;
         return         301 https://$server_name$request_uri;
      }

      server {
         listen 443 ssl;
         server_name mattermost.example.com;

         ssl on;
         ssl_certificate /etc/letsencrypt/live/yourdomainname/fullchain.pem;
         ssl_certificate_key /etc/letsencrypt/live/yourdomainname/privkey.pem;
         ssl_session_timeout 5m;
         ssl_protocols TLSv1 TLSv1.1 TLSv1.2;
         ssl_ciphers 'EECDH+AESGCM:EDH+AESGCM:AES256+EECDH:AES256+EDH';
         ssl_prefer_server_ciphers on;
         ssl_session_cache shared:SSL:10m;

         location / {
            gzip off;
            proxy_set_header X-Forwarded-Ssl on;
            client_max_body_size 50M;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
            proxy_set_header Host $http_host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
            proxy_set_header X-Frame-Options SAMEORIGIN;
            proxy_pass http://10.10.10.2:8065;
         }
      }

6. Be sure to restart NGINX
  * ``\ sudo service nginx start``
7. Add the following line to cron so the cert will renew every month
  * ``crontab -e``
  * ``@monthly /home/ubuntu/letsencrypt/letsencrypt-auto certonly --reinstall -d yourdomainname && sudo service nginx reload``
8. Check that your SSL certificate is set up correctly
  * Test the SSL certificate by visiting a site such as `https://www.ssllabs.com/ssltest/index.html <https://www.ssllabs.com/ssltest/index.html>`_
  * If there’s an error about the missing chain or certificate path, there is likely an intermediate certificate missing that needs to be included

Test setup and configure Mattermost Server
------------------------------------------

1. Navigate to ``https://mattermost.example.com`` and create a team and
   user.
2. The first user in the system is automatically granted the
   ``system_admin`` role, which gives you access to the System Console.
3. From the ``town-square`` channel click the dropdown and choose the
   ``System Console`` option
4.  Update **Notification** > **Email** settings to setup an SMTP email service. The example below assumes AmazonSES.

   -  Set *Send Email Notifications* to ``true``
   -  Set *Require Email Verification* to ``true``
   -  Set *Feedback Name* to ``No-Reply``
   -  Set *Feedback Email* to ``mattermost@example.com``
   -  Set *SMTP Username* to ``[YOUR_SMTP_USERNAME]``
   -  Set *SMTP Password* to ``[YOUR_SMTP_PASSWORD]``
   -  Set *SMTP Server* to ``email-smtp.us-east-1.amazonaws.com``
   -  Set *SMTP Port* to ``465``
   -  Set *Connection Security* to ``TLS``
   -  Save the Settings

5. Update **File** > **Storage** settings:

   -  Change *Local Directory Location* from ``./data/`` to
      ``/mattermost/data``

6. Update **General** > **Logging** settings:

   -  Set *Log to The Console* to ``false``

7. Update **Advanced** > **Rate Limiting** settings:

   -  Set *Vary By Remote Address* to ``false``
   -  Set *Vary By HTTP Header* to ``X-Real-IP``

8. Update **Advanced** > **Database** settings:

   -  Set *Maximum Idle Connections* to ``20``
   -  Set *Maximum Open Connections* to ``300``

9. Feel free to modify other settings.
10. Restart the Mattermost Service by typing:

    -  ``sudo restart mattermost``

Installing Mattermost Load Test
============================================

1. Download Mattermost Load Test::

      git clone https://github.com/mattermost/mattermost-load-test.git

2. Confirm your environment is clear (deletes ``cache.db`` if it exists)::

      make clean

3. Build the Mattermost Load Test:: 

      make install

4. Run the default load test script (this takes about 40 minutes):: 

      ./bin/run_example.sh

The default load test script requires about 40 minutes to create all the users. This is because Mattermost users a bcrypt function to generate user passwords that is designed to be computationally intensive. 

Load Test Setup
------------------------------------------

1. Change the time the kernel recycles closed connections from 60 seconds to 30 seconds. The change will need to be done on the NGINX server, Mattermost server and load tester server.

   - ``sudo sysctl -w net.ipv4.tcp_fin_timeout=30``
   - You can verify it's set correctly with
   - ``cat /proc/sys/net/ipv4/tcp_fin_timeout``

2. Change the number of usable ports for nginx and mattermost. The change will need to be done on the NGINX server, Mattermost server and load tester server.

   - ``sudo sysctl -w net.ipv4.ip_local_port_range="1024 65000"``
   - You can verify it's set correctly with
   - ``cat /proc/sys/net/ipv4/ip_local_port_range``

3. You should also add the changes above to ``/etc/sysctl.conf`` to be persistent across reboots.

Verifying Mattermost Load Test
============================================

1. Add your System Admin account to the default team (where user@example.com is the email of your System Admin account)::

      platform -join_team -email="user@example.com" -team_name="team"

You can now use your System Administrator account to login to Mattermost and view any of the channels where the simulated users are posting messages. 


Estimating Performance 
============================================

TBD

Varying Mattermost Load Test
============================================

Total Potential Users 
--------------------------------------------------

Setting: ``THREADCOUNT="15000"``

- Total number of simulated users 

Inactive Users
--------------------------------------------------

Setting: ``THREADOFFSET="0"``

- Total number of unused users 

Setup Time 
--------------------------------------------------

Setting: ``RAMPSEC="1000"``

- Number of seconds to log in all users. This setting is used to prevent errors from the CPU from being overloaded by the bcrypt function used to create new users. It can be adjusted based on the number of users and CPU utilization observed during initialization. 

Max Wait 
--------------------------------------------------

Setting: ``MESSAGEBREAK="240"``

- Maximum number of seconds one user randomly waits before sending the next message. The average wait time is one half of the maximum wait time. 

Reply Percentage 
--------------------------------------------------

Setting: ``REPLYPERCENT="2"``

- Percentage of users who reply when they receive a message.

Tips and Useful Commands
============================================


Check number of Connections
--------------------------------------------------

To see the number of connections to the mattermost server you can run cmds like

```bash
sudo netstat -an | grep :8065 | wc -l
```
or
```bash
ss | grep ESTA | grep 8065
```

Look for slow SQL queries in MySQL
--------------------------------------------------

```sql
SET GLOBAL log_output = 'TABLE';
SET GLOBAL slow_query_log = 'ON'; 
SET GLOBAL long_query_time = 1;
SET GLOBAL log_queries_not_using_indexes = 'OFF';

show global variables WHERE Variable_name IN ('log_output', 'slow_query_log', 'long_query_time', 'long_query_time', 'log_queries_not_using_indexes');
```

```sql
SELECT *, CAST(sql_text AS CHAR(10000) CHARACTER SET utf8) AS Query FROM mysql.slow_log ORDER BY start_time DESC LIMIT 100 
```

```sql
TRUNCATE mysql.slow_log; 
```

Generate Profiling Data
--------------------------------------------------

Start the server with 

```bash
./bin/platform -httpprofiler
```

Look at different profiles with

```bash
go tool pprof platform http://localhost:8065/debug/pprof/profile
go tool pprof platform http://localhost:8065/debug/pprof/heap
go tool pprof platform http://localhost:8065/debug/pprof/block
go tool pprof platform http://localhost:8065/debug/pprof/goroutine

```

