package database

import (
	migrate "github.com/rubenv/sql-migrate"
	"go.vocdoni.io/api/types"
)

type Database interface {
	// Entity
	CreateEntity(integratorID, planID, apiQuota int, ethAddress, metadataPrivKey []byte, publicToken, headerUri, avatarUri string) (int, error)
	UpdateEntity(id int, planID, apiQuota int, ethAddress []byte, headerUri, avatarUri string) (int, error)
	UpdateEntityMetadataPrivKey(id int, metadataPrivKey []byte) (int, error)
	UpdateEntityPublicToken(id int, publicToken string) (int, error)
	GetEntity(id int, entityID []byte) (*types.Entity, error)
	DeleteEntity(id int, entityID []byte) error
	ListEntities(id int, filter *types.ListOptions) ([]types.Entity, error)
	CountEntities(integratorID int) (int, error)
	// Integrator
	CreateIntegrator(secretApiKey, cspPubKey []byte, name, cspUrlPrefix string) (int, error)
	UpdateIntegrator(id int, cspPubKey []byte, name, cspUrlPrefix string) (int, error)
	UpdateIntegratorApiKey(id int, secretApiKey []byte) (int, error)
	GetIntegrator(id int) (*types.Integrator, error)
	DeleteIntegrator(id int) error
	CountIntegrators() (int, error)
	//
	// Manage DB
	Ping() error
	Close() error
	// Migrations
	Migrate(dir migrate.MigrationDirection) (int, error)
	MigrateStatus() (int, int, string, error)
	MigrationUpSync() (int, error)
}
