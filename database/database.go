package database

import (
	migrate "github.com/rubenv/sql-migrate"
	"go.vocdoni.io/api/types"
)

type Database interface {
	// Entity
	CreateOrganization(integratorID, planID, apiQuota int, ethAddress, metadataPrivKey []byte, publicApiToken, headerUri, avatarUri string) (int32, error)
	UpdateOrganization(ethAddress []byte, planID, apiQuota int, headerUri, avatarUri string) (int, error)
	UpdateOrganizationMetadataPrivKey(id int, newMetadataPrivKey []byte) (int, error)
	UpdateOrganizationPublicToken(id int, newPublicApiToken string) (int, error)
	GetOrganization(integratorID int, ethAddress []byte) (*types.Organization, error)
	DeleteOrganization(integratorID int, ethAddress []byte) error
	ListOrganizations(integratorID int, filter *types.ListOptions) ([]types.Organization, error)
	CountOrganizations(integratorID int) (int, error)
	// Integrator
	CreateIntegrator(secretApiKey, cspPubKey []byte, name, cspUrlPrefix string) (int32, error)
	UpdateIntegrator(id int, newCspPubKey []byte, newName, newCspUrlPrefix string) (int, error)
	UpdateIntegratorApiKey(id int, newSecretApiKey []byte) (int, error)
	GetIntegrator(id int) (*types.Integrator, error)
	GetIntegratorByKey(secretApiKey []byte) (*types.Integrator, error)
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
