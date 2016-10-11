// Copyright (c) 2016 Spinpunch, Inc. All Rights Reserved.
// See License.txt for license information.

package loadtestconfig

type UserEntitiesConfiguration struct {
	NumClientEntities                int
	ActionFrequencyMultiplier        int
	EntityRampupDistanceMilliseconds int
}

func (config *UserEntitiesConfiguration) SetDefaultsIfRequired() {
	if config.NumClientEntities == 0 {
		config.NumClientEntities = 100
	}

	if config.ActionFrequencyMultiplier == 0 {
		config.ActionFrequencyMultiplier = 1
	}

	if config.EntityRampupDistanceMilliseconds == 0 {
		config.EntityRampupDistanceMilliseconds = 100
	}
}
