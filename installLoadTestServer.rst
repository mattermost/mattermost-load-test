===========================
Install Load Test  
===========================


The *loadtest* is installed atop an existing Mattermost installation. If you do not have Mattermost installed along with its ancillary componets then the installations in  must be completed.


Prerequisites to Load Test
===========================================

From the `Mattermost Administrator's Guide <https://docs.mattermost.com/guides/administrator.html>`_ follow the instructions for a standard install. 

Once completed the basic installation follow the instructions for the *loadtest* install defined below.

Load Test Install
==========================================

The following steps define the *loadtest* install.


1. Install the build-essential package.

    ``sudo apt-get install build-essential``
    
Linux
--------------

1. Open a ``Terminal`` and download the Go 1.8 binary file for Linux with the following command:

    ``wget https://storage.googleapis.com/golang/go1.8.linux-amd64.tar.gz``

#. Install the Go 1.8 binary file for Linux with the following command:

      ``sudo tar -C /usr/local -xzf go1.8.linux-amd64.tar.gz``

#. Modify permissions on ``/usr/local/go`` by replacing {user} and {group} with the user and group to which you are logged in. Use the following command:

      ``sudo chown -R {user}.{group} /usr/local/go``
      
Mac
-------------------------------

1. Open a ``Terminal`` and run the following command:

    ``brew install go``

#. Set up a  Go workspace with the following command:

    ``mkdir -p ~/go/bin``

#. Add the following lines to the ``~/.bashrc`` file:

    ::      
        
        export GOPATH=$HOME/go
        export PATH=$PATH:$GOPATH/bin
        export PATH=$PATH:/usr/local/go/bin
        ulimit -n 8096
      

#. Reload your bash configuration with the following command:

    ``source ~/.bashrc``.
    
Install Glide on Linux and Mac
--------------------------------------------------

#. Install Glide with the following command:

    ``curl https://glide.sh/get | sh``
    
Download and Compile Loadtest on Linux and Mac
--------------------------------------------------------------------------------

#. Download the Mattermost `loadtest` code and create the directory for the code with the following command.

      ``mkdir -p ~/go/src/github.com/mattermost``

#. Change to the directory that you created with the following command:

      ``cd ~/go/src/github.com/mattermost``

#. Clone the `loadtest` repository with the following command:

      ``git clone https://github.com/mattermost/mattermost-load-test.git``

#. Change to the `loadtest` directory you created with the following command:

    ``cd ~/go/src/github.com/mattermost/mattermost-load-test``
    
#. Build the `loadtest` binary with the following command:
    
    ``make install``
   
   .. note::   
        You have to  set your test configuration before running a load test.  See the `Configuration File Documentation <loadTestConfiguration.rst>`_ for documentation on ``loadtestconfig.json``.
