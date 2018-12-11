# Common issues

### "Unable to set X. System Console is set to read-only when High Availability is enabled"

If you see this error the loadtests are trying to set a configuration setting but can't because HA mode is enabled. You will need to manually update your configuration. The required settings are:

 - `EnableOpenServer`: true
 - `MaxUsersPerTeam`: 50000 (or more)
 - `MaxChannelsPerTeam`: 50000 (or more)
 - `EnableIncomingWebhooks`: true
 - `EnableAdminOnlyIntegrations`: false

### "Run Test Failed: Unable to connect to server dial tcp: missing address"

Check that your SSH fields are set correctly in the loadtest config and try again. [Find more detail on the config settings here](https://github.com/mattermost/mattermost-load-test/blob/master/loadtestconfig.md#connection-configuration).

### I can't setup the load tests to use SSH

You can manually generate and load the test users into the Mattermost server manually.

1. Run `loadtest genbulkload`. A file called `loadtestbulkload.json` should be created.
2. Upload this file to the Mattermost app server.
3. On the Mattermost app server run `./bin/mattermost import bulk --workers 64 --apply loadtestbulkload.json`
4. Make sure you set the configuration setting "SkipBulkLoad" to true.
