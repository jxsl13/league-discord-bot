package migrations

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	nurl "net/url"
	"strconv"
	"strings"
	"sync/atomic"

	"context"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	_ "modernc.org/sqlite"
)

func init() {
	database.Register("sqlite", &sqliteDriver{})
}

var (
	DefaultMigrationsTable = "schema_migrations"
	ErrDatabaseDirty       = fmt.Errorf("database is dirty")
	ErrNilConfig           = fmt.Errorf("no config")
	ErrNoDatabaseName      = fmt.Errorf("no database name")
)

type config struct {
	Context         context.Context
	MigrationsTable string
	DatabaseName    string
	NoTxWrap        bool
}

type sqliteDriver struct {
	db       *sql.DB
	isLocked atomic.Bool

	config *config
}

func withInstance(instance *sql.DB, config *config) (database.Driver, error) {
	if config == nil {
		return nil, ErrNilConfig
	}

	if err := instance.Ping(); err != nil {
		return nil, err
	}

	if len(config.MigrationsTable) == 0 {
		config.MigrationsTable = DefaultMigrationsTable
	}

	mx := &sqliteDriver{
		db:     instance,
		config: config,
	}
	if err := mx.ensureVersionTable(); err != nil {
		return nil, err
	}
	return mx, nil
}

// ensureVersionTable checks if versions table exists and, if not, creates it.
// Note that this function locks the database, which deviates from the usual
// convention of "caller locks" in the sqliteDriver type.
func (m *sqliteDriver) ensureVersionTable() (err error) {
	if err = m.Lock(); err != nil {
		return err
	}

	defer func() {
		if e := m.Unlock(); e != nil {
			if err == nil {
				err = e
			} else {
				err = errors.Join(err, e)
			}
		}
	}()

	query := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s (version uint64,dirty bool);
  CREATE UNIQUE INDEX IF NOT EXISTS version_unique ON %s (version);
  `, m.config.MigrationsTable, m.config.MigrationsTable)

	if _, err := m.db.Exec(query); err != nil {
		return err
	}
	return nil
}

func (m *sqliteDriver) Open(url string) (database.Driver, error) {
	purl, err := nurl.Parse(url)
	if err != nil {
		return nil, err
	}
	dbfile := strings.Replace(migrate.FilterCustomQuery(purl).String(), "sqlite://", "", 1)
	db, err := sql.Open("sqlite", dbfile)
	if err != nil {
		return nil, err
	}

	qv := purl.Query()

	migrationsTable := qv.Get("x-migrations-table")
	if len(migrationsTable) == 0 {
		migrationsTable = DefaultMigrationsTable
	}

	noTxWrap := false
	if v := qv.Get("x-no-tx-wrap"); v != "" {
		noTxWrap, err = strconv.ParseBool(v)
		if err != nil {
			return nil, fmt.Errorf("x-no-tx-wrap: %s", err)
		}
	}

	mx, err := withInstance(db, &config{
		Context:         m.config.Context,
		DatabaseName:    purl.Path,
		MigrationsTable: migrationsTable,
		NoTxWrap:        noTxWrap,
	})
	if err != nil {
		return nil, err
	}
	return mx, nil
}

func (m *sqliteDriver) Close() error {
	return m.db.Close()
}

func (m *sqliteDriver) Drop() (err error) {
	query := `SELECT name FROM sqlite_master WHERE type = 'table';`
	tables, err := m.db.Query(query)
	if err != nil {
		return &database.Error{OrigErr: err, Query: []byte(query)}
	}
	defer func() {
		if errClose := tables.Close(); errClose != nil {
			err = errors.Join(err, errClose)
		}
	}()

	tableNames := make([]string, 0)
	for tables.Next() {
		var tableName string
		if err := tables.Scan(&tableName); err != nil {
			return err
		}
		if len(tableName) > 0 {
			tableNames = append(tableNames, tableName)
		}
	}
	if err := tables.Err(); err != nil {
		return &database.Error{OrigErr: err, Query: []byte(query)}
	}

	if len(tableNames) > 0 {
		for _, t := range tableNames {
			query := "DROP TABLE " + t
			err = m.executeQuery(query)
			if err != nil {
				return &database.Error{OrigErr: err, Query: []byte(query)}
			}
		}
		query := "VACUUM"
		_, err = m.db.Query(query)
		if err != nil {
			return &database.Error{OrigErr: err, Query: []byte(query)}
		}
	}

	return nil
}

func (m *sqliteDriver) Lock() error {
	if !m.isLocked.CompareAndSwap(false, true) {
		return database.ErrLocked
	}
	return nil
}

func (m *sqliteDriver) Unlock() error {
	if !m.isLocked.CompareAndSwap(true, false) {
		return database.ErrNotLocked
	}
	return nil
}

func (m *sqliteDriver) Run(migration io.Reader) error {
	migr, err := io.ReadAll(migration)
	if err != nil {
		return err
	}
	query := string(migr[:])

	if m.config.NoTxWrap {
		return m.executeQueryNoTx(query)
	}
	return m.executeQuery(query)
}

func (m *sqliteDriver) executeQuery(query string) error {
	tx, err := m.db.BeginTx(m.config.Context, nil)
	if err != nil {
		return &database.Error{OrigErr: err, Err: "transaction start failed"}
	}
	defer func() {
		err = errors.Join(err, tx.Rollback())
	}()
	_, err = tx.Exec(query)
	if err != nil {
		return &database.Error{OrigErr: err, Query: []byte(query)}
	}

	err = tx.Commit()
	if err != nil {
		return &database.Error{OrigErr: err, Err: "transaction commit failed"}
	}
	return nil
}

func (m *sqliteDriver) executeQueryNoTx(query string) error {
	_, err := m.db.Exec(query)
	if err != nil {
		return &database.Error{OrigErr: err, Query: []byte(query)}
	}
	return nil
}

func (m *sqliteDriver) SetVersion(version int, dirty bool) error {
	tx, err := m.db.BeginTx(m.config.Context, nil)
	if err != nil {
		return &database.Error{OrigErr: err, Err: "transaction start failed"}
	}
	defer func() {
		err = errors.Join(err, tx.Rollback())
	}()

	query := "DELETE FROM " + m.config.MigrationsTable
	if _, err := tx.Exec(query); err != nil {
		return &database.Error{OrigErr: err, Query: []byte(query)}
	}

	// Also re-write the schema version for nil dirty versions to prevent
	// empty schema version for failed down migration on the first migration
	// See: https://github.com/golang-migrate/migrate/issues/330
	if version >= 0 || (version == database.NilVersion && dirty) {
		query := fmt.Sprintf(`INSERT INTO %s (version, dirty) VALUES (?, ?)`, m.config.MigrationsTable)
		_, err = tx.Exec(query, version, dirty)
		if err != nil {
			return &database.Error{OrigErr: err, Query: []byte(query)}
		}
	}

	err = tx.Commit()
	if err != nil {
		return &database.Error{OrigErr: err, Err: "transaction commit failed"}
	}

	return nil
}

func (m *sqliteDriver) Version() (version int, dirty bool, err error) {
	query := "SELECT version, dirty FROM " + m.config.MigrationsTable + " LIMIT 1"
	err = m.db.QueryRow(query).Scan(&version, &dirty)
	if err != nil {
		return database.NilVersion, false, nil
	}
	return version, dirty, nil
}
