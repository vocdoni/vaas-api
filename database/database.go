package database

import (
	migrate "github.com/rubenv/sql-migrate"
	"go.vocdoni.io/api/types"
)

type Database interface {
	// Entity
	CreateEntity(integratorID, entityID []byte, info *types.EntityInfo) error
	UpdateEntity(integratorID, entityID []byte, info *types.EntityInfo) (int, error)
	GetEntity(integratorID, entityID []byte) (*types.Entity, error)
	DeleteEntity(integratorID, entityID []byte) error
	AuthorizeEntity(integratorID, entityID []byte) error
	ListEntities(integratorID []byte, filter *types.ListOptions) ([]types.Entity, error)
	CountEntities(integratorID []byte) (int, error)
	// Integrator
	CreateIntegrator(integratorID []byte, info *types.IntegratorInfo) error
	UpdateIntegrator(entityID []byte, info *types.IntegratorInfo) (int, error)
	GetIntegrator(integratorID []byte) (*types.Integrator, error)
	DeleteIntegrator(entityID []byte) error
	AuthorizeIntegrator(integratorID []byte) error
	CountIntegrators(integratorID []byte) (int, error)
	// Manage DB
	Ping() error
	Close() error
	// Migrations
	Migrate(dir migrate.MigrationDirection) (int, error)
	MigrateStatus() (int, int, string, error)
	MigrationUpSync() (int, error)
}
