========================
Load Test Configuration
========================

The following configurable variables comprise the loadtestconfig.json. 
It is part of the ``mattermost-load-test-master`` directory at the top level. 
 

``mattermost-load-test-master > loadtestconfig.json``

The ``loadtestconfig.json`` configuration file is parted into  sections.


.. contents::Sections of JSON Configuration File
    :depth:1


Connection Configuration
===========================

The Connection Configuration section contains all the configuration variables related to the connection between the host machine and any network accessed machines. 

The Connection Configuration section contains variables for  Secure Shell (SSH) the database endpoint a local commands option for the command line interface (CLI) and other connection related variables.

The variables are as follow below. 

    


ServerURL 
-------------------------------------------------------

    ``ServerURL`` is the URL for the the load test.
    It should be the public facing URL of the Mattermost instance.
    
    -example    ``"ServerURL":http//localhost:8065``
    

WebsocketURL  
-----------------------------------------------------

    ``WebsocketURL`` is in most cases the same URL as ServerURL with `http`
    replaced with `ws` or `https` replaced with `wss`.

    -example    ``"WebsocketURL":ws//localhost:8065``

PProfURL 
-----------------------

    ``PProfURL`` is the URL for the profiling server.

    -example    ``"PProfURL":http//localhost:8067/debug/pprof``
    
    
    .. tip:: Package pprof in the *Go* API writes runtime profiling data in the format expected by the pprof visualization tool. For further inquiry visit the `Go API <https://golang.org/pkg/net/http/pprof/>`_.
    
    

DBEndpoint
------------------------

    ``DBEndpoint`` is the database endpoint for the load test configuration.
    

    -example    ``"DBEndpoint":mmuserIjqOyuhpSmBQwFOrl@tcp(loadtest1ar.cwr8afqyinx1.us-east-1.rds.amazonaws.com:3306)/mattermost?charset=utf8mb4utf8&readTimeout=20s&writeTimeout=20s&timeout=20s``

LocalCommands 
--------------------------

    ``LocalCommands`` runs Mattermost CLI commands locally instead of going through SSH. Set this configuration to true if you are running the load tests on the same machine as one of the application servers.
    
    -options    true/false
    -default    true


SSHHostnamePort 
----------------------------

    ``SSHHostnamePort`` is the hostname and port of any one of the application servers you are testing. Mattermost CLI commands are run here for the test.
    
    -example    ``"SSHHostnamePort":2222``

SSHUsername
-----------------------

    ``SSHUsername`` is the username for the SSH authentication.
    
    -example    ``"SSHUsername":spellChristopher``

SSHPassword
------------------------

    ``SSHPassword`` is the SSH password to connect with or "" if using a key.
    
    -example    ``"SSHPassword":yellowsubmarine07``

    

SSHKey
-------------

    ``SSHKey`` is the file path of the SSH key used to connect.
    
    -example    ``"SSHKey":12345abcd``
    
    .. where would and ssh key typically go in a Mattermost install?

MattermostInstallDir 
-------------------------------

    ``MattermostInstallDir`` is the location of the Mattermost installation
    directory. It must be the directory on the machine from which the CLI commands run.
    This is determined by the LocalCommands_ setting.


    -example    ``"MattermostInstallDir":/home/christopher/go/src/github.com/mattermost/mattermost-server/dist/mattermost``

ConfigFileLoc 
-------------------------------------

    ``ConfigFileLoc`` is the location of the Mattermost configuration file.
    If this user variable is not empty it is passed to the Mattermost binary as the --config parameter.
    
    -example    ``"ConfigFileLoc":/home/christopher/go/src/github.com/mattermost/mattermost-server/dist/mattermost/config``

AdminEmail 
-------------------------------------

      ``AdminEmail`` is the email address of the Administrator account on the server. It is created if it does not already exist.

    -example    ``"AdminEmail":success+ltadmin@simulator.amazonses.com``

AdminPassword 
-------------------------------------

    ``AdminPassword`` is the password for the Administrator account.

    -example    ``"AdminPassword":ltpassword``

SkipBulkload 
-------------------------------------

    ``SkipBulkload`` can be set to true to save time with verification if you are running the load test multiple times and have already loaded all the users into the database.


    -options    true/false
    -default    false
    -example    ``"SkipBulkload":false``

WaitForServerStart 
-------------------------------------

    ``WaitForServerStart`` decides whether to wait for the server to start before connecting.
    
    -options    true/false
    -default    false
    -example    ``"WaitForServerStart":false``

 

 
Loadtest Enviroment Configuration
=========================================================

The Loadtest Enviroment Configuration section contains all the variables for setting up the enviroment.
In the Mattermost lexicon enviroment means the load limits and timing of communications. 
This includes the number of teams, channels, users, posts, percentages, channel selections, 
and other things.

The variables are as follow below.


NumTeams
----------------------------------------------------

    ``NumTeams`` is the number of teams you want for the test.
    
    -example    ``"NumTeams":1,``




NumChannelsPerTeam
-----------------------------------------------------------------------------

    ``NumChannelsPerTeam`` is the number of channels allocated to each team.
    
    -example      ``"NumChannelsPerTeam":400,``
    



NumUsers 
----------------------------------

    ``NumUsers`` is the number of users in the load test.

    -example    ``"NumUsers":1000,``

NumPosts 
-----------------------------------------------------------------------------

    ``NumPosts`` is the number of posts in the load test.
    
        -example    ``"NumPosts":20000000,``


PostTimeRange
-----------------------------------------------------------------------------

    ``PostTimeRange`` is the range of time allocated to a post in milliseconds.
    
    -example    ``"PostTimeRange":2600000,``


PercentHighVolumeTeams
-----------------------------------------------------------------------------

    ``PercentHighVolumeTeams`` is the percentage of the teams created which will be *high volume*.
    
    -example ``"PercentHighVolumeTeams":0.2,``
    
.. note::
    
        In the *volume* variables, the number is the percentage. A number such as *0.2* represents 20 percent.



PercentMidVolumeTeams
-----------------------------------------------------------------------------

    ``PercentMidVolumeTeams`` is the percentage of the teams created which will be *medium volume*.
    
    -example    ``"PercentMidVolumeTeams":0.5,``
    


PercentLowVolumeTeams
-----------------------------------------------------------------------------

    ``PercentLowVolumeTeams`` is the percentage of the teams created which will be *low volume*.
    
    -example    ``"PercentLowVolumeTeams":0.3,``


PercentUsersHighVolumeTeams
-----------------------------------------------------------------------------

    ``PercentUsersHighVolumeTeams`` is the percentage of the users  allocated  to *high volume teams*.

    -example    ``"PercentUsersHighVolumeTeams":0.9,``

PercentUsersMidVolumeTeams
-----------------------------------------------------------------------------

    ``PercentUsersMidVolumeTeams`` is the percentage of the users  allocated  to *medium volume teams*.

    -example    ``"PercentUsersHighVolumeTeams":0.9,``

PercentUsersLowVolumeTeams
-----------------------------------------------------------------------------

    ``PercentUsersLowVolumeTeams`` is the percentage of the users  allocated  to *low volume teams*.
    
    -example    ``"PercentUsersLowVolumeTeams":0.1,``



PercentHighVolumeChannels
-----------------------------------------------------------------------------

    ``PercentHighVolumeChannels`` is the percentage of channels  allocated  to *high volume*.
    
    -example    ``"PercentHighVolumeChannels":01,``


PercentMidVolumeChannels
-----------------------------------------------------------------------------

    ``PercentMidVolumeChannels`` is the percentage of channels  allocated  to *medium volume*.
    
    -example    ``"PercentHighVolumeChannels":0.5,``

PercentLowVolumeChannels
-----------------------------------------------------------------------------

    ``PercentLowVolumeChannels`` is the percentage of channels  allocated  to *low volume*.
    
    -example    ``"PercentLowVolumeChannels":0.3,``

PercentUsersHighVolumeChannel
-----------------------------------------------------------------------------

    ``PercentUsersHighVolumeChannel`` is the percentage of channels  allocated  to *high volume*.
    
    -example    ``"PercentUsersHighVolumeChannel":0.1,``
    

PercentUsersMidVolumeChannel
-----------------------------------------------------------------------------

    ``PercentUsersMidVolumeChannel`` is the percentage of users allocated to *medium volume* channels. 
    
    -example    ``"PercentUsersMidVolumeChannel:" 0.003,``
    

PercentUsersLowVolumeChannel
-----------------------------------------------------------------------------

     ``PercentUsersLowVolumeChannel`` is  the percentage of users allocated to *low volume* channels. 
    
    -example    ``"PercentUsersLowVolumeChannel:" 0.0002,"``
    

HighVolumeTeamSelectionWeight
-----------------------------------------------------------------------------

    ``HighVolumeTeamSelectionWeight``  is the probability of selection of a *high volume*  team.
    
    -example    ``"HighVolumeTeamSelectionWeight":3`` 
    
    



MidVolumeTeamSelectionWeight
-----------------------------------------------------------------------------

    ``MidVolumeTeamSelectionWeight`` is  the probability of selection of a *medium volume*  team.
    
    -example    ``"MidVolumeTeamSelectionWeight":2``
    

LowVolumeTeamSelectionWeight
-----------------------------------------------------------------------------

    ``LowVolumeTeamSelectionWeight`` is the probability of selection of a *low volume*  team.
    
    -example    ``"LowVolumeTeamSelectionWeight":1``
    

HighVolumeChannelSelectionWeight
-----------------------------------------------------------------------------

    ``HighVolumeChannelSelectionWeight`` is the probability of selection of a *high volume*  channel.
    
    -example    ``"HighVolumeChannelSelectionWeight":1``
    
    

MidVolumeChannelSelectionWeight
-----------------------------------------------------------------------------

    ``MidVolumeChannelSelectionWeight`` is the probability of selection of a *medium volume*  channel.
    
    -example    ``"MidVolumeChannelSelectionWeight":3``
    
    

LowVolumeChannelSelectionWeight
-----------------------------------------------------------------------------

    `LowVolumeChannelSelectionWeight`` is the probability of selection of a *low volume*  channel.
    
    -example    ``"LowVolumeChannelSelectionWeight":1``

   


User Entities Configuration
==============================


An *entity* in the ``loadtestconfig.json`` configuration file is a boundary or limitation applied to the test.

The variables are as follow below.


TestLengthMinutes
-------------------------------------

The ``TestLengthMinutes`` variable determines the length of the test in minutes. 

    -example    ``"TestLengthMinutes":20``
    

NumActiveEntities
---------------------------------

    The ``NumActiveEntities`` variable determines how many entities are run. 
    This should be set to your number of expected active users.

    -example    ``"NumActiveEntities":500"``

ActionRateMilliseconds
---------------------------------------------

    The ``ActionRateMilliseconds`` variable determines how often each entity should take an action. 

    -example    ``"ActionRateMilliseconds" :60000``
    
.. tip::For an entity that only posts this would be the time between posts.
    

ActionRateMaxVarianceMilliseconds
-------------------------------------------------------

    ``ActionRateMaxVarianceMilliseconds`` is the maximum variance in action rate for each wait period. 


    -example    ``"ActionRateMaxVarianceMilliseconds":1500``
    
    

EnableRequestTiming
-----------------------------------------

     ``EnableRequestTiming``  enables timing for requests. 
          

    -options    true/false
    -default    true
    -example    ``"EnableRequestTiming":true``
    

UploadImageChance
--------------------------------------

    ``UploadImageChance`` is the chance an image will be uploaded.
    
    
    -example    ``"UploadImageChance":0.01``
     
     
     
.. note:: Chance here is used in the same way as *weight* in the ``loadtestconfig.json`` file. It means a probability deifned by the numer allocated to this variable. The higher the number the higher the chance. In this instance the *chance* is the uploading of an image. 
    

DoStatusPolling
--------------------------------

    ``DoStatusPolling`` varibale enables status polling.


    -options    true/false
    -default    true
    -example    ``"DoStatusPolling":true``



RandomizeEntitySelection
---------------------------------------------

    ``RandomizeEntitySelection`` allows a test where these  `User Entities Configuration`_  choices are randomly made.
    
    
    -options    true/false
    -default    false
    -example    ``"RandomizeEntitySelection":true``


   
DisplayConfiguration 
========================


    The Display Configuration section has two options for how the results of a test are seen.
    
    They are as follows. 
    

ShowUI
--------------------------------

    ``ShowUI`` determines whether or not the user interface (UI) is the means by which the test results are seen. It presents in graphs and other easy to read display methods.
    
    -options    true/false
    -default    false
    -example    ``"ShowUI":false``
    

LogToConsole 
-------------------------------

    ``LogToConsole`` variable determines whether  or not the results are piped to console. 
    
    -options    true/false
    -default    false
    -example    ``"LogToConsole":false``
   
Results Configuration 
========================

    The variables in the Results Configuration section configure all aspects of test reporting.
    
    The variables are as follow below.


CustomReportText 
------------------------------------------------------------------------------------

    ``CustomReportText`` allows for custom report text. 
    
    -options    true/false
    -default    false
    -example    ``"CustomReportText":false``


SendReportToMMServer
------------------------------------------------------------------------------------

    ``SendReportToMMServer`` determines whether or not a report is sent to the Mattermost server.
    
    -options    true/false
    -default    false
    -example    ``"SendReportToMMServer":false``


ResultsServerURL 
------------------------------------------------------------------------------------

    ``ResultsServerURL`` is the name of the URL for the server where the results will be sent. 
    
  
    -example    ``"SendReportToMMServer":false``

ResultsChannelId 
------------------------------------------------------------------------------------

    ``ResultsChannelId`` defines a specific channel from which you want to derive results.
    
    -example    ``"ResultsChannelId":1``


ResultsUsername 
------------------------------------------------------------------------------------

    ``ResultsUsername`` is the username to log onto the ResultsServerURL_.
    
    -example    ``"ResultsUsername":spellChristopher``



ResultsPassword 
------------------------------------------------------------------------------------

    ``ResultsPassword`` is the password to the  ResultsServerURL_.
    
    -example    ``"ResultsPassword":yellowsubmarine07``


PProfDelayMinutes
------------------------------------------------------------------------------------

    ``PProfDelayMinutes`` is a user defined delay in minutes on the profiling service.
    
    -example    ``"PProfDelayMinutes": 15``



PProfLength
------------------------------------------------------------------------------------

    ``PProfLength`` is the length is seconds of the profiling to be done. 
    
    -example    ``"PProfLength":450``





