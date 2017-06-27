#Installing the loadtest program

1. Install the build-essential package.

  `sudo apt-get install build-essential`

1. If you're installing loadtest on Linux: Download and install Go 1.8 for Linux:

  a. Download the Go binary.

    `wget https://storage.googleapis.com/golang/go1.8.linux-amd64.tar.gz`

  b. Install the Go binary.

    `sudo tar -C /usr/local -xzf go1.8.linux-amd64.tar.gz`

  c. Modify permissions on `/usr/local/go`. Replace {user} and {group} with the user and group that you are logged in as.

    `sudo chown -R {user}.{group} /usr/local/go`

1. If you're installing loadtest on Mac: Download and install Go:

    `brew install go`

1. Set up your Go workspace:

    a. `mkdir -p ~/go/bin`

    b. Add the following lines to your ~/.bashrc file:

      ```bash
      export GOPATH=$HOME/go
      export PATH=$PATH:$GOPATH/bin
      export PATH=$PATH:/usr/local/go/bin
      ulimit -n 8096
      ```

      Reload your bash configuration.

      `source ~/.bashrc`

1. Install Glide

  `curl https://glide.sh/get | sh`

1. Download the Mattermost loadtest code:

  a. Create the directory for the code.

    `mkdir -p ~/go/src/github.com/mattermost`

  b. Change to the directory that you created.

    `cd ~/go/src/github.com/mattermost`

  c. Clone the loadtest repository.

    `git clone https://github.com/mattermost/mattermost-load-test.git`

1. Build the loadtest binary

  ```console
  cd ~/go/src/github.com/mattermost/mattermost-load-test
  make install
  ```
Before running the load test, you must first customize the configuration. See the [Configuration File Documentation](loadtestconfig.md).
