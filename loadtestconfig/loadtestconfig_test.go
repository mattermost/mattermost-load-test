// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package loadtestconfig

import "testing"
import "os"

func TestLoadDefaults(t *testing.T) {
	SetupConfig()
	config := GetConfig()

	if *config != defaultSettings {
		t.Fatal("Config was not written properly. Check you set defaults correctly in loadtestconfig.go")
	}

	// Do it again now that the file has been created
	SetupConfig()
	config2 := GetConfig()

	if *config2 != defaultSettings {
		t.Fatal("Config was not written properly.")
	}

	// Cleanup the file we made
	os.Remove("loadtestconfig.json")
}
