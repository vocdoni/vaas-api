package database

import (
	migrate "github.com/rubenv/sql-migrate"
	"go.vocdoni.io/api/types"
)

type Database interface {
	// Entity
	CreateEntity(integratorID, planID, apiQuota int, ethAddress, metadataPrivKey []byte, publicToken, headerUri, avatarUri string) (int32, error)
	UpdateEntity(ethAddress []byte, planID, apiQuota int, headerUri, avatarUri string) (int, error)
	UpdateEntityMetadataPrivKey(id int, newMetadataPrivKey []byte) (int, error)
	UpdateEntityPublicToken(id int, newPublicToken string) (int, error)
	GetEntity(integratorID int, ethAddress []byte) (*types.Entity, error)
	DeleteEntity(integratorID int, ethAddress []byte) error
	ListEntities(integratorID int, filter *types.ListOptions) ([]types.Entity, error)
	CountEntities(integratorID int) (int, error)
	// Integrator
	CreateIntegrator(secretApiKey, cspPubKey []byte, name, cspUrlPrefix string) (int32, error)
	UpdateIntegrator(secretApiKey, newCspPubKey []byte, newName, newCspUrlPrefix string) (int, error)
	UpdateIntegratorApiKey(secretApiKey, newSecretApiKey []byte) (int, error)
	GetIntegrator(secretApiKey []byte) (*types.Integrator, error)
	GetIntegratorByID(id int) (*types.Integrator, error)
	DeleteIntegrator(secretApiKey []byte) error
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
