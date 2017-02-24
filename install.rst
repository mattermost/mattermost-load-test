..  _prod-ubuntu:

===============================================
Mattermost Load Test Install Guide
===============================================

This install guide sets up Mattermost Load Test to replicate a set of consistent results.

Confirm Benchmark Hardware
============================================

Confirm you will be able to provision the following hardware to replicate the results of Mattermost Load Tests: 

- One (1) Load Test Server - Amazon EC2 instance of size `m4.xlarge`
- One (1) Mattermost Application Server - Amazon EC2 instance of size `m4.xlarge`
- One (1) Mattermost Database Server - Amazon MySQL RDS instance of size `m4.xlarge`
- One (1) Mattermost Proxy - Amazon EC2 instance of size `m4.xlarge`

See `Amazon EC2 Instance Types <https://aws.amazon.com/ec2/instance-types/>`_ for details on hardware used. 

You shoud run ``sudo apt-get update`` and ``sudo apt-get upgrade`` on each machine to get the most recent secrurity patches.

Set up Database Server
----------------------

The Mattermost Load Test is benchmarked using MySQL on a Amazon RDS instance. 

To set up an Amazon RDS with MySQL,from the EC2 main menu in Amazon Web Services click: 

   - Launch Database using RDS: 
   - Get Started Now > Select MySQL > Select "Production" then Next Step 
   - Select DB Instance Class `m4.xlarge` > Select 100GB to enable SSD storage 
   - Launch DB instance
   - Remember to setup security groups or your server won't be able to access your DB

Set up Mattermost Server
------------------------

Follow the "Setup Mattermost Server" instructions in our `Production Ubuntu Install Guide <https://docs.mattermost.com/install/prod-ubuntu.html#set-up-mattermost-server>`_ to setup your mattermost server.

Set up NGINX Server
-------------------

Follow the "Setup NGINX Server" or "Setup NGINX with SSL" instructions in our `Production Ubuntu Install Guide <https://docs.mattermost.com/install/prod-ubuntu.html#set-up-nginx-server>`_ to setup your NGINX server.

Additional Mattermost Configuration
-------------------------------------

Follow the "Additional Mattermost Configuration" instructions which can be done from the graphical system console. `Addtional Mattermost Configuration <https://docs.mattermost.com/install/prod-ubuntu.html#test-setup-and-configure-mattermost-server>`_

Notes on Running Simulations
-------------------------------------

**1) Create a new account to evaluate performance, DO NOT use the default administrator account**

The default administrator account was used to populate the database and is joined to every channel, which is not a realistic use case. If you use the default administrator account during a load test you will see an unrealistic slow down of your browser. 

**2) Load Test simulates actual users**

If you stop the load test, the load test users will appear disconnected from the Mattermost server, which will mark the users as "away" for 5 minutes after they are disconnected, before showing them as offline. 

If you stop the load test server and start it again either wait 5-10 minutes between tests, or reset the Mattermost server to clear the "away" states of the users. 


Tips and Useful Performance Testing Commands
===============================================

Check number of Connections
--------------------------------------------------

To see the number of connections to the mattermost server you can run commands like:

   sudo netstat -an | grep :8065 | wc -l

or:

   ss | grep ESTA | grep 8065


Verify the resource limits are set correctly
---------------------------------------------

- You can verify the NGINX process has the correct amounts by running:

    ps -aux | grep nginx
    cat /proc/<worker process ID>/limits


Look for slow SQL queries in MySQL
--------------------------------------------------

Considering using the following: 

   SET GLOBAL log_output = 'TABLE';
   SET GLOBAL slow_query_log = 'ON'; 
   SET GLOBAL long_query_time = 1;
   SET GLOBAL log_queries_not_using_indexes = 'OFF';

   show global variables WHERE Variable_name IN ('log_output', 'slow_query_log', 'long_query_time', 'long_query_time', 'log_queries_not_using_indexes');

   SELECT *, CAST(sql_text AS CHAR(10000) CHARACTER SET utf8) AS Query FROM mysql.slow_log ORDER BY start_time DESC LIMIT 100 

   TRUNCATE mysql.slow_log; 


To process the logs use mysqldumpslow::
 mysqldumpslow -s c -t 100 mysql-slowquery.log > top100-c.log
 mysqldumpslow -s r -t 100 mysql-slowquery.log > top100-r.log
 mysqldumpslow -s ar -t 100 mysql-slowquery.log > top100-ar.log
 mysqldumpslow -s t -t 100 mysql-slowquery.log > top100-t.log
 mysqldumpslow -s at -t 100 mysql-slowquery.log > top100-at.log
 grep "FROM Status" mysql-slowquery.log | wc -l

Generate Profiling Data
--------------------------------------------------

Start the server with: 

   ./bin/platform -httpprofiler


Look at different profiles with:

   go tool pprof platform http://localhost:8065/debug/pprof/profile
   go tool pprof platform http://localhost:8065/debug/pprof/heap
   go tool pprof platform http://localhost:8065/debug/pprof/block
   go tool pprof platform http://localhost:8065/debug/pprof/goroutine

Check the process list in the MySQL Database
--------------------------------------------------

   SHOW FULL PROCESSLIST



Check the sql engine status in the MySQL Database
--------------------------------------------------

   SHOW ENGINE INNODB STATUS

