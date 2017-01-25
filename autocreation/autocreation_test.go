package autocreation

import (
	"testing"

	"github.com/mattermost/mattermost-load-test/loadtestconfig"
	"github.com/mattermost/mattermost-load-test/testlib"
	"github.com/mattermost/platform/model"
	"github.com/mattermost/platform/utils"
)

func TestDoAutocreation(t *testing.T) {
	type args struct {
		config loadtestconfig.LoadTestConfig
	}
	tests := []struct {
		name string
		args args
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			DoAutocreation(tt.args.config)
		})
	}
}

func Test_getAdminClient(t *testing.T) {
	testlib.RunMattermostServer()

	if client := getAdminClient("http://localhost:8065", "test@test.com", "passwd"); client == nil {
		t.Fatal("Got a nil client.")
	}
}

func Test_checkConfigForLoadtests(t *testing.T) {
	testlib.RunMattermostServer()

	var adminClient *model.Client
	if adminClient = getAdminClient("http://localhost:8065", "test@test.com", "passwd"); adminClient == nil {
		t.Fatal("Got a nil client.")
	}

	if err := checkConfigForLoadtests(adminClient); err != nil {
		t.Fatal("Failed to fix configuration")
	}

	if utils.Cfg.TeamSettings.MaxUsersPerTeam != 50000 {
		t.Fatal("Failed to set max users per team")
	}

	if *utils.Cfg.TeamSettings.EnableOpenServer != true {
		t.Fatal("Failed to open the server")
	}
}

func Test_getAdminForTestingClient(t *testing.T) {
	testlib.RunMattermostServer()

	var adminClient *model.Client
	if adminClient = getAdminClient("http://localhost:8065", "test@test.com", "passwd"); adminClient == nil {
		t.Fatal("Got a nil client.")
	}

	if testingClient := getAdminForTestingClient(adminClient); testingClient == nil {
		t.Fatal("Failed to retrieve tesing client")
	}
}

func Test_getTestingTeam(t *testing.T) {
	testlib.RunMattermostServer()

	var adminClient *model.Client
	if adminClient = getAdminClient("http://localhost:8065", "test@test.com", "passwd"); adminClient == nil {
		t.Fatal("Got a nil client.")
	}

	var testingClient *model.Client
	if testingClient = getAdminForTestingClient(adminClient); testingClient == nil {
		t.Fatal("Failed to retrieve tesing client")
	}

	var testingTeam *model.Team
	if testingTeam = getTestingTeam(testingClient); testingTeam == nil {
		t.Fatal("Failed to get testing team")
	}
}

func TestGetOrCreateChannels(t *testing.T) {
	testlib.RunMattermostServer()

	var adminClient *model.Client
	if adminClient = getAdminClient("http://localhost:8065", "test@test.com", "passwd"); adminClient == nil {
		t.Fatal("Got a nil client.")
	}

	var testingClient *model.Client
	if testingClient = getAdminForTestingClient(adminClient); testingClient == nil {
		t.Fatal("Failed to retrieve tesing client")
	}

	var testingTeam *model.Team
	if testingTeam = getTestingTeam(testingClient); testingTeam == nil {
		t.Fatal("Failed to get testing team")
	}

	channels := getOrCreateChannels(testingClient, testingTeam, 10)
	if channels == nil {
		t.Fatal("Was not able to create channels")
	}
}

func TestFetchExistingUsers(t *testing.T) {
	testlib.RunMattermostServer()

	var adminClient *model.Client
	if adminClient = getAdminClient("http://localhost:8065", "test@test.com", "passwd"); adminClient == nil {
		t.Fatal("Got a nil client.")
	}

	users := fetchExistingUsers(adminClient)
	if users == nil || len(users) < 1 {
		t.Fatal("Failed to fetch existing users")
	}
}

func TestGetOrCreateUsers(t *testing.T) {
	testlib.RunMattermostServer()

	var adminClient *model.Client
	if adminClient = getAdminClient("http://localhost:8065", "test@test.com", "passwd"); adminClient == nil {
		t.Fatal("Got a nil client.")
	}

	if users := getOrCreateUsers(adminClient, 10, make(map[string]*model.User)); len(users) != 10 {
		t.Fatal("Did not create the right number of users")
	}
}
