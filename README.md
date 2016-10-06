# Mattermost Load Test

## Fast Configure & Run

Run `make install` to compile and install the binaries. After that you should have access to `loadtest`, `mcreate`, and `mmanage`. For help with each of these commands, run them without parameters.

To run a load test from a fresh Mattermost install, first you have to create the first system admin user. After you have done that, edit the `loadtestconfig.json` file "AdminEmail" and "AdminPassword" with your system admin credentials. Then you should run the following command:
```
mcreate users -r -n 5 | mcreate teams -r -n 2 | mcreate channels -n 10 | mmanage login | mmanage jointeam | mmanage joinchannel > state.json
```
This will create 5 users 2 teams and 10 channels. It will then login all the users, join them to both teams and all the channels in those teams. It then saves the server state it created in the `state.json` file.
To start the loadtest run:
```
cat state.json | loadtest active
```
This will start a loadtest using the parameters in the loadtestconfig.json file. In that file you can edit how many user entities are created, the ramp up time, and all other available settings.

Statistics about the run will be output to standard output. More details tracing info and specific errors are output to the file `status.log`. If you want to follow along in realtime you can use `tail -f status.log` in another terminal to view the log in real-time. 

## Old loadtest

The old load tests are still available. To run them use `loadtest old`


## Advanced use

TODO: WRITE ADVANCED USE

There are two parts to the loadtest system. The initialization and manipulation stage, and the loadtest stage. 

