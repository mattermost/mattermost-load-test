# Mattermost load test [![Docker Build Status](https://img.shields.io/docker/build/mattermost/mattermost-load-test.svg)](https://hub.docker.com/r/mattermost/mattermost-load-test/)

A set of tools for testing/proving Mattermost servers under load. 

## Loadtesting with the ltops (load test ops) tool

The ltops tool allows you to easily spin up and loadtest a cluster of Mattermost servers with all the trimmings. Currently it supports AWS with support for other cloud platforms and Kubernetes planned in the future. It is powered by [Terraform](https://www.terraform.io/)

### Installation

Install with go get
```
go get github.com/mattermost/mattermost-load-test/cmd/ltops
```

or clone the repository and build for yourself.

```
git clone https://github.com/mattermost/mattermost-load-test
make install
```

Type `ltops` to check tool is installed properly. For help with any command, use `ltops help <command>`

Install Terraform: https://www.terraform.io/intro/getting-started/install.html

Type `terraform` to check tool is installed properly.

### Configure for AWS

Fill in your `~/.aws/credentials` file. The ltops tool will use the profile named `ltops`. You can add a profile using the aws CLI:
```
aws configure --profile ltops
```
More info on setting up the credentials file here: https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-getting-started.html


### Running a loadtest

1. Create a cluster:
```
ltops create --name myloadtestcluster --app-count 1 --db-count 1 --loadtest-count 1 --app-type m4.large --db-type db.r4.large
```

2. Deploy Mattermost, configure proxy, loadtest. Note that the options support local files and URLs.
```
ltops deploy -c myloadtestcluster -m https://releases.mattermost.com/4.9.2/mattermost-4.9.2-linux-amd64.tar.gz -l ~/mylicence.mattermost-license -t https://releases.mattermost.com/mattermost-load-test/mattermost-load-test.tar.gz
```

3. Run loadtests
```
ltops loadtest -c myloadtestcluster
```

4. Logs, including loadtest results will show up in ~/.mattermost-load-test-ops/myloadtestcluster/results

To generate a textual summary:
```
ltparse results --file ~/.mattermost-load-test-ops/myloadtestcluster/results --display text
```

To generate a markdown summary:
```
ltparse results --file ~/.mattermost-load-test-ops/myloadtestcluster/results --display markdown
```

To aggregate results from multiple test runs:
```
cat /path/to/results/1 /path/to/results/2 /path/to/results/3 | ltparse results --aggregate --display markdown
```

To generate a markdown summary comparing the results with a previous results file representing a baseline:
```
ltparse results --file ~/.mattermost-load-test-ops/myloadtestcluster/results --display markdown --baseline /path/to/other/results
```

5. Delete cluster when done
```
ltops delete myloadtestcluster
```

### SSH into machines

SSH into app server 0:
```
ltops ssh app myloadtestcluster 0
```

SSH into proxy server 1:
```
ltops ssh proxy myloadtestcluster 1
```

SSH into loadtest server 0:
```
ltops ssh loadtest myloadtestcluster 0
```

### Get status of clusters

```
ltops status
```


## Using the loadtest agent directly

### 1) Setup your Mattermost machines

Follow the regular Mattermost installation instructions for the operating system that you're using. Make sure you pick large enough machines in a configuration that will support the load you are testing for. If you have access to Enterprise Edition, make sure you setup [Metrics](https://docs.mattermost.com/deployment/metrics.html) as this will allow you to more easily debug any performance or configuration issues you encounter.

### 2) Setup loadtest servers

You should use 1 machine per 20K users you wish to test. The loadtest machines should be similar hardware to the Mattermost application servers. Make sure you set the ulimits on these machines the same as you did on the Mattermost application servers.

Download and unpack the loadtest agent on each loadtest machine: https://releases.mattermost.com/mattermost-load-test/mattermost-load-test.tar.gz

### 3) Configure loadtest instances

Edit the [configuration file](https://github.com/mattermost/mattermost-load-test/blob/master/loadtestconfig.json) on the load test machine. Alternatively, you can use environment variables such as `MMLOADTEST_CONNECTIONCONFIGURATION_SERVERURL` to set configuration values. Make sure the fields under "ConnectionConfiguration" are set correctly.

To produce useful results, set `NumUsers` to at least 5000, and `TestLengthMinutes` to at least 20.

You can find explanations of the configuration fields in the [Configuration File Documentation](loadtestconfig.md)

### 4) Run the tests

Now you can run the tests by invoking the command `loadtest all` on each load test machine.

### 5) Analyze test results

Once the test is complete, a summary will be printed and saved to a file called results.txt.

The text file will have two sections:

a) Settings Report: Details on test length, [number of active entities](https://github.com/mattermost/mattermost-load-test/blob/master/loadtestconfig.md#numactiveentities), and the [action rate](https://github.com/mattermost/mattermost-load-test/blob/master/loadtestconfig.md#actionratemilliseconds).

b) Timings Report: Includes number of hits, error rates and response times of the most common API calls. 

You should expect low error rates (below 1%). If you see higher numbers, this may be an indication that the system was low overloaded during the load test. Check the file loadtest.log to find out potential issues. Note that the loadtest.log file will typically contain errors due to underlying race conditions, so focus on the most frequent errors for your investigation.

The timings report also includes response times for the API calls. Check that the response times are reasonable for your system. Note that response times are not comparable across organizations due to different network and infrustructure.

## Development

If you have followed the [Mattermost developer setup instructions](https://docs.mattermost.com/developer/dev-setup.html) you should be good to go.
