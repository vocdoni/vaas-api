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
-- 21 All columns are defined as NOT NULL to ease communication with Golang

CREATE EXTENSION IF NOT EXISTS pgcrypto SCHEMA public;

-- SQL in section 'Up' is executed when this migration is applied
--------------------------- TABLES DEFINITION
-------------------------------- -------------------------------- -------------------------------- 


--------------------------- Integrators
-- An Integrtor
CREATE TABLE integrators (
    updated_at timestamp without time zone DEFAULT (now() at time zone 'utc') NOT NULL,
    created_at timestamp without time zone DEFAULT (now() at time zone 'utc') NOT NULL,
    secret_api_key BYTEA NOT NULL,
    id SERIAL NOT NULL,
    name TEXT NOT NULL,
    email TEXT NOT NULL,
    csp_url_prefix TEXT NOT NULL,
    csp_pub_key TEXT NOT NULL
);

ALTER TABLE ONLY integrators
    ADD CONSTRAINT integrators PRIMARY KEY (id);

ALTER TABLE ONLY integrators
    ADD CONSTRAINT integrators_email_unique UNIQUE (email);

ALTER TABLE ONLY integrators
ADD CONSTRAINT integrators_secret_api_key_unique UNIQUE (secret_api_key);

--------------------------- ENTITTIES
-- An Organization

CREATE TABLE organizations (
    updated_at timestamp without time zone DEFAULT (now() at time zone 'utc') NOT NULL,
    created_at timestamp without time zone DEFAULT (now() at time zone 'utc') NOT NULL,
    id SERIAL NOT NULL ,
    integrator_id  INTEGER NOT NULL,
    integrator_api_key BYTEA NOT NULL,
    name TEXT NOT NULL,
    eth_address BYTEA NOT NULL,
    eth_priv_key_cipher BYTEA NOT NULL,
    header_uri TEXT NOT NULL,
    avatar_uri TEXT NOT NULL,
    public_api_token  TEXT NOT NULL,
    quota_plan_id INTEGER NOT NULL,
    public_api_quota INTEGER NOT NULL
);

ALTER TABLE ONLY organizations
    ADD CONSTRAINT organizations_pkey PRIMARY KEY (integrator_id, eth_address);

ALTER TABLE ONLY organizations
    ADD CONSTRAINT organizations_address_unique UNIQUE (address);

ALTER TABLE ONLY organizations
    ADD CONSTRAINT organizations_integrator_id_fkey FOREIGN KEY (integrator_id) REFERENCES integrators(id) ON DELETE CASCADE;

ALTER TABLE ONLY organizations
    ADD CONSTRAINT organizations_integrator_api_key_fkey FOREIGN KEY (integrator_api_key) REFERENCES integrators(secret_api_key) ON UPDATE CASCADE;

ALTER TABLE ONLY organizations
    ADD CONSTRAINT organizations_quota_plan_id_fkey FOREIGN KEY (quota_plan_id) REFERENCES quota_plans(id);

ALTER TABLE ONLY organizations
    ADD CONSTRAINT organizations_integrator_id_name_unique UNIQUE (integrator_id, name);

--------------------------- Election
-- Election processes

CREATE TABLE elections (
    updated_at timestamp without time zone DEFAULT (now() at time zone 'utc') NOT NULL,
    created_at timestamp without time zone DEFAULT (now() at time zone 'utc') NOT NULL,
    id SERIAL NOT NULL ,
    organization_eth_address  BYTEA NOT NULL,
    integrator_api_key BYTEA NOT NULL,
    process_id BYTEA NOT NULL,
    title TEXT NOT NULL,
    census_id INTEGER NOT NULL,
    start_block BIGINT NOT NULL,
    end_block BIGINT NOT NULL,
    confidential  BOOLEAN DEFAULT false NOT NULL,
    hidden_results  BOOLEAN DEFAULT false NOT NULL
    -- if needed the following field should be activated
    -- metadata_priv_key BYTEA NOT NULL 
);

ALTER TABLE ONLY elections
    ADD CONSTRAINT elections_pkey PRIMARY KEY (process_id);

ALTER TABLE ONLY elections
    ADD CONSTRAINT elections_organization_eth_address_fkey FOREIGN KEY (organization_eth_address) REFERENCES organizations(eth_address) ON DELETE CASCADE;

ALTER TABLE ONLY elections
    ADD CONSTRAINT elections_integrator_api_key_fkey FOREIGN KEY (integrator_api_key) REFERENCES integrators(secret_api_key) ON UPDATE CASCADE;

ALTER TABLE ONLY elections
    ADD CONSTRAINT elections_census_id_fkey FOREIGN KEY (census_id) REFERENCES censuses(id);


--------------------------- Census
-- Censuses as defined by an integrator

CREATE TABLE censuses (
    updated_at timestamp without time zone DEFAULT (now() at time zone 'utc') NOT NULL,
    created_at timestamp without time zone DEFAULT (now() at time zone 'utc') NOT NULL,
    id SERIAL NOT NULL,
    organization_id  INTEGER NOT NULL,
    name TEXT NOT NULL
);

ALTER TABLE ONLY censuses
    ADD CONSTRAINT censuses_pkey PRIMARY KEY (id);

ALTER TABLE ONLY censuses
    ADD CONSTRAINT censuses_organization_id_fkey FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE;

--------------------------- Census Member
-- Census members

CREATE TABLE census_members (
    updated_at timestamp without time zone DEFAULT (now() at time zone 'utc') NOT NULL,
    created_at timestamp without time zone DEFAULT (now() at time zone 'utc') NOT NULL,
    id SERIAL NOT NULL,
    census_id  INTEGER NOT NULL,
    public_key BYTEA NOT NULL,
    redeem_token TEXT NOT NULL,
    weight INTEGER NOT NULL DEFAULT 1,
);

ALTER TABLE ONLY census_members
    ADD CONSTRAINT census_members_pkey PRIMARY KEY (census_id, public_key);

ALTER TABLE ONLY census_members
    ADD CONSTRAINT census_members_census_id_fkey FOREIGN KEY (census_id) REFERENCES census(id)  ON DELETE CASCADE;


--------------------------- Billing Plans
-- Billing plans

CREATE TABLE quota_plans (
    updated_at timestamp without time zone DEFAULT (now() at time zone 'utc') NOT NULL,
    created_at timestamp without time zone DEFAULT (now() at time zone 'utc') NOT NULL,
    id SERIAL NOT NULL,
    name TEXT NOT NULL,
    max_census_size INTEGER NOT NULL,
    max_process_count INTEGER NOT NULL
);

ALTER TABLE ONLY census_members
    ADD CONSTRAINT census_members_pkey PRIMARY KEY (id);

`

const migration1down = `
DROP TABLE integrators;
DROP TABLE organizations;
DROP TABLE elections;
DROP TABLE censuses;
DROP TABLE census_members;
DROP TABLE quota_plans;
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
