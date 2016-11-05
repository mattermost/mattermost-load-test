// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package loadtestconfig

type UserEntitiesConfiguration struct {
	FirstEntityNumber                int
	LastEntityNumber                 int
	EntityRampupDistanceMilliseconds int
}

func (config *UserEntitiesConfiguration) SetDefaultsIfRequired() {
	if config.LastEntityNumber == 0 {
		config.LastEntityNumber = 100
	}

	if config.EntityRampupDistanceMilliseconds == 0 {
		config.EntityRampupDistanceMilliseconds = 100
	}
}
