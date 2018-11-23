# Kubernetes Loadtest Cluster

Although currently in beta, Mattermost Kubernetes is platform agnostic and simpler to setup than its Terraform equivalent.

## Installation

First install the load test binaries:
```
go get github.com/mattermost/mattermost-load-test/cmd/ltops
go get github.com/mattermost/mattermost-load-test/cmd/loadtest
go get github.com/mattermost/mattermost-load-test/cmd/ltparse
```

Then install [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) and [helm](https://docs.helm.sh/using_helm/#installing-helm). 

You will also require an existing Kubernetes cluster to which to deploy Mattermost and the loadtesting tools. To set up a Kubernetes cluster, use one of the following guides:
* AWS - https://github.com/kubernetes/kops/blob/master/docs/aws.md 
* Azure - https://github.com/Azure/acs-engine/blob/master/docs/kubernetes/deploy.md
* Google Cloud Engine - https://kubernetes.io/docs/getting-started-guides/gce/

See also https://kubernetes.io/docs/setup/pick-right-solution/ for more options.

## Create a loadtest cluster

Creating a loadtest cluster using Kubernetes requires very little configuration, as the resources required are created as part of the `deploy` step below:
```
ltops create --cluster cluster-name --type kubernetes
```

## Deploy Mattermost and tooling

Deploy the Mattermost loadtesting helm chart, deploying Mattermost, the Nginx proxy and the `loadtest` tool to your Kubernetes cluster:
```
ltops deploy \
    --cluster cluster-name \
    --license ~/mylicence.mattermost-license \
    --users 5000
```

The `--license` flag supports a URL or local file path to activate enterprise features. Note that this flag is currently required, even though it is technically possible to loadtest a standalone server without the high availability and metrics enterprise feature.

The `--users` flag is a proxy for configuring the number of application, database and loadtesting instances. Use a number appropriate for your loadtesting setup. Your Kubernetes cluster must have sufficient resources for the deployment to succeed.

The cluster deployment is currently asynchronous, and may take up to 10 minutes. Check the status of your clusters:
```
ltops status
```

If `SiteURL` and `Metrics` shows an IP address or URL, the cluster should be online and ready to run a loadtest.

## Run a loadtest

Run a loadtest using the default configuration:
```
ltops loadtest --cluster cluster-name
```

Note that the Kubernetes cluster does not yet support the `--config` flag to customize your loadtest parameters, relying exclusively on the `--users` parameter configured during deployment above. Consult [loadtest.md](loadtest.md) for more details.

## Generate loadtest results

To generate a markdown summary of the loadtest results:
```
ltparse results --file $HOME/.mattermost-load-test-ops/myloadtestcluster/results/*.txt --display markdown
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

The `ssh` subcommand supports connecting to `app`, `loadtest`, `metrics` or `proxy` instances, as well as specifying `--instance` if there are more than one per type. Unlike its Terraform equivalent, however, it does not support immediately running a command instead of launching an interactive shell.

## Destroying your loadtest cluster

To release the resources used by your loadtest cluster (but leave your Kubernetes cluster intact), destroy your loadtest cluster:
```
ltops destroy --cluster cluster-name
```
