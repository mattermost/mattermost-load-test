package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/caarlos0/env"
	"github.com/mattermost/mattermost-load-test/lib"

	"github.com/boltdb/bolt"
)

var (
	// Config holds the parsed environemnt variables
	Config = EnvironmentalLoad{}
	// Info log
	Info *log.Logger
	// Warning log
	Warning *log.Logger
	// Error log
	Error *log.Logger
	// Cache Storage
	Cache *bolt.DB
)

func main() {
	parseErr := env.Parse(&Config)
	if parseErr != nil {
		fmt.Println("Could not parse environment variables", parseErr)
		os.Exit(1)
	}

	Cache, err := bolt.Open("my.db", 0600, nil)
	if err != nil {
		fmt.Println("Failed to open Cache file sessions.db", ":", err)
		os.Exit(1)
	}
	defer Cache.Close()

	file, err := os.OpenFile(Config.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println("Failed to open log file ", Config.LogPath, ":", err)
		os.Exit(1)
	}
	errfile, err := os.OpenFile(Config.ErrorPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println("Failed to open error file ", Config.ErrorPath, ":", err)
		os.Exit(1)
	}

	multi := io.MultiWriter(file, os.Stdout)
	errMutli := io.MultiWriter(errfile, os.Stderr)
	logInit(multi, errMutli)

	Info.Println("**********************************************")
	Info.Println("**********************************************")
	Info.Println("***************Load Test Started**************")
	Info.Println("**********************************************")
	Info.Println("**********************************************")

	defer func() {
		if r := recover(); r != nil {
			if Error != nil {
				Error.Printf("ERROR ON STARTED CAUGHT: %v", r)
			} else {
				fmt.Printf("ERROR ON STARTED CAUGHT: %v", r)
			}
			os.Exit(1)
		}
	}()

	registeredTest, ok := TypeRegistry[Config.TestPlan]
	if !ok {
		Error.Printf("Could not find Test Plan(%v) in TestRegistry", Config.TestPlan)
		os.Exit(1)
	}
	testPlanGenerator := registeredTest.Generator

	testID := GenerateUUID()
	gm := lib.GroupManager{TestID: testID}
	gm.Error = Error
	gm.Info = Info

	if Config.DbURL != "" {
		gm.InitDB(Config.DbURL, Config.DbUser, Config.DbPass)
	}

	Info.Printf("Test ID: %q", testID)
	Info.Printf("Development: %t", Config.Development)
	Info.Printf("Test %q ", Config.TestPlan)
	Info.Printf("API Server: %q", Config.PlatformURL)
	Info.Printf("Socket Server: %q", Config.SocketURL)
	Info.Printf("Database %q", Config.DbURL)
	Info.Printf("Starting %d threads in %d seconds", Config.Threads, Config.RampSec)
	Info.Printf("ID offset: %d", Config.ThreadOffset)
	Info.Printf("MessageBreak: %d, LoginBreak: %d, ReplyPercent %d", Config.MessageBreak, Config.LoginBreak, Config.ReplyPercent)

	if !Config.Development {
		// Require Confirmation
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Press [enter] to run the test (ctrl + c to exit)")
		reader.ReadString('\n')
	}

	gm.Start(testPlanGenerator, Config.Threads, Config.ThreadOffset, Config.RampSec)
}

func logInit(multiHandle io.Writer, errHandle io.Writer) {

	Info = log.New(multiHandle,
		"INFO: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Warning = log.New(multiHandle,
		"WARNING: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Error = log.New(errHandle,
		"ERROR: ",
		log.Ldate|log.Ltime|log.Lshortfile)
}

func SaveSession(key, value string) {
	Cache.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("sessions"))
		if err != nil {
			return err
		}
		return b.Put([]byte(key), []byte(value))
	})
}

func GetSession(key string) string {
	value := ""
	Cache.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("sessions"))
		if b != nil {
			value = string(b.Get([]byte(key)))
		}
		return nil
	})
	return value
}

func DeleteSession(key string) {
	Cache.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("sessions"))
		if err != nil {
			return err
		}
		return b.Delete([]byte(key))
	})
}
