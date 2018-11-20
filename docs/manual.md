# Manual Loadtest Cluster

The `loadtest` tool many be run manually to loadtest an existing cluster, regardless of how that was cluster was deployed. This is especially useful when testing changes to `mattermost-load-test` itself and running a local loadtest.

# Configure your loadtest cluster

Follow the regular Mattermost [installation guide](https://docs.mattermost.com/guides/administrator.html) for the operating system that you're using. Be sure to pick a machine configuration to support your desired loadtest. If testing with an Enterprise license, setup [Metrics](https://docs.mattermost.com/deployment/metrics.html) to simplify debugging of any performance or configuration issues you encounter.

In addition to Mattermost itself, setup discrete machines to run the loadtest agents. Plan for one loadtest agent per 5000 active entities or 20,000 users. The machines should have similar specifications to the Mattermost application servers, down to matching ulimits.

## Deploy tooling

To run the latest loadtest code, install go on the loadtest agent machines and install the load test binaries:
```
go get github.com/mattermost/mattermost-load-test/cmd/loadtest
go get github.com/mattermost/mattermost-load-test/cmd/ltparse
```

## Configure a loadtest

Duplicate the default [configuration file](../loadtestconfig.default.json) and save as `loadtestconfig.json` on the load test machine. 

Configure `ConnectionConfiguration.ServerURL` and `ConnectionConfiguration.WebsocketURL` to point at your Mattermost instance or proxy server. Configure `ConnectionConfiguration.DriverName` and `ConnectionConfiguration.DataSource` to match your Mattermost server database configuration, noting that the loadtest agents must be able to connect directly to this database.

Configure `ConnectionConfiguration.AdminEmail` and `ConnectionConfiguration.AdminPassword` with credentials for a system administrator on the server.

Configure `ConnectionConfiguration.LocalCommands` to be `false`, unless the Mattermost server is running on the same (development) machine as the loadtest agent. Configure `ConnectionConfiguration.SSHHostnamePort`, and either `ConnectionConfiguration.SSHUsername` and `ConnectionConfiguration.SSHPassword`, or `ConnectionConfiguration.SSHKey` to allow the loadtest agent to connect via SSH to one of the Mattermost instances.

Consult [loadtestconfig.md](loadtestconfig.md) for more documentation on the configuration parameters.

## Run a loadtest

From each loadtest agent, invoke the `loadtest` tool:
```
loadtest all
```

The `loadtest` tool accepts various subcommands, including `all` and `basic`. Run `loadtest help` for more options.

## Generate loadtest results

To generate a markdown summary of the loadtest results:
```
ltparse results --file results.txt --display markdown
```

Consult [loadtest.md](loadtest.md#Results) for more details.
