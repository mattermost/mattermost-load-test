package lib

import (
	"time"

	r "gopkg.in/dancannon/gorethink.v2"
)

const (
	samplesTable string = "samples"
	mmDatabase   string = "mattermost-load-test"
)

// Database container for rethinkdb logic and actions
type Database struct {
	url     string
	session *r.Session
}

// Checkin database structure
type Checkin struct {
	Time        time.Time `gorethink:"time"`
	ThreadCount int       `gorethink:"threadCount"`
	LaunchCount int       `gorethink:"launchCount"`
	ActiveCount int       `gorethink:"activeCount"`
	ActionCount int       `gorethink:"actionCount"`
	Errors      []string  `gorethink:"errors"`
	TestID      string    `gorethink:"testID"`
}

// CreateDBConnection to rethinkdb
func CreateDBConnection(url, user, password string) (Database, error) {
	var err error
	db := Database{url: url}
	session, err := r.Connect(r.ConnectOpts{
		Address:    url,
		InitialCap: 10,
		MaxOpen:    10,
		Database:   mmDatabase,
	})

	if err != nil {
		return db, err
	}

	db.session = session
	return db, nil
}

func (db *Database) writeCheckin(checkin Checkin) (string, error) {
	res, err := r.Table(samplesTable).Insert(checkin).RunWrite(db.session)
	if err != nil {
		return "", err
	}
	return res.GeneratedKeys[0], nil
}
