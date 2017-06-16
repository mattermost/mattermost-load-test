# Load Test Configuration

## Connection Configuration

### ServerURL

The URL to direct the load. Should be the public facing URL of the Mattermost instance. 

### WebsocketURL

In most cases this will be the same URL as above with `http` replaced with `ws` or `https` replaced with `wss`.

### LocalCommands

Runs Mattermost CLI commands locally instead of over SSH. Set to true if you are running the load tests on the same machine as one of the app servers. 

### SSHHostnamePort

The hostname and port of any one of the app servers you are testing. Mattermost CLI commands will be run here.

### SSHHUsername

Username to connect over SSH with.

### SSHPassword

Password to connect with or "" if using a key.

### SSHKey

File path of the SSH key to connect with.

### MattermostInstallDir

The location of the mattermost installation directory on the machine we are going to run CLI commands on. (Determined by the LocalCommands setting)

### ConfigFileLoc

The location of the mattermost configuration file. If not empty will be passed to the mattermost binary as the --config parameter.

### AdminEmail

The email address of an admin account on the server. Will be created if it does not already exist.

### AdminPassword

The password for the admin account given above.

### SkipBulkload

If your running the test multiple times and know you have already loaded all the users into the database, this can be set to true to save time verifying this.

## Loadtest Environment Config

These settings control the creation of users and teams. 

- Percent*Volume(Teams/Channels) determines what percentage of teams/channels are considered high/med/low volume.
- PercentUsers*Volume(Teams/Channels) determines what percentage of users are in a team/channel considered high/med/low volume.
- SelectionWeight settings determine how likely that class of team/channel is to be selected when picking a team/channel at random. (So a higher weight means more posts will go to it)

## User Entities Configuration

### TestLengthMinutes

How long the test should run for.

### NumActiveEntities

How many entities should be run. This should be set to your number of expected active users.

### ActionRateMilliseconds

How often each entity should take an action. For example, for an entity that only posts this would be the time between posts.

### ActionRateMaxVarianceMilliseconds

This is the maximum variance in action rate for each wait period. So if the action rate was 2000ms and the max variance was 500ms. The min and max action rate would be 1500ms and 2500ms.

### EnableRequestTiming

Set this to true

## Display Configuration

### ShowUI

Show the fancy UI with graphs.

### LogToConsole

If not showing the fancy UI, enable to disable output to the console. 
