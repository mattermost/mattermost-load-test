# Mattermost load test [![Docker Build Status](https://img.shields.io/docker/build/mattermost/mattermost-load-test.svg)](https://hub.docker.com/r/mattermost/mattermost-load-test/)

A set of tools for testing/proving Mattermost servers under load. 

## Loadtesting with the ltops (load test ops) tool

The ltops tool allows you to easily spin up and load test a cluster of Mattermost servers with all the trimmings. Currently it supports AWS via [Terraform](https://www.terraform.io/) and Kubernetes.

### Installation

Download the binaries from the latest release https://github.com/mattermost/mattermost-load-test/releases.

or

Install with go get
```
go get github.com/mattermost/mattermost-load-test/cmd/ltops
```

or

Clone the repository and build for yourself.

```
git clone https://github.com/mattermost/mattermost-load-test
make install
```

Type `ltops` to check tool is installed properly. For help with any command, use `ltops help <command>`

### Kubernetes

We recommend running load tests on Kubernetes as it's platform agnostic and makes set up simpler.

#### Configuration

You need to have an existing Kubernetes cluster configured. If you're not sure if you have one, then you probably don't.

To set up a Kubernetes cluster, use one of the following guides:
* AWS - https://github.com/kubernetes/kops/blob/master/docs/aws.md 
* Azure - https://github.com/Azure/acs-engine/blob/master/docs/kubernetes/deploy.md
* Google Cloud Engine - https://kubernetes.io/docs/getting-started-guides/gce/

See https://kubernetes.io/docs/setup/pick-right-solution/ for more options.

You'll also need to install kubectl and helm.

Install kubectl: https://kubernetes.io/docs/tasks/tools/install-kubectl/

Type `kubectl` to check the tool is installed properly.

Install helm: https://docs.helm.sh/using_helm/#installing-helm

Type `helm` to check the tool is installed properly.

Make sure helm is configured on your cluster and locally by running `helm init --upgrade`

#### Set up a load test with Kubernetes

1. Set up a cluster:
```
ltops create --name myloadtestcluster --type kubernetes
```

2. Deploy and configure the helm chart:
```
ltops deploy -c myloadtestcluster --license ~/mylicence.mattermost-license --users 5000
```

3. Wait 5-10 minutes for the helm release to spin up. It should be ready when the following command shows `SiteURL` and `Metrics` wit h an IP address or URL:
```
ltops status
```

4. [Go here to run a load test](https://github.com/mattermost/mattermost-load-test#run-a-load-test)

### Terraform

If you want to run load test clusters on AWS, you need to install terraform.

Install Terraform: https://www.terraform.io/intro/getting-started/install.html

Type `terraform` to check tool is installed properly.

#### Configure for AWS

Fill in your `~/.aws/credentials` file. The ltops tool will use the profile named `ltops`. You can add a profile using the aws CLI:
```
aws configure --profile ltops
```
More info on setting up the credentials file here: https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-getting-started.html


#### Set up a load test with Terraform

1. Create a cluster:
```
ltops create --name myloadtestcluster --type terraform --app-count 1 --db-count 1 --loadtest-count 1 --app-type m4.large --db-type db.r4.large
```

2. Deploy Mattermost, configure proxy, loadtest. Note that the options support local files and URLs.
```
ltops deploy -c myloadtestcluster -m https://releases.mattermost.com/4.9.2/mattermost-4.9.2-linux-amd64.tar.gz -l ~/mylicence.mattermost-license -t https://releases.mattermost.com/mattermost-load-test/mattermost-load-test.tar.gz
```
3. [Go here to run a load test](https://github.com/mattermost/mattermost-load-test#run-a-load-test)

### Run a load test

Now that you have a cluster set up in either AWS or Kubernetes, do the following to run a load test:

1. Run load tests:
    ```
    ltops loadtest -c myloadtestcluster
    ```
    - This command will do two things:
        1. Bulk load the data needed for the tests. Depending on how many users you're running with this could take anywhere from a few minutes to an hour.
        2. Runs a load test, coordinating between all the load test agents. By default the load test will run for 20 minutes.

2. To view the metrics and evaluate system performance while the tests are running, use `ltops status` to get the metrics URL.
    - At that URL, login with `admin/admin`
    - Import these three dashboards, selecting ${DS_MATTERMOST} as the source
        - https://grafana.com/dashboards/2539
        - https://grafana.com/dashboards/2542
        - https://grafana.com/dashboards/2545
    - Signs of system health depend greatly on the load you're running but the key metrics to watch are:
        - `API Errors per Second` should be low. 5-10 is normal but if there is many more than that there may be an issue
        - `Mean API Request Time` should generally be under 200ms. Some spikes are OK but a rising mean request time could be indicative of a problem. This may spike at the start-up of load tests because logging in of users is not rate limited. This is normal.
        - `Mean Cluster Request Time` should generally be under 10ms
        - `Goroutines` should be rise and then plateau. If it's continuously rising, there may be an issue
        - `CPU Usage` will depend greatly on the load and the hardware specs of your cluster. Generally if it's maxing out, the load is too much for the hardware. Note that it's 100% per core, so a machine with 4 cores could hit 400% usage
    - Peformance of the system in regards to how any actions the system is completing can be viewed by:
        - `API Requests per Second` will let you know how many API requests the system is handling
        - `Number of Messages per Second` tells the number of posts being sent in Mattermost per second
        - `Number of Broadcasts per Second` tells the number of events being delivered from the server to a client using the WebSocket

3. Logs, including loadtest results will show up in ~/.mattermost-load-test-ops/myloadtestcluster/results
    - To view them as the test is running, run `tail -f` on any of the files in that directory
    - Some errors are OK but if you're seeing pages and pages of errors you likely have an issue

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

### SSH into machines

For AWS, this will actually SSH into the EC2 instances.

For Kubernetes, this will open an interactive shell to pods in the cluster.

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

### Destroy a cluster

```
ltops delete myloadtestcluster
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
