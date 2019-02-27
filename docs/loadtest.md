# Loadtesting

## Bulkloading

Whether using `ltops loadtest` or the `loadtest` tool directly, loadtesting requires some preexisting data on the server for use in exercising the system. Depending on the loadtest configuration parameters, this could take anywhere from a few minutes to a few hours.

Terraform and manual loadtest clusters bulkload this data onto the server when the loadtest agent starts. If the cluster is being used repeatedly with the same configuration, configure `ConnectionConfiguration.SkipBulkload` to be `true` to speed up loadtest agent startup.

Kubernetes loadtest clusters also bulkload this data onto the server when the loadtest agent starts, but more intelligently bulkloads only from the first loadtest agent and only if the bulkload wasn't previously completed.

## Configuration

Terraform and manual loadtest clusters will, by default, run 500 active entities for 20 minutes before terminating the test. Configure the `LoadtestEnvironmentConfig` block in the loadtest configuration to fine tune the number of teams, users, channels and other parameters. Configure the `UserEntitiesConfiguration` block in the loadtest configuration to fine tune the number of active entities, test length, frequency of interaction and other parameters.

Kubernetes loadtest clusters rely exclusively on the `--users` flag provided during deployment. In the future, more customization may be possible.

## Metrics

To view the metrics and evaluate system performance while the tests are running, use `ltops status` to get the metrics URL. An instance of Grafana will be running there, displaying metrics emitted by the cluster and ingested by Prometheus. Some manual configuration is required when first connecting to Grafana:

1. Login with a username of `admin` and a password of `admin`.
2. Import these three dashboards, selecting `${DS_MATTERMOST}` as the source
   - https://grafana.com/dashboards/2539
   - https://grafana.com/dashboards/2542
   - https://grafana.com/dashboards/2545

Signs of system health depend greatly on the load you're running but the key metrics to watch are:
- `API Errors per Second` should be low. 5-10 is normal but if there is many more than that there may be an issue
- `Mean API Request Time` should generally be under 200ms. Some spikes are OK but a rising mean request time could be indicative of a problem. This may spike at the start-up of load tests because logging in of users is not rate limited. This is normal.
- `Mean Cluster Request Time` should generally be under 10ms
- `Goroutines` should be rise and then plateau. If it's continuously rising, there may be an issue
- `CPU Usage` will depend greatly on the load and the hardware specs of your cluster. Generally if it's maxing out, the load is too much for the hardware. Note that it's 100% per core, so a machine with 4 cores could hit 400% usage

Performance of the system in regards to how any actions the system is completing can be viewed by:
- `API Requests per Second` will let you know how many API requests the system is handling
- `Number of Messages per Second` tells the number of posts being sent in Mattermost per second
- `Number of Broadcasts per Second` tells the number of events being delivered from the server to a client using the WebSocket

## Results

Provided the `ltops` command stays running, loadtest logs against a cluster named `cluster-name` will be output to STDOUT, and simultaneously piped to `$HOME/.mattermost-load-test-ops/cluster-name/results`, with a file per loadtest instance. Be on the lookout for an unexpectedly high volume of errors: this typically indicates some kind of setup problem.

To generate a textual summary:
```
ltparse results --file $HOME/.mattermost-load-test-ops/cluster-name/results/*.txt --display text
```

To generate a markdown summary:
```
ltparse results --file $HOME~/.mattermost-load-test-ops/cluster-name/results/*.txt --display markdown
```

To aggregate results from multiple test runs:
```
cat /path/to/results/1 /path/to/results/2 /path/to/results/3 | ltparse results --aggregate --display markdown
```

To generate a markdown summary comparing the results with a previous results file representing a baseline:
```
ltparse results --file $HOME/.mattermost-load-test-ops/cluster-name/results --display markdown --baseline /path/to/baseline/results
```

Examining the individual API results. A successful loadtest has low error rates (below 1%), otherwise the system was likely overloaded during the load test. Timing will vary based on the configuration parameters, network setup and infrastructure, but should be within your target threshold to be considered successful.
