package pgsql

import (
	"fmt"

	migrate "github.com/rubenv/sql-migrate"
	"go.vocdoni.io/api/database"
	"go.vocdoni.io/dvote/log"
)

// Migrations available
var Migrations = migrate.MemoryMigrationSource{
	Migrations: []*migrate.Migration{
		{
			Id:   "1",
			Up:   []string{migration1up},
			Down: []string{migration1down},
		},
	},
}

const migration1up = `
-- NOTES
-- 1. pgcrpyto is assumed to be enabled in public needing superuser access
--    CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;
-- 2. All columns are defined as NOT NULL to ease communication with Golang

CREATE EXTENSION IF NOT EXISTS pgcrypto SCHEMA public;

-- SQL in section 'Up' is executed when this migration is applied
--------------------------- TABLES DEFINITION
-------------------------------- -------------------------------- -------------------------------- 


--------------------------- Integrators
-- An Integrtor
CREATE TABLE integrators (
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    id bytea NOT NULL ,
    email text NOT NULL,
    name text NOT NULL,
);

ALTER TABLE ONLY integrators
    ADD CONSTRAINT integrators PRIMARY KEY (id);

ALTER TABLE ONLY integrators
    ADD CONSTRAINT integrators_email_unique UNIQUE (email);


--------------------------- ENTITTIES
-- An Entity/Organization

CREATE TABLE entities (
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    id bytea NOT NULL ,
    integrator_id  bytea NOT NULL,
    email text NOT NULL,
    name text NOT NULL,
);

ALTER TABLE ONLY entities
    ADD CONSTRAINT entities_pkey PRIMARY KEY (id);

ALTER TABLE ONLY entities
    ADD CONSTRAINT entities_address_unique UNIQUE (address);

`

const migration1down = `
DROP TABLE integrators;
DROP TABLE entities;
DROP EXTENSION IF EXISTS pgcrypto;
`

func Migrator(action string, db database.Database) error {
	switch action {
	case "upSync":
		log.Infof("checking if DB is up to date")
		mTotal, mApplied, _, err := db.MigrateStatus()
		if err != nil {
			return fmt.Errorf("could not retrieve migrations status: (%v)", err)
		}
		if mTotal > mApplied {
			log.Infof("applying missing %d migrations to DB", mTotal-mApplied)
			n, err := db.MigrationUpSync()
			if err != nil {
				return fmt.Errorf("could not apply necessary migrations (%v)", err)
			}
			if n != mTotal-mApplied {
				return fmt.Errorf("could not apply all necessary migrations (%v)", err)
			}
		} else if mTotal < mApplied {
			return fmt.Errorf("someting goes terribly wrong with the DB migrations")
		}
	case "up", "down":
		log.Info("applying migration")
		op := migrate.Up
		if action == "down" {
			op = migrate.Down
		}
		n, err := db.Migrate(op)
		if err != nil {
			return fmt.Errorf("error applying migration: (%v)", err)
		}
		if n != 1 {
			return fmt.Errorf("reported applied migrations !=1")
		}
		log.Infof("%q migration complete", action)
	case "status":
		break
	default:
		return fmt.Errorf("unknown migrate command")
	}

	total, actual, record, err := db.MigrateStatus()
	if err != nil {
		return fmt.Errorf("could not retrieve migrations status: (%v)", err)
	}
	log.Infof("Total Migrations: %d\nApplied migrations: %d (%s)", total, actual, record)
	return nil
}
