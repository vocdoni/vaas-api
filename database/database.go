package database

import (
	migrate "github.com/rubenv/sql-migrate"
	"go.vocdoni.io/api/types"
)

type Database interface {
	// Entity
	CreateOrganization(integratorAPIKey, ethAddress, ethPrivKeyCipher []byte, planID, publiApiQuota int, publicApiToken, headerUri, avatarUri string) (int32, error)
	UpdateOrganization(integratorAPIKey, ethAddress []byte, planID, apiQuota int, headerUri, avatarUri string) (int, error)
	UpdateOrganizationEthPrivKeyCipher(integratorAPIKey, ethAddress, newEthPrivKeyCicpher []byte) (int, error)
	UpdateOrganizationPublicAPIToken(integratorAPIKey, ethAddress []byte, newPublicApiToken string) (int, error)
	GetOrganization(integratorAPIKey, ethAddress []byte) (*types.Organization, error)
	DeleteOrganization(integratorAPIKey, ethAddress []byte) error
	ListOrganizations(integratorAPIKey []byte, filter *types.ListOptions) ([]types.Organization, error)
	CountOrganizations(integratorAPIKey []byte) (int, error)
	// Integrator
	CreateIntegrator(secretApiKey, cspPubKey []byte, cspUrlPrefix, name string) (int32, error)
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
