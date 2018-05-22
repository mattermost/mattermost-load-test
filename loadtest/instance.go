package loadtest

import (
	"database/sql"
	"fmt"
	"time"

	sqlx "github.com/jmoiron/sqlx"
	"github.com/mattermost/mattermost-server/model"
	"github.com/pkg/errors"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"

	"github.com/mattermost/mattermost-server/mlog"
)

const (
	InstanceHeartbeatInterval = 30 * time.Second
	InstanceExpiredInterval   = InstanceHeartbeatInterval * 4
)

func createInstanceSchema(db *sqlx.DB) error {
	query := `
	    CREATE TABLE IF NOT EXISTS LoadtestInstances(
		Id	    VARCHAR(26) PRIMARY KEY,
		CreateAt    BIGINT,
		ActiveAt    BIGINT,
		Idx	    INTEGER UNIQUE
	    )
`

	if _, err := db.Exec(query); err != nil {
		return errors.Wrap(err, "failed to create instance schema")
	}

	return nil
}

func insertInstance(db *sqlx.DB, id string, now time.Time) (int, error) {
	var err error

	for attempts := 1; attempts <= 5; attempts++ {
		var index sql.NullInt64
		row := db.QueryRow(`
		    SELECT 
			CASE 
			    WHEN li_lower.Idx IS NULL AND li.Idx > 0 THEN li.Idx - 1
			    WHEN li_higher.Idx IS NULL THEN li.Idx + 1
			    ELSE NULL
			END
		    FROM 
			LoadtestInstances li 
		    LEFT JOIN
			LoadtestInstances li_lower ON ( li_lower.Idx = li.Idx - 1 )
		    LEFT JOIN
			LoadtestInstances li_higher ON ( li_higher.Idx = li.Idx + 1 )
		    WHERE
			li.Idx > 0 AND li_lower.Id IS NULL
		     OR li_higher.Id IS NULL
		`)
		if err := row.Scan(&index); err != nil && err != sql.ErrNoRows {
			return 0, errors.Wrap(err, "failed to find available instance index")
		}

		query := `
		    INSERT INTO LoadtestInstances
			(Id, CreateAt, ActiveAt, Idx) 
		    VALUES
			(?, ?, ?, ?)
    `
		_, err = db.Exec(db.Rebind(query), id, now.Unix()*1000, now.Unix()*1000, index.Int64)
		if err != nil {
			// Try again, on the off chance we tried to create an instance with the same index.
			mlog.Info("failed to insert instance", mlog.String("instance_id", id), mlog.Int64("index", index.Int64))
			time.Sleep(time.Duration(attempts) * time.Second)
		} else {
			return int(index.Int64), nil
		}
	}

	return 0, fmt.Errorf("failed to insert instance `%s` with unique index: %s", id, err.Error())
}

func recordInstanceHeartbeat(db *sqlx.DB, id string, now time.Time) error {
	query := `UPDATE LoadtestInstances SET ActiveAt = ? WHERE Id = ?`
	_, err := db.Exec(db.Rebind(query), now.Unix()*1000, id)
	return err
}

func pruneInstances(db *sqlx.DB, now time.Time) error {
	query := `DELETE FROM LoadtestInstances WHERE ActiveAt <= ?`

	if result, err := db.Exec(db.Rebind(query), now.Add(-1*InstanceExpiredInterval).Unix()*1000); err != nil {
		return errors.Wrapf(err, "failed to prune instances")
	} else if count, _ := result.RowsAffected(); count > 0 {
		mlog.Info("Pruned expired instances", mlog.Int64("count", count))
	}

	return nil
}

func getCoordinatedRandomSeed(db *sqlx.DB) (int64, error) {
	var seed int64
	row := db.QueryRow(`SELECT li.CreateAt FROM LoadtestInstances li WHERE li.Idx = 0`)
	if err := row.Scan(&seed); err != nil {
		return 0, err
	}

	return seed, nil
}

func deleteInstance(db *sqlx.DB, id string) error {
	query := `DELETE FROM LoadtestInstances WHERE Id = ?`

	_, err := db.Exec(db.Rebind(query), id)
	return err
}

// Instance represents a running instance of a loadtest.
type Instance struct {
	Id             string
	Index          int
	EntityStartNum int
	Seed           int64

	db     *sqlx.DB
	close  chan bool
	closed chan bool
}

func NewInstance(db *sqlx.DB, numActiveEntities int) (*Instance, error) {
	if err := createInstanceSchema(db); err != nil {
		return nil, err
	}

	now := time.Now()

	if err := pruneInstances(db, now); err != nil {
		mlog.Error("failed to prune instances", mlog.Err(err))
	}

	id := model.NewId()
	index, err := insertInstance(db, id, now)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to insert instance `%s`", id)
	}

	// Attempt to arrive at a seed by which to coordinate randomness across loadtest instances.
	// Note that this is not resilient in the event that the loadtest with Index 0 restarts,
	// since its CreateAt time is used as the seed value. All loadtest instances should
	// be restarted in such a case to maintain coordination.
	seed, err := getCoordinatedRandomSeed(db)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for coordinated random seed")
	}

	i := &Instance{
		Id:    id,
		Index: index,
		// TODO: Support variable number of configured entities per instance.
		EntityStartNum: index * numActiveEntities,
		Seed:           seed,

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
				mlog.Info("failed to record instance heartbeat", mlog.String("instance_id", i.Id), mlog.Int64("time", t.Unix()), mlog.Err(err))
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
