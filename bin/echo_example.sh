
export DEVELOPMENT=false
## Database
export DBURL="localhost:28015"  # RethinkDB URL, Leave empty to disable rethinkdb usage
export DBUSER=""                # RethinkDB Pass
export DBPASS=""                # RethinkDB User
export BOLTFILE="cache.db"   # BoltDB File Storage

## Logs
export INFOLOG="stats.log"
export ERRORSLOG="errors.log"

## Network
export PLATFORMURL="https://example.com"
export SOCKETURL="wss://example:443"

## Test Plan Settings
export TESTPLAN="UserListenTestPlan"  # TP Structre to run, must be in registry
export THREADCOUNT="20"               # Number of TPs to run
export THREADOFFSET="0"               # Threads are have numerical names(ex: USEREMAIL_1@example.com)
export RAMPSEC="10"                   # Approximately how many seconds until all threads are active

## TP Specific Variables
export TESTCHANNEL="test-chit-chat-"
export MESSAGEBREAK="20"
export LOGINBREAK="20"
export USEREMAILPRE="emailprefix"
export USEREMAILPOST="@example.com"
export USERFIRST="Jane"
export USERLAST="Doe"
export USERNAME="usernameprefix"
export USERPASS="passwordprefix"
export TEAMNAME="exampleteam"
export MEDIAPERCENT="10"
export REPLYPERCENT="2"
export LOGINBREAK=1
