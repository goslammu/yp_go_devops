package pgxstorage

import (
	"database/sql"
	"log"

	"github.com/dcaiman/YP_GO/internal/pkg/metric"
)

const (
	tabname = `metrics`

	schema = `
	(
		mname CHARACTER VARYING PRIMARY KEY,
		mtype CHARACTER VARYING,
		mval DOUBLE PRECISION,
		mdel BIGINT
	)`

	stUpdateMetric = `
	INSERT INTO ` + tabname + `
	VALUES ($1, $2, $3, $4) 
	ON CONFLICT (mname)
	DO
	UPDATE
	SET mtype = $2, mval = $3, mdel = metrics.mdel + $4`

	stGetMetric = `
	SELECT * 
	FROM ` + tabname + ` 
	WHERE mname = $1`

	stGetBatch = `
	SELECT * 
	FROM ` + tabname

	stCreateTableIfNotExists = `
	CREATE TABLE IF NOT EXISTS ` + tabname + ` ` + schema

	stDropTableIfExisis = `
	DROP TABLE IF EXISTS ` + tabname
)

// Realization of metrics storage based on Postgesql.
type MetricStorage struct {
	DB *sql.DB
}

// Constructor. Existing metrics table could be dropped on demand.
func New(dbAddr string, drop bool) (*MetricStorage, error) {
	ms := &MetricStorage{}

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
	_, err = ms.DB.Exec(stCreateTableIfNotExists)
	if err != nil {
		return nil, err
	}
	return ms, nil
}

// Needed to call this by defer after using Cconstructor.
func (st *MetricStorage) Close() error {
	return st.DB.Close()
}

// Checks if storage is initialized.
func (st *MetricStorage) AccessCheck() error {
	return st.DB.Ping()
}

// Returns existing metric by it's name.
func (st *MetricStorage) GetMetric(name string) (*metric.Metric, error) {
	rows, err := st.DB.Query(stGetMetric, name)
	if err != nil {
		return nil, err
	}

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

	if er := rows.Close(); er != nil {
		return nil, er
	}

	return &m, nil
}

// Returns all storaged metrics in slice.
func (st *MetricStorage) GetBatch() ([]*metric.Metric, error) {
	rows, err := st.DB.Query(stGetBatch)
	if err != nil {
		return nil, err
	}

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

	if er := rows.Close(); er != nil {
		return nil, er
	}

	return allMetrics, nil
}

// Updates metric valuable fields: overrides Value and increments Delta.
func (st *MetricStorage) UpdateMetric(m *metric.Metric) error {
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
func (st *MetricStorage) UpdateBatch(batch []*metric.Metric) error {
	tx, err := st.DB.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if er := tx.Rollback(); er != nil {
			log.Println(er.Error())
			return
		}

	}()

	txStUpdateMetric, err := tx.Prepare(stUpdateMetric)
	if err != nil {
		return err
	}

	for i := range batch {
		if batch[i] == nil {
			return metric.ErrCannotUpdateInvalidFormat
		}

		if batch[i].ID == "" {
			return metric.ErrCannotUpdateInvalidFormat
		}

		if _, err := txStUpdateMetric.Exec(batch[i].ID, batch[i].MType, batch[i].Value, batch[i].Delta); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	if er := txStUpdateMetric.Close(); er != nil {
		return er
	}

	return nil
}
