package database

import (
	"time"

	migrate "github.com/rubenv/sql-migrate"
	"go.vocdoni.io/api/types"
)

type Database interface {
	// Integrator
	CreateIntegrator(secretApiKey, cspPubKey []byte, cspUrlPrefix, name, email string) (int, error)
	UpdateIntegrator(id int, newCspPubKey []byte, newName, newCspUrlPrefix string) (int, error)
	UpdateIntegratorApiKey(id int, newSecretApiKey []byte) (int, error)
	GetIntegrator(id int) (*types.Integrator, error)
	GetIntegratorByKey(secretApiKey []byte) (*types.Integrator, error)
	DeleteIntegrator(id int) error
	CountIntegrators() (int, error)
	GetIntegratorApiKeysList() ([][]byte, error)
	// Plans
	CreatePlan(name string, maxCensusSize, maxProcessCount int) (int, error)
	GetPlan(id int) (*types.QuotaPlan, error)
	GetPlanByName(name string) (*types.QuotaPlan, error)
	DeletePlan(id int) error
	UpdatePlan(id, newMaxCensusSize, neWMaxProcessCount int, newName string) (int, error)
	GetPlansList() ([]types.QuotaPlan, error)
	// Organization
	CreateOrganization(integratorAPIKey, ethAddress, ethPrivKeyCipher []byte, planID, publiApiQuota int, publicApiToken, headerUri, avatarUri string) (int, error)
	UpdateOrganization(integratorAPIKey, ethAddress []byte, planID, apiQuota int, headerUri, avatarUri string) (int, error)
	UpdateOrganizationEthPrivKeyCipher(integratorAPIKey, ethAddress, newEthPrivKeyCicpher []byte) (int, error)
	UpdateOrganizationPublicAPIToken(integratorAPIKey, ethAddress []byte, newPublicApiToken string) (int, error)
	GetOrganization(integratorAPIKey, ethAddress []byte) (*types.Organization, error)
	DeleteOrganization(integratorAPIKey, ethAddress []byte) error
	ListOrganizations(integratorAPIKey []byte, filter *types.ListOptions) ([]types.Organization, error)
	CountOrganizations(integratorAPIKey []byte) (int, error)
	// Election
	CreateElection(integratorAPIKey, orgEthAddress, processID []byte, title string, startDate, endDate time.Time, censusID, startBlock, endBlock int, confidential, hiddenResults bool) (int32, error)
	GetElection(integratorAPIKey, orgEthAddress, processID []byte) (*types.Election, error)
	// Manage DB
	Ping() error
	Close() error
	// Migrations
	Migrate(dir migrate.MigrationDirection) (int, error)
	MigrateStatus() (int, int, string, error)
	MigrationUpSync() (int, error)
}
