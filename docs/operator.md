# Load Test with the Mattermost Operator on Kubernetes

Follow these instructions to run a load test with the [Mattermost Kubernetes Operator](https://github.com/mattermost/mattermost-operator) for 100 users.

Steps for 5000, 10000, and 25000 user load tests is coming soon.

## 1. Set up a Kubernetes cluster

If you are on AWS, the easiest way to get a Kubernetes cluster running is to use Amazon's [eksctl tool](https://docs.aws.amazon.com/eks/latest/userguide/getting-started-eksctl.html). If you do not want to run on AWS, then see the [official Kubernetes documentation for setup instructions](https://kubernetes.io/docs/setup/#production-environment).

Use this chart to determine your cluster size:

| User Count | Node Size | Node Count |
| ---------- | --------- | ---------- |
| 100 | t3.medium (2CPU, 4GB) | 4 |

## 2. Install the Mattermost Operator

Follow [section 1 of the Mattermost Operator install instructions](https://github.com/mattermost/mattermost-operator#1-prerequisites) to install the Mattermost Operator and all its prerequisites.

## 3. Deploy a Mattermost Installation

### 3.1 AWS or Azure

If your Kubernetes cluster is on AWS or Azure, run the following command with the manifest file corresponding to your user count:


| User Count | Manifest |
| ---------- | -------- | 
| 100 | https://raw.githubusercontent.com/mattermost/mattermost-load-test/master/manifests/mattermost_100users_aws_azure.yaml |

```
$ kubectl apply -f <link-to-manifest>
```

Wait 3-5 minutes for the deploy to complete and then run the following:

```
$ kubectl get svc
```

This will show you three services. The one with the type `LoadBalancer` will have an IP or hostname under the `EXTERNAL-IP` column. Use this to access the Mattermost installation. Write it down, you will need it later.

Optionally, you can use Route53 or another DNS service to create a CNAME to the above external IP. This is not necessary for the load test.

### 3.2 Not on AWS or Azure

If your Kubernetes cluster is running somewhere other than on Azure or AWS, you'll need to install the [NGINX Ingress Controller](https://kubernetes.github.io/ingress-nginx/deploy/). Follow the instructions in that link to install it.

Once NGINX Ingress is installed, run the following command with the manifest file corresponding to your user count.

| User Count | Manifest |
| ---------- | -------- | 
| 100 | https://raw.githubusercontent.com/mattermost/mattermost-load-test/master/manifests/mattermost_100users_anywhere.yaml |

```
$ kubectl apply -f <link-to-manifest>
```

Wait 3-5 minutes for the deployment to complete.

To get the hostname or IP address to access your Mattermost installation, you need to look at the service for the ingress controller. You can do that with:

```
$ kubectl -n ingress-nginx get svc
```

Write it down, you will need it later.

### 4. Configure Mattermost

In a browser, go to the URL for the Mattermost installation.

Create an account and write down the credentials for that account. You will need them later.

Go to the `System Console -> General -> Users and Teams` and set `Max Users Per Team` to `100000`.

Then go to `System Console -> Customization -> Posts` and set `Enable Link Previews` to `true`.

Make sure to save on both pages.

## 5. Run the Load Test

### 5.1 Create the Profiling Service

Create the service used by the load test agent to profile the Mattermost app servers.

```
$ kubctl apply -f https://raw.githubusercontent.com/mattermost/mattermost-load-test/master/manifests/service.yaml
```

### 5.2 Configure the Load Test

Download the config map manifest file corresponding to your user count:

| User Count | Manifest |
| ---------- | -------- | 
| 100 | https://raw.githubusercontent.com/mattermost/mattermost-load-test/master/manifests/configmap_100users.yaml |

Edit the manifest and make the following replacements:

| String to Replace | Value to Use | Notes |
| ----------------- | ------------ | ----- |
| %MM_URL% | The URL used to access the Mattermost installation | Make sure to use http/ws unless you set up TLS then use https/wss |
| %DATA_SOURCE% | The DB connection string to MySQL | You can get this from running `kubectl describe po <app-pod-name>`. __Omit the `mysql://` part of the connection string.__ |
| %ADMIN_EMAIL% | The email you used to create the first account | This must be the first account created on the system |
| %ADMIN_PASSWORD% | The password you used to create the first account | |

Save the manifest and apply it with:

```
$ kubectl apply -f <path-to-configmap.yaml>
```

### 5.3 Bulk Load the Test Data

To bulk load the data needed for the load test, download the following job manifest:

https://raw.githubusercontent.com/mattermost/mattermost-load-test/master/manifests/bulkload_job.yaml

Edit the manifest and make the following replacements:

| String to Replace | Value to Use | Notes |
| ----------------- | ------------ | ----- |
| %DATA_SOURCE% | The DB connection string to MySQL | You can get this from running `kubectl describe po <app-pod-name>`. __Include the `mysql://` part of the connection string.__ |

Save and then run the bulk load with:

```
$ kubectl apply -f bulkload_job.yaml
```

To view the status of the bulk load, you can watch the logs of the pod starting with the name `mattermost-bulk-load*` using:

```
$ kubectl logs -f <bulk-load-pod-name>
```

The bulk load will take between 5-30 minutes to run based on your user count.

### 5.4 Run the Load Test Job

Download the job manifest file corresponding to your user count:

| User Count | Manifest |
| ---------- | -------- | 
| 100 | https://raw.githubusercontent.com/mattermost/mattermost-load-test/master/manifests/job_100users.yaml |

Edit the manifest and make the following replacements:

| String to Replace | Value to Use | Notes |
| ----------------- | ------------ | ----- |
| %DATA_SOURCE% | The DB connection string to MySQL | You can get this from running `kubectl describe po <app-pod-name>`. __Include the `mysql://` part of the connection string.__ |

Save the manifest and apply it with:

```
$ kubectl apply -f <path-to-job.yaml>
```

This will start the load test.

## 6. Monitor Load Test

To monitor the load test you can watch the logs of the pods running the load test. To do this, first list the pods:

```
$ kubectl get po
```

Then copy the pod name for the pods starting with `mattermost-load-test*`. Depending on your user count there may be one or more load test pods. Write these pod names down, you will need them to get the results of the load test.

Watch the logs for one of those pods by running:

```
$ kubectl logs -f <load-test-pod-name>
```

The load test will take between 15-45 minutes to run depending on your user count.

## 7. Parse Load Test Results

Once the load test is completed, the logs need to be parsed to see the results.

To do this you need the `ltparse` tool. Download it from https://github.com/mattermost/mattermost-load-test/releases.

Collect the logs from each pod that's name starts with `mattermost-load-test*`. To see these pods run:

```
$ kubectl get po
```

Then to collect the logs from each of the pods with:

```
$ kubectl logs <podname1> > lt.log
$ kubectl logs <podname2> >> lt.log
...
```

With all the logs collected, use `ltparse` to get the results:

```
$ ltparse results -f lt.log
```