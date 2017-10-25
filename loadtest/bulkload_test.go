// Copyright (c) 2017 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package loadtest

import "testing"

var testconfig1 = &LoadtestEnviromentConfig{
	NumTeams:           5,
	NumChannelsPerTeam: 100,
	NumUsers:           20000,

	PercentHighVolumeTeams: 0.20,
	PercentMidVolumeTeams:  0.50,
	PercentLowVolumeTeams:  0.30,

	PercentUsersHighVolumeTeams: 0.90,
	PercentUsersMidVolumeTeams:  0.50,
	PercentUsersLowVolumeTeams:  0.10,

	PercentHighVolumeChannels: 0.20,
	PercentMidVolumeChannels:  0.50,
	PercentLowVolumeChannels:  0.30,

	PercentUsersHighVolumeChannel: 0.90,
	PercentUsersMidVolumeChannel:  0.50,
	PercentUsersLowVolumeChannel:  0.10,
}

func TestStampServer(t *testing.T) {
	GenerateBulkloadFile(testconfig1)
}
