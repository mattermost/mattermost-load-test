# mattermost-load-test-ops
🚧Work-in-progress tool for management of Mattermost load test clusters 🚧

## Basic Instructions

This command will create a cluster:

```
AWS_PROFILE=mm go run main.go -v create-cluster --name cb-loadtest-cluster --app-instance-type c5.large --app-instance-count 2 --db-instance-type db.m4.large
```

It should complete in about 5 minutes. If it doesn't, it'll probably time out in 15 or so and tell you there's insufficient capacity for the dedicated host. In that case, I usually just try a different instance type. Once it's complete, all your AWS resources will be in place.

To actually place a build on those resources, use this command:

```
AWS_PROFILE=mm go run main.go -v deploy ../mattermost-server/dist/mattermost-enterprise-linux-amd64.tar.gz --cluster-name cb-loadtest-cluster --license-file ~/Downloads/mattermost_ci-81564.mattermost-license
```

That command will take just a few seconds, and once it completes, you'll have a fully functional HA Mattermost installation. You can find the load balancer for it in AWS and open it up in your browser if you want to poke around it.

And finally, to run a load test:

```
AWS_PROFILE=mm go run main.go -v loadtest --cluster-name cb-loadtest-cluster --config loadtestconfig.json -- all
```

The instances have SSH / pprof ports exposed. So if you need to SSH into one, you can use this command:

```
AWS_PROFILE=mm go run main.go -v ssh --cluster-name cb-loadtest-cluster
```

Or for pprof, find the load balancer URL and use pprof directly:

```
go tool pprof -http 127.0.0.1:9002 http://cb-loadte-LoadBala-WM1VN1SYOCV9-1541897120.us-east-1.elb.amazonaws.com:8067/debug/pprof/profile
```

When you're done with a cluster:

```
AWS_PROFILE=mm go run main.go -v delete-cluster --name cb-loadtest-cluster
```
