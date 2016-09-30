go build
rc=$?;
if
  (($rc != 0 ))
then
  echo "go build failed"
else
  export DBURL="localhost:28015"
  export PLATFORMURL="https://example.com"
  export SOCKETURL="wss://example.com:443"
  export BOLTFILE="cache.db"
  export DEVELOPMENT=false

  export THREADCOUNT="16000"
  export THREADOFFSET="100"
  export RAMPSEC="2000"
  export MESSAGEBREAK="240"
  export LOGINBREAK="100"
  export REPLYPERCENT="2"

  export TESTPLAN="UserListenTestPlan"
  export TESTCHANNEL="test-channel-"

  export USEREMAILPRE="user_"
  export USEREMAILPOST="@example.com"
  export USERFIRST="Jane"
  export USERLAST="Doe"
  export USERNAME="test_user_"
  export USERPASS="password"
  export TEAMNAME="team"
  ./mattermost-load-test
fi
