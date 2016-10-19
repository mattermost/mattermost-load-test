# Load Test Config

## ConnectionConfiguration

| Config Setting | Description |
|----|---|
| Server URL | The URL used to connect Mattermost server or proxy. Ex. `http://localhost:8065` |
| WebsocketURL | The URL used to connect Mattermost websocket. Ex. `ws://localhost:8065` |
| AdminEmail | The email address of the admin user you wish to use to create users/teams/channels. Note that this user will be a member of all the teams/channels created.  |
| Admin Password | The password for the admin user given in AdminEmail.  |
| RetryWebsockets | Whether or not tests that use websockets should try to reconnect on failure. |
| MaxRetryWebsocket | If RetryWebsockets is enabled, how many times to try reconnecting before giving up. |

## UserEntitiesConfiguration

| Config Setting | Description |
|----|---|
| FirstEntityNumber | The first entity number to start at. This will determine the first user that is used. Leave this at 0 unless you are testing across multiple loadtest machines. |
| LastEntityNumber | The last entity number to use. LastEntityNumber - FirstEntityNumber determines the total number of entities that will be used in the load test. |
| EntityRampupDistanceMilliseconds | This determines the wait time between starting individual user entities. For example, if you wanted all of your 20000 entities to start over 4 minutes. You would set this to 12 because 240000ms / 20000 entities = 12ms.

If your tying to test across multiple machines, you need to use `FirstEntityNumber` and `LastEntityNumber` to spread the load across machines.
For example, if you where testing 40000 users across 2 machines. The first machine would have FirstEntityNumber = 0 and LastEntityNumber = 20000. On the second machine you would set FirstEntityNumber = 20000 and LastEntityNumber = 40000.

## UserEntityPosterConfiguration

| Config Setting | Description |
|----|---|
| Posting Frequency Seconds | User entities will post at this frequency. The start of this time is when the entity was started. Therefore posts will be spread out according to `EntityRampupDistanceMilliseconds`. Ex. If set to 60, each entity will post every minute. |

## TeamCreationConfiguration

| Config Setting | Description |
|----|---|
| Num | The number of teams to create. |

## ChannelCreationConfiguration

| Config Setting | Description |
|----|---|
| NumPerTeam | The number of channels to create per team. |

## UserCreationConfiguration

| Config Setting | Description |
|----|---|
| Num | Number of users to create |
| NumChannelsToJoin | The number of channels each user should join. For now, the only distribution is a bunched flat distribution where each users joins the first NumChannelsToJoin channels that are not full in sequential order. |
