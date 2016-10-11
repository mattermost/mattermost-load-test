package oldloadtest

import "github.com/mattermost/mattermost-load-test/cmd/loadtest/oldloadtest/lib"

// TypeRegistry will hold class names by string for lookup
var TypeRegistry = make(map[string]lib.TestPlan)

func init() {
	TypeRegistry["UserConstantMediaPlan"] = &UserConstantMediaPlan{}
	TypeRegistry["UserConstantTestPlan"] = &UserConstantTestPlan{}
	TypeRegistry["UserJoinTestPlan"] = &UserJoinTestPlan{}
	TypeRegistry["UserListenTestPlan"] = &UserListenTestPlan{}
	TypeRegistry["UserPartyTestPlan"] = &UserPartyTestPlan{}
}
