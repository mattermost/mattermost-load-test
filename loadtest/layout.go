// Copyright (c) 2017 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package loadtest

// LayoutConfig

const (
	HIGH_VOLUME = 1
	MID_VOLUME  = 2
	LOW_VOLUME  = 3
)

type LayoutTeam struct {
	Num      int
	Volume   int
	Channels []LayoutChannel
}

type LayoutChannel struct {
	Num int
}

type OrganizationLayout struct {
	Config             *LoadtestEnviromentConfig
	TeamNameToIdMap    map[string]string
	ChannelNameToIdMap map[string]string // Assumes unique channel names
}
