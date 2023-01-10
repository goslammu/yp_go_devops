package pgxstorage

import (
	"database/sql"
	"errors"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/goslammu/yp_go_devops/internal/pkg/metric"
)

var (
	errUnableToGetMigrations = errors.New("unable to get migrations, use standard database scheme")
)

var (
	migrationsPath = "./migrations/metrics"

	migration = `
	CREATE TABLE IF NOT EXISTS metrics (
		mname CHARACTER VARYING PRIMARY KEY,
		mtype CHARACTER VARYING,
		mval DOUBLE PRECISION,
		mdel BIGINT
	)`
)

const (
	stUpdateMetric = `
	INSERT INTO metrics
	VALUES ($1, $2, $3, $4) 
	ON CONFLICT (mname)
	DO
	UPDATE
	SET mtype = $2, mval = $3, mdel = metrics.mdel + $4`

	stGetMetric = `
	SELECT * 
	FROM metrics 
	WHERE mname = $1`

	stGetBatch = `
	SELECT * 
	FROM metrics`

	stDropTableIfExisis = `
	DROP TABLE IF EXISTS metrics`
)

// Realization of metrics storage based on Postgesql.
type pgxStorage struct {
	DB *sql.DB
}

// Constructor. Existing metrics table could be dropped on demand.
func New(dbAddr string, drop bool) (*pgxStorage, error) {
	ms := &pgxStorage{}

	DB, err := sql.Open("pgx", dbAddr)
	if err != nil {
		return nil, err
	}
	ms.DB = DB

	if drop {
		_, er := ms.DB.Exec(stDropTableIfExisis)
		if er != nil {
			return nil, er
		}
	}

	migrationFromFile, err := os.ReadFile(migrationsPath)
	if err != nil {
		log.Println(errUnableToGetMigrations)
	} else {
		migration = string(migrationFromFile)
	}

	_, err = ms.DB.Exec(migration)
	if err != nil {
		return nil, err
	}
	return ms, nil
}

// Needed to call this by defer after using Cconstructor.
func (st *pgxStorage) Close() error {
	return st.DB.Close()
}

// Checks if storage is initialized.
func (st *pgxStorage) AccessCheck() error {
	return st.DB.Ping()
}

// Returns existing metric by it's name.
func (st *pgxStorage) GetMetric(name string) (*metric.Metric, error) {
	rows, err := st.DB.Query(stGetMetric, name)
	if err != nil {
		return nil, err
	}
	defer func() {
		if errRowsClose := rows.Close(); errRowsClose != nil {
			log.Println(errRowsClose)
		}
	}()

	m := metric.Metric{}
	for rows.Next() {
		if err := rows.Scan(&m.ID, &m.MType, &m.Value, &m.Delta); err != nil {
			return nil, err
		}
	}
	if m.ID == "" {
		return nil, metric.ErrMetricDoesntExist
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &m, nil
}

// Returns all storaged metrics in slice.
func (st *pgxStorage) GetBatch() ([]*metric.Metric, error) {
	rows, err := st.DB.Query(stGetBatch)
	if err != nil {
		return nil, err
	}
	defer func() {
		if errRowsClose := rows.Close(); errRowsClose != nil {
			log.Println(errRowsClose)
		}
	}()

	allMetrics := []*metric.Metric{}
	for rows.Next() {
		m := metric.Metric{}
		if err := rows.Scan(&m.ID, &m.MType, &m.Value, &m.Delta); err != nil {
			return nil, err
		}
		allMetrics = append(allMetrics, &m)
	}
	if err := rows.Err(); err != nil {
		return nil, err

	}

	return allMetrics, nil
}

// Updates metric valuable fields: overrides Value and increments Delta.
func (st *pgxStorage) UpdateMetric(m *metric.Metric) error {
	if m == nil {
		return metric.ErrCannotUpdateInvalidFormat
	}

	if m.ID == "" {
		return metric.ErrCannotUpdateInvalidFormat
	}

	if _, err := st.DB.Exec(stUpdateMetric, m.ID, m.MType, m.Value, m.Delta); err != nil {
		return err
	}
	return nil
}

// Updates metrics collected in input batch by valuable fields: overrides Values and increments Deltas.
func (st *pgxStorage) UpdateBatch(batch []*metric.Metric) (err error) {
	tx, err := st.DB.Begin()
	if err != nil {
		return
	}

	defer func() {
		if err != nil {
			if errRollback := tx.Rollback(); errRollback != nil {
				log.Println(errRollback)
				return
			}
		}
	}()

	txStUpdateMetric, err := tx.Prepare(stUpdateMetric)
	if err != nil {
		return
	}

	for i := range batch {
		if batch[i] == nil {
			err = metric.ErrCannotUpdateInvalidFormat
			return
		}

		if batch[i].ID == "" {
			err = metric.ErrCannotUpdateInvalidFormat
			return
		}

		if _, err = txStUpdateMetric.Exec(batch[i].ID, batch[i].MType, batch[i].Value, batch[i].Delta); err != nil {
			return
		}
	}

	if errCommit := tx.Commit(); errCommit != nil {
		err = errCommit
		return
	}

	if errTxClose := txStUpdateMetric.Close(); errTxClose != nil {
		err = errTxClose
		return
	}

	return
}
