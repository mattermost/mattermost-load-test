package loadtest

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/mattermost/mattermost-server/model"
	"github.com/pkg/errors"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"

	"github.com/mattermost/mattermost-load-test/cmdlog"
)

const (
	InstanceHeartbeatInterval = 30 * time.Second
	InstanceExpiredInterval   = InstanceHeartbeatInterval * 4
)

func createInstanceSchema(db *sql.DB) error {
	sql := `
	    CREATE TABLE IF NOT EXISTS LoadtestInstances(
		Id	    VARCHAR(26) PRIMARY KEY,
		CreateAt    BIGINT,
		ActiveAt    BIGINT,
		Idx	    INTEGER UNIQUE
	    )
`

	if _, err := db.Exec(sql); err != nil {
		return errors.Wrap(err, "failed to create instance schema")
	}

	return nil
}

func insertInstance(db *sql.DB, id string, now time.Time) (int, error) {
	for attempts := 1; attempts <= 5; attempts++ {
		var index int
		row := db.QueryRow(`SELECT COUNT(*) FROM LoadtestInstances`)
		if err := row.Scan(&index); err == sql.ErrNoRows {
			return 0, fmt.Errorf("failed to count instances")
		}

		sql := `
		    INSERT INTO LoadtestInstances
			(Id, CreateAt, ActiveAt, Idx) 
		    VALUES
			(?, ?, ?, ?)
    `
		if _, err := db.Exec(sql, id, now.Unix()*1000, now.Unix()*1000, index); err != nil {
			// Try again, on the off chance we tried to create an instance with the same index.
			cmdlog.Infof("failed to insert instance `%s` with index %d, trying again", id, index)
			time.Sleep(time.Duration(attempts) * time.Second)
		} else {
			return index, err
		}
	}

	return 0, fmt.Errorf("failed to insert instance `%s` with unique index", id)
}

func recordInstanceHeartbeat(db *sql.DB, id string, now time.Time) error {
	sql := `UPDATE LoadtestInstances SET ActiveAt = ? WHERE Id = ?`
	_, err := db.Exec(sql, now.Unix()*1000, id)
	return err
}

func pruneInstances(db *sql.DB, now time.Time) error {
	sql := `DELETE FROM LoadtestInstances WHERE ActiveAt <= ?`

	if result, err := db.Exec(sql, now.Add(-1*InstanceExpiredInterval).Unix()*1000); err != nil {
		return errors.Wrapf(err, "failed to prune instances")
	} else if count, _ := result.RowsAffected(); count > 0 {
		cmdlog.Infof("Pruned %d expired instances", count)
	}

	return nil
}

func deleteInstance(db *sql.DB, id string) error {
	sql := `DELETE FROM LoadtestInstances WHERE Id = ?`

	_, err := db.Exec(sql, id)
	return err
}

// Instance represents a running instance of a loadtest.
type Instance struct {
	Id             string
	Index          int
	EntityStartNum int

	db     *sql.DB
	close  chan bool
	closed chan bool
}

func NewInstance(db *sql.DB, numActiveEntities int) (*Instance, error) {
	if err := createInstanceSchema(db); err != nil {
		return nil, err
	}

	now := time.Now()

	if err := pruneInstances(db, now); err != nil {
		cmdlog.Errorf("failed to prune instances: %s", err.Error())
	}

	id := model.NewId()
	index, err := insertInstance(db, id, now)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to insert instance `%s`", id)
	}

	i := &Instance{
		Id:    id,
		Index: index,
		// TODO: Support variable number of configured entities per instance.
		EntityStartNum: index * numActiveEntities,

		db:     db,
		close:  make(chan bool),
		closed: make(chan bool),
	}
	go i.heartbeat()

	return i, nil
}

func (i *Instance) heartbeat() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	defer close(i.closed)

	for {

		select {
		case t := <-ticker.C:
			if err := recordInstanceHeartbeat(i.db, i.Id, t); err != nil {
				cmdlog.Infof("failed to record instance heartbeat for `%s` at `%s`: %s", i.Id, t, err.Error())
			}

		case <-i.close:
			return
		}
	}
}

func (i *Instance) Close() error {
	close(i.close)
	<-i.closed

	if err := deleteInstance(i.db, i.Id); err != nil {
		return errors.Wrapf(err, "failed to delete instance `%s`", i.Id)
	}

	return nil
}
