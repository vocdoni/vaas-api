package pgsql

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/jmoiron/sqlx"
	migrate "github.com/rubenv/sql-migrate"

	_ "github.com/jackc/pgx/stdlib"
	"go.vocdoni.io/dvote/log"

	"go.vocdoni.io/api/config"
)

const connectionRetries = 5

type Database struct {
	db *sqlx.DB
	// For using pgx connector
	// pgx    *pgxpool.Pool
	// pgxCtx context.Context
}

// New creates a new postgres SQL database connection
func New(dbc *config.DB) (*Database, error) {
	db, err := sqlx.Open("pgx", fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s client_encoding=%s",
		dbc.Host, dbc.Port, dbc.User, dbc.Password, dbc.Dbname, dbc.Sslmode, "UTF8"))
	if err != nil {
		return nil, fmt.Errorf("error initializing postgres connection handler: %w", err)
	}

	// Try to get a connection, if fails connectionRetries times, return error.
	// This is necessary for ensuting the database connection is alive before going forward.
	for i := 0; i < connectionRetries; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		log.Infof("trying to connect to postgres")
		if _, err = db.Conn(ctx); err == nil {
			break
		}
		log.Warnf("database connection error (%s), retrying...", err)
		time.Sleep(time.Second * 2)
	}
	if err != nil {
		return nil, err
	}
	log.Info("connected to the database")

	// For using pgx connector
	// ctx := context.Background()
	// pgx, err := pgxpool.Connect(ctx, connectionString)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}
	// TODO: Set MaxOpenConnections, MaxLifetime (MaxIdle?)
	// MaxOpen should be the number of expected clients? (Different apis?)
	// db.SetMaxOpenConns(2)

	return &Database{db: db}, err
}

func (d *Database) Close() error {
	defer d.db.Close()
	// defer d.pgx.Close()
	return nil
	// return d.db.Close()
}

func bulkInsert(tx *sqlx.Tx, bulkQuery string, bulkData interface{}, numField int) error {
	// This function allows to solve the postgresql limit of max 65535 parameters in a query
	// The number of placeholders allowed in a query is capped at 2^16, therefore,
	// divide 2^16 by the number of fields in the struct, and that is the max
	// number of bulk inserts possible. Use that number to chunk the inserts.
	maxBulkInsert := ((1 << 16) / numField) - 1

	s := reflect.ValueOf(bulkData)
	if s.Kind() != reflect.Slice {
		return fmt.Errorf(" given a non-slice type")
	}
	bulkSlice := make([]interface{}, s.Len())
	for i := 0; i < s.Len(); i++ {
		bulkSlice[i] = s.Index(i).Interface()
	}

	// send batch requests
	for i := 0; i < len(bulkSlice); i += maxBulkInsert {
		// set limit to i + chunk size or to max
		limit := i + maxBulkInsert
		if len(bulkSlice) < limit {
			limit = len(bulkSlice)
		}
		batch := bulkSlice[i:limit]
		result, err := tx.NamedExec(bulkQuery, batch)
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				return fmt.Errorf("something is very wrong: could not rollback performing batch insert %s %w", bulkQuery, rollbackErr)
			}
			return fmt.Errorf("error during batch insert %s %w", bulkQuery, err)
		}
		addedRows, err := result.RowsAffected()
		if err != nil {
			if rollErr := tx.Rollback(); err != nil {
				return fmt.Errorf("something is very wrong: could not rollback performing batch insert: %v after error on counting affected rows: %w", rollErr, err)
			}
			return fmt.Errorf("could not verify updated rows: %w", err)
		}
		if addedRows != int64(len(batch)) {
			if rollErr := tx.Rollback(); err != nil {
				return fmt.Errorf("something is very wrong: error rolling back: %v expected to have inserted %d rows but inserted %d", rollErr, addedRows, len(batch))
			}
			return fmt.Errorf("expected to have inserted %d rows but inserted %d", addedRows, len(batch))
		}

	}
	return nil
}

func (d *Database) Ping() error {
	return d.db.Ping()
}

// Migrate performs a concrete migration (up or down)
func (d *Database) Migrate(dir migrate.MigrationDirection) (int, error) {
	n, err := migrate.ExecMax(d.db.DB, "postgres", Migrations, dir, 1)
	if err != nil {
		return 0, fmt.Errorf("failed migration: %w", err)
	}
	return n, nil
}

// Migrate returns the total and applied number of migrations,
// as well a string describing the perform migrations
func (d *Database) MigrateStatus() (int, int, string, error) {
	total, err := Migrations.FindMigrations()
	if err != nil {
		return 0, 0, "", fmt.Errorf("cannot retrieve total migrations status: %w", err)
	}
	record, err := migrate.GetMigrationRecords(d.db.DB, "postgres")
	if err != nil {
		return len(total), 0, "", fmt.Errorf("cannot  retrieve applied migrations status: %w", err)
	}
	recordB, err := json.Marshal(record)
	if err != nil {
		return len(total), len(record), "", fmt.Errorf("failed to parse migration status: %w", err)
	}
	return len(total), len(record), string(recordB), nil
}

// MigrationUpSync performs the missing up migrations in order to reach to highest migration
// available in migrations.go
func (d *Database) MigrationUpSync() (int, error) {
	n, err := migrate.ExecMax(d.db.DB, "postgres", Migrations, migrate.Up, 0)
	if err != nil {
		return 0, fmt.Errorf("cannot  perform missing migrations: %w", err)
	}
	return n, nil
}
