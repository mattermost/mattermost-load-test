# Terraform Loadtest Cluster

Setting up an AWS cluster using Terraform and the `ltops` tool is currently the best way to profile the mattermost-server. Note that while other cloud providers are on the roadmap, only AWS is supported at present.

## Installation

First install the load test binaries:
```
go get github.com/mattermost/mattermost-load-test/cmd/ltops
go get github.com/mattermost/mattermost-load-test/cmd/loadtest
go get github.com/mattermost/mattermost-load-test/cmd/ltparse
```

Then install [terraform](https://www.terraform.io/intro/getting-started/install.html), and optionally, install the [AWS CLI tool](https://aws.amazon.com/cli/).

## Configure for AWS

The `ltops` tool is hard-coded to use an AWS profile named `ltops`. Configure the credentials for this profile in `$HOME/.aws/credentials`, or use the AWS CLI to configure one for you:
```
aws configure --profile ltops
```

You will need to supply an Access Key ID and a Secret Access Key. For more information on setting up the credentials file, see https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-getting-started.html. Mattermost Core Committers should setup their credentials using the `mattermost-loadtest` account to isolate loadtests from other infrastructure and billing.

## Create a loadtest cluster

Creating a loadtest cluster using terraform requires specifying certain counts and instance types:
```
ltops create \
    --cluster cluster-name \
    --type terraform \
    --app-count 1 \
    --db-count 1 \
    --loadtest-count 1 \
    --app-type m4.large \
    --db-type db.r4.large
```

The `--app-count` flag determines how many Mattermost instances are deployed behind the proxy server. The `--app-type` flag determines the [Amazon EC2 instance type](https://aws.amazon.com/ec2/instance-types/) used for each Mattermost instance.

The `--db-count` flag determines how many Amazon RDS instances are deployed. One instance is always the master, with the remaining configured as read-only replicas. The `--db-type` flag determines the [Amazon RDS DB Instance Class](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/Concepts.DBInstanceClass.html).

More and bigger Mattermost and Amazon RDS instances will help your cluster scale to larger load tests.

The `--loadtest-count` flag determines how many EC2 instances are setup to run the `loadtest` tool. Each loadtest instance is hard-coded to the `m4.xlarge` [Amazon EC2 instance type](https://aws.amazon.com/ec2/instance-types/), and typically supports up to 5000 active entities and 20000 users. Configure additional loadtest instances only if your testing requires more active entities or users.

Note that, unlike `terraform` itself, the `ltops` tool does not currently support cluster resizing.

## Deploy Mattermost and tooling

Deploy Mattermost, the Nginx proxy and the `loadtest` tool to prepare for running a loadtest:
```
ltops deploy \
    --cluster cluster-name \
    --mattermost master \
    --license $HOME/mylicence.mattermost-license 
    --loadtests master
```

The `--mattermost` flag deploys the configured version to each application instance in the cluster. It supports `master` to deploy the latest server code; a branch name or pull request # to deploy a corresponding, successful build; or a URL or local file path to deploy a previously built [mattermost-server](https://github.com/mattermost/mattermost-server) package.

The `--license` flag supports a URL or local file path to activate enterprise features. Note that this flag is currently required with `--mattermost`, even though it is technically possible to loadtest a standalone server without the high availability enterprise feature.

The `--loadtests` flag supports `master` to deploy the latest loadtesting code, or a URL or local file path to deploy a previously built [mattermost-load-test](https://github.com/mattermost/mattermost-load-test) package.

This deployment step can be run as often as necessary to change the software deployed to the cluster. It is also possible to deploy just `--mattermost` (and `--license`) or `--loadtests`.

## Run a loadtest

Run a loadtest using the default configuration:
```
ltops loadtest --cluster cluster-name
```

The optional `--config` flag supports a local file path to customize your loadtest parameters. The default parameters are specified by [loadtestconfig.default.json](../loadtestconfig.default.json). Consult [loadtest.md] for more details.

## Generate loadtest results

To generate a markdown summary of the loadtest results:
```
ltparse results --file $HOME/.mattermost-load-test-ops/cluster-name/results/*.txt --display markdown
```

Consult [loadtest.md](loadtest.md#Results) for more details.

## Debugging your loadtest cluster

Display your cluster configurations:
```
ltops status
```

Launch the MySQL CLI against the master database instance configured for your cluster:
```
ltops db --cluster cluster-name
```

Connect via SSH to the first application instance (`ltops ssh app`):
```
ltops ssh app --cluster cluster-name
```

The `ssh` subcommand supports connecting to `app`, `loadtest`, `metrics` or `proxy` instances, as well as specifying `--instance` if there are more than one per type. It also supports immediately running a command instead of launching an interactive shell:
```
ltops ssh app --verbose --cluster cluster-name sudo systemctl restart mattermost
```

## Destroying your loadtest cluster

AWS clusters are generally billed regardless of whether or not a loadtest is active. To avoid incurring costs after your loadtest is complete, destroy your loadtest cluster:
```
ltops destroy --cluster cluster-name
```
