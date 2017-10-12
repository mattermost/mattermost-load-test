# Mattermost load test

The Mattermost load test provides infrastructure for simulating real-world usage of the Mattermost Enterprise Edition E20 at scale.

This guide will help you:

1. Deploy Mattermost in a production configuration, potentially in high availability mode.
2. Deploy a Mattermost load test server to apply simulated load to your production deployment.
3. Interpret the results of the load test and debug any problems.

If you have questions about configuration, please contact your Account Manager. An overview of support available to E20 customers is available at https://about.mattermost.com/support/

## Requirements

SOFTWARE

- **Software required for Mattermost production deployment** - see [Software and Hardware Requirements Guide](https://docs.mattermost.com/install/requirements.html) for software requirements

HARDWARE

- **Hardware required for Mattermost production deployment** - see [Software and Hardware Requirements Guide](https://docs.mattermost.com/install/requirements.html) for sizing hardware based on projected needs
- **Load Test Server** - to run load tests with hardware similar to Mattermost application server in production setup

## Running the tests

To run the load test simulation, complete the following:

### 1) Set up a Mattermost server to run the tests against.

Follow the regular Mattermost installation instructions for the operating system that you're using. Make sure you pick large enough machines in a configuration that will support the load you are testing for.

### 2) Set up a load test server.

The hardware specifications of the server running the load test should be similar to the hardware of your application server.

Install the `loadtest` command on the load test server using [these instructions](install-load-test-server.md).

### 3) Configure

Edit the [configuration file](https://github.com/mattermost/mattermost-load-test/blob/master/loadtestconfig.json) on the load test machine. Make sure the fields under "ConnectionConfiguration" are set correctly.

To produce useful results, set `NumUsers` to at least 5000, and `TestLengthMinutes` to at least 20.

You can find explanations of the configuration fields in the [Configuration File Documentation](loadtestconfig.md)

### 4) Run the tests

Now you can run the tests from the load test machine by using the command `loadtest all`.

A summary of activity will be output to the console so you can monitor the test. While the tests are running, it is a good idea to check the health of the server and the databases (e.g. reasonable CPU).

### 5) Analyze test results

Once the test is complete, a summary will be printed and saved to a file called results.txt. [You can see a sample output here](https://github.com/mattermost/mattermost-load-test/blob/master/docs/sample-results.txt).

The text file will have two sections:

a) Settings Report: Details on test length, [number of active entities](https://github.com/mattermost/mattermost-load-test/blob/master/loadtestconfig.md#numactiveentities), and the [action rate](https://github.com/mattermost/mattermost-load-test/blob/master/loadtestconfig.md#actionratemilliseconds).

b) Timings Report: Includes number of hits, error rates and response times of the most common API calls. 

You should expect low error rates (below 1%). If you see higher numbers, this may be an indication that the system was low performant during the load test. Check the file loadtest.log to find out potential issues. Note that the loadtest.log file will typically contain errors due to underlying race conditions, so focus on the most frequent errors for your investigation.

The timings report also includes response times for the API calls. Check that the response times are reasonable for your system. Note that response times are not comparable across organizations due to different network and infrustructure.

If you don't get any meaningful information from results.txt (for instance, all values are zeros), try increasing the number of users to 5000 and test length to 20 minutes. See the [configuration file](https://github.com/mattermost/mattermost-load-test/blob/master/loadtestconfig.json) for sample values.

## Common issues

### "Unable to set X. System Console is set to read-only when High Availability is enabled"

If you see this error the loadtests are trying to set a configuration setting but can't because HA mode is enabled. You will need to manually update your configuration. The required settings are:

 - `EnableOpenServer`: true
 - `MaxUsersPerTeam`: 50000 (or more)
 - `MaxChannelsPerTeam`: 50000 (or more)
 - `EnableIncomingWebhooks`: true
 - `EnableAdminOnlyIntegrations`: false

## Compiling for non master branch Mattermost

Note that the load tests only support master and possibly 1 version back (although you may need to use a branch)

1. Edit the `glide.yaml` file under `github.com/mattermost/platform` change `version: master` to the branch you want to build against. For a release the branch is called `release-x-x`, eg `release-3.9`
2. run `make clean`
3. run `make package`

## Look for slow SQL queries in MySQL

Consider using the following:

   SET GLOBAL log_output = 'TABLE';
   SET GLOBAL slow_query_log = 'ON';
   SET GLOBAL long_query_time = 1;
   SET GLOBAL log_queries_not_using_indexes = 'OFF';

   show global variables WHERE Variable_name IN ('log_output', 'slow_query_log', 'long_query_time', 'long_query_time', 'log_queries_not_using_indexes');

   SELECT *, CAST(sql_text AS CHAR(10000) CHARACTER SET utf8) AS Query FROM mysql.slow_log ORDER BY start_time DESC LIMIT 100

   TRUNCATE mysql.slow_log;

To process the logs use mysqldumpslow::
 - mysqldumpslow -s c -t 100 mysql-slowquery.log > top100-c.log
 - mysqldumpslow -s r -t 100 mysql-slowquery.log > top100-r.log
 - mysqldumpslow -s ar -t 100 mysql-slowquery.log > top100-ar.log
 - mysqldumpslow -s t -t 100 mysql-slowquery.log > top100-t.log
 - mysqldumpslow -s at -t 100 mysql-slowquery.log > top100-at.log
 - grep "FROM Status" mysql-slowquery.log | wc -l

## Generate profiling data

Start the server with:

   ./bin/platform -httpprofiler

Look at different profiles with:

   - go tool pprof platform http://localhost:8065/debug/pprof/profile
   - go tool pprof platform http://localhost:8065/debug/pprof/heap
   - go tool pprof platform http://localhost:8065/debug/pprof/block
   - go tool pprof platform http://localhost:8065/debug/pprof/goroutine
