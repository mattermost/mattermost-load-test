# Mattermost Load Test

The Load Test project is responsible for spinning up goroutines to simulate
load on any mattermost server. By creating `TestPlans`, the application can
generate a huge bulk of output that is sure to stress any system.

## Configure & Run
In most cases you should be in the command line and your working
directory should be the project folder

Gather your dependencies: `glide install`  
Build the project: `go build`  
Echo your environment variables: `export THREADS=3000; export ...`  
Run Project: `./mattermost-load-test`  
Enjoy the results....  

The `bin/` folder holds a few examples of short scripts that may help. Notice
the `echo_example.sh` that exports your setup. Or the `run_example.sh` that
could be duplicated for each scenario you want to run.

*RethinkDB* if you set the `DBURL` variable, then the application will expect
RethinkDB running. Simply set the variable to ` ` to ignore this feature.
This will eventually feed into a graphical front-end (**read: contribution**)

*HTTPS/WSS* if you set any of the mattermost server or websocket urls to use
a secure protocol, then the certificate must be signed. The platform library
has a hard time handling self-signed certificates.

## Application Flow
The main function will load the config from `environment.go` and maps env
variables to the `Config`. Loggers and Cache are created. A `GroupManager` is
spawned and responsible for creating it's `Group`. The `Group` handles the kick
off of Global Startup and staring for all `Threads` within the ramp up time.
`Threads` run the test plan that was selected via env variables
and report back to the `Group` which aggregates the metrics. The `GroupManger`
will sample the thread `Group` every 5 seconds, clear the data, and send off the results
to `Database` and logs.

## Docker Runtime
There is a limited test with Docker included. More details to be included.
