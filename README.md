# Mattermost Load Test

Compile this project to run performance load tests on the Mattermost server. 

## Fast Configure & Run

### Installing Mattermost Load Test

Run `make install` to compile and install the binaries. This creates three (3) commands: `loadtest`, `mcreate`, and `mmanage`. For help with each of these commands, run them without parameters.

### Quickstart

1. Make sure you have run `make install` from above.
2. Run `./setup.sh` and follow the prompts. It will remind you to configure the server properly.
2. Run `./run.sh` and follow the prompts.


### Setting up a Load Test Environment 

#### Sample DB

There is a sample DB available under `sample-dbs/loadtest1-20000-shift.sql` to load this from the command line use `mysql -u username < file.sql`
You can then run a loadtest against it using the command `cat loadtest1-20000-shift-state.json | loadtest listenandpost`

#### Manually

In preparation for running a load test, install a Mattermost server and populate it with users and a team: 

1. Install a Mattermost server using any of the [online installation guides](https://docs.mattermost.com/guides/administrator.html#install-guides). 
2. Create an account on your Mattermost server. The first account created on a server is automatically provided the System Admin role. 
3. Edit the `loadtestconfig.json` file and set `AdminEmail` and `AdminPassword` to the credentials of your System Admin account. 
4. Then run the following command:

```
mcreate users -n 5 | mcreate teams -n 1 | mcreate channels -n 10 | mmanage login | mmanage jointeam | mmanage joinchannel -n 5 > state.json
```

This creates 5 users, 1 team and 10 channels. It then logs in all the users, joins them to the team and 5 channels in that team. The 5 channels will be the 5 channels following the users numer in sequance modulo the number of channels. 

The server state created is then saved to `state.json`.

### Running a Load Test 

To start the loadtest run:
```
cat state.json | loadtest listenandpost
```

This will start a loadtest using the parameters in the `loadtestconfig.json` file. In that file you can edit how many user entities are created, the ramp up time, and all other available settings.

Statistics about the run will be output to standard output. More details tracing info and specific errors are output to the file `status.log`. If you want to follow along in realtime you can use `tail -f status.log` in another terminal to view the log in real-time. 

### Notes 

The previous version of load test automation is available by running `loadtest old`

## Advanced use

TODO: WRITE ADVANCED USE

There are two parts to the loadtest system. The initialization and manipulation stage, and the loadtest stage. 
