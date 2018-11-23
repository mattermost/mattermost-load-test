# Load Test Configuration

## ConnectionConfiguration

### ServerURL

The URL to direct the load. Should be the public facing URL of the Mattermost instance. 

### WebsocketURL

In most cases this will be the same URL as above with `http` replaced with `ws` or `https` replaced with `wss`.

### PProfURL

The URL or IP Address and port to a Mattermost application server on which pprof is enabled.

### DriverName

One of `mysql` or `postgres` to configure the database type.

### DataSource

The connection string to the master database.

### LocalCommands

Runs Mattermost CLI commands locally instead of over SSH. Set to `true` if you are running the load tests on the same machine as one of the app servers, or, for example, when developing locally.

### SSHHostnamePort

The hostname and port of any one of the app servers you are testing. Mattermost CLI commands will be run here.

### SSHHUsername

SSH username by which to connect.

### SSHPassword

SSH password by which to connect. Leave empty if using `SSHKey`.

### SSHKey

SSH key by which to connect. Leave empty if using `SSHPassword`.

### MattermostInstallDir

The location of the mattermost installation directory on the machine on which CLI commands are to be run.

### ConfigFileLoc

The location of the mattermost configuration file. If provided, the mattermost binary will be invoked with the `--config` parameter pointing at this file.

### AdminEmail

The email address of an admin account on the server. Will be created if it does not already exist.

### AdminPassword

The password for the admin account given above.

### SkipBulkload

If a loadtest is to be repeated and the bulkload has already completed successfully, set to this `true` to speed up loadtest agent startup.

### MaxIdleConns

The maximum number of idle connections held open from the loadtest agent to all servers.

### MaxIdleConnsPerHost

The maximum number of idle connections held open from the loadtest agent to any given server.

### IdleConnTimeoutMilliseconds

The number of milliseconds to leave an idle connection open between the loadtest agent an another server.

## LoadtestEnvironmentConfig

### NumTeams

The number of teams to bulkload.

### NumChannelsPerTeam

The number of public channels to bulkload per team.

### NumPrivateChannelsPerTeam

The number of private channels to bulkload per team.

### NumDirectMessageChannels

The number of direct message channels to bulkload. Note that this is constrained by `NumUsers`.

### NumGroupMessageChannels

The number of group message channels to bulkload. Note that this is constrained by `NumUsers`.

### NumUsers

The number of users to bulkload.

### NumChannelSchemes

The number of advanced permission channel schemes to bulkload.

### NumTeamSchemes

The number of advanced permission team schemes to bulkload.

### NumPosts

The number of posts to bulkload. Note that, by default, posts are not included in the bulkload but must be bulkloaded via the `loadtest loadposts` command.

### NumEmoji

The number of custom emoji to bulkload.

### NumPlugins

The number of plugins to bulkload.

### PostTimeRange

The time interval into the post in which posts are bulkloaded.

### ReplyChance

The chance that a post is made in reply to another. Set to `0` if you do not expect to use the threading feature.

### Percent(Users?)(High|Mid|Low)Volume(Teams|Channels), (High|Mid|Low)Volume(Team|Channel)SelectionWeight

Determines what percentage of teams or channels are considered high, medium and low volume, and thus how users are distributed to them. These numbers are hard to interpret, and thus not recommended for use at this time.

### PercentCustomSchemeTeams

The chance that a team has a custom scheme.

### PercentCustomSchemeChannels

The chance that a team has a custom channel.

## UserEntitiesConfiguration

### TestLengthMinutes

The number of minutes for which the test should run.

### NumActiveEntities

How many entities should be run by each load test machine. This should be set to your number of expected active users divided by the number of load test machines.

### ActionRateMilliseconds

The ActionRateMilliseconds specifies the length of time an entity waits -- on average -- between actions. For example, for an entity configured to only posts this would be the time between posts. For an entity that switches channels, this would be the time between switching channels, with multiple API requests made as part of a given channel switch.

### ActionRateMaxVarianceMilliseconds

This is the maximum variance in action rate for each wait period. So if the action rate was 2000ms and the max variance was 500ms, the min and max action rate would be 1500ms and 2500ms.

### ChannelLinkChance

The probability that a post will include a channel reference (`~channel`).

### UploadImageChance

The probabiliy that a post will include one or more uploaded images.

### LinkPreviewChance

The probability that loading a channel will require resolving a link preview.

### CustomEmojiChance

The probability that loading a channel will require fetching a custom emoji.

### NeedsProfilesByUsernameChance

The probability that loading a channel will require fetching unknown profiles by username.

### NeedsProfilesByIdChance

The probability that loading a channel will require fetching unknown profiles by id.

### NeedsProfilesStatusChance

The probability that loading a channel will require fetching profile statuses.

### DoStatusPolling

Whether or not an entity should include a periodic poll of user statuses.

### RandomizeEntitySelection

Whether or not to shuffle the users assigned to active entities.

## ResultsConfiguration

### PProfDelayMinutes

The number of minutes to wait for the cluster to stabilize before starting a pprof.

### PProfLength

The number of seconds over which to collect pprof stats.

## LogSettings

### EnableConsole

If true, the server outputs log messages to the console based on ConsoleLevel option.

### ConsoleLevel

Level of detail at which log events are written to the console.

### ConsoleJson

When true, logged events are written in a machine readable JSON format. Otherwise they are printed as plain text.

### EnableFile

When true, logged events are written to the file specified by the `FileLocation` setting. 

### FileLevel

Level of detail at which log events are written to log files.

### FileJson

When true, logged events are written in a machine readable JSON format. Otherwise they are printed as plain text.  

### FileLocation

The location of the log file.
