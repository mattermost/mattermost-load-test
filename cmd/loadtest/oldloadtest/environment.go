package oldloadtest

// EnvironmentalLoad sturcture will be parsed for environment variables to find
type EnvironmentalLoad struct {
	PlatformURL  string `env:"PLATFORMURL" envDefault:""`
	SocketURL    string `env:"SOCKETURL" envDefault:""`
	UserEmail    string `env:"USEREMAILPRE" envDefault:"user_"`
	EmailDomain  string `env:"USEREMAILPOST" envDefault:"@example.com"`
	UserFirst    string `env:"USERFIRST" envDefault:"Jane"`
	UserLast     string `env:"USERLAST" envDefault:""`
	UserName     string `env:"USERLAST" envDefault:"TestUser"`
	UserPassword string `env:"USERPASS" envDefault:"password-"`
	TeamName     string `env:"TEAMNAME" envDefault:"example-team"`
	Development  bool   `env:"DEVELOPMENT" envDefault:"true"`

	TestPlan     string `env:"TESTPLAN" envDefault:"UserListenTestPlan"`
	Threads      int    `env:"THREADCOUNT" envDefault:"20"`
	ThreadOffset int    `env:"THREADOFFSET" envDefault:"0"`
	RampSec      int    `env:"RAMPSEC" envDefault:"10"`

	LogPath   string `env:"INFOLOG" envDefault:"stats.log"`
	ErrorPath string `env:"ERRORSLOG" envDefault:"errors.log"`
	DbURL     string `env:"DBURL" envDefault:""`
	DbUser    string `env:"DBUSER" envDefault:""`
	DbPass    string `env:"DBPASS" envDefault:""`
	BoltFile  string `env:"BOLTFILE" envDefault:"cache.db"`

	// test specific variables
	TestChannel  string `env:"TESTCHANNEL" envDefault:"creation-test-"`
	MessageBreak int    `env:"MESSAGEBREAK" envDefault:"10"`
	LoginBreak   int    `env:"LOGINBREAK" envDefault:"1"`
	ReplyPercent int    `env:"REPLAYPERCENT" envDefault:"2"`
	MediaPercent int    `env:"MEDIAPERCENT" envDefault:"10"`
}
