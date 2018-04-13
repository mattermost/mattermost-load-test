# mattermost-load-test-ops
Work-in-progress tool for management of Mattermost load test clusters

# Installation

```
go get github.com/mattermost/mattermost-load-test-ops/cmd/ltops
```

or

```
git clone https://github.com/mattermost/mattermost-load-test-ops
dep ensure
go install ./cmd/ltops
```

Install terraform: https://www.terraform.io/intro/getting-started/install.html

Setup your aws cli: https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-getting-started.html
(really we just need the ~/.aws/credentials file to be populated)

# Running a loadtest

1. Create a cluster:
```
ltops create --name myloadtestcluster --app-count 1 --db-count 1 --loadtest-count 1 --app-type m4.large --db-type db.r4.large
```

2. Deploy Mattermost, configure proxy, loadtest
```
ltops deploy -c myloadtestcluster -m ~/go/src/github.com/mattermost/mattermost-server/dist/mattermost-enterprise-linux-amd64.tar.gz  -l ~/mylicence.mattermost-license -t ~/go/src/github.com/mattermost/mattermost-load-test/dist/mattermost-load-test.tar.gz
```

3. Run loadtests
```
ltops loadtest -c myloadtestcluster
```

4. Results will show up in ~/.mattermost-load-test-ops/myloadtestcluster/results

5. Delete cluster when done
```
ltops delete myloadtestcluster
```

# SSH into machines

SSH into app server 0:
```
ltops ssh app myloadtestcluster 0
```

SSH into proxy server 1:
```
ltops ssh proxy myloadtestcluster 1
```

SSH into loadtest server 0:
```
ltops ssh loadtest myloadtestcluster 0
```

# Get status of clusters

```
ltops status
```
