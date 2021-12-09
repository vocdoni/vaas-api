package database

import (
	"time"

	"github.com/google/uuid"
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
	CreatePlan(name string, maxCensusSize, maxProcessCount int) (uuid.UUID, error)
	GetPlan(id uuid.UUID) (*types.QuotaPlan, error)
	GetPlanByName(name string) (*types.QuotaPlan, error)
	DeletePlan(id uuid.UUID) error
	UpdatePlan(id uuid.UUID, newMaxCensusSize, neWMaxProcessCount int, newName string) (int, error)
	GetPlansList() ([]types.QuotaPlan, error)
	// Organization
	CreateOrganization(integratorAPIKey, ethAddress, ethPrivKeyCipher []byte, planID uuid.NullUUID, publiApiQuota int, publicApiToken, headerUri, avatarUri string) (int, error)
	UpdateOrganization(integratorAPIKey, ethAddress []byte, planID uuid.NullUUID, apiQuota int, headerUri, avatarUri string) (int, error)
	UpdateOrganizationEthPrivKeyCipher(integratorAPIKey, ethAddress, newEthPrivKeyCicpher []byte) (int, error)
	UpdateOrganizationPublicAPIToken(integratorAPIKey, ethAddress []byte, newPublicApiToken string) (int, error)
	GetOrganization(integratorAPIKey, ethAddress []byte) (*types.Organization, error)
	DeleteOrganization(integratorAPIKey, ethAddress []byte) error
	ListOrganizations(integratorAPIKey []byte, filter *types.ListOptions) ([]types.Organization, error)
	CountOrganizations(integratorAPIKey []byte) (int, error)
	// Election
	CreateElection(integratorAPIKey, orgEthAddress, processID []byte, title string, startDate, endDate time.Time, censusID uuid.NullUUID, startBlock, endBlock int, confidential, hiddenResults bool) (int, error)
	GetElection(integratorAPIKey, orgEthAddress, processID []byte) (*types.Election, error)
	ListElections(integratorAPIKey, orgEthAddress []byte) ([]types.Election, error)
	// Manage DB
	Ping() error
	Close() error
	// Migrations
	Migrate(dir migrate.MigrationDirection) (int, error)
	MigrateStatus() (int, int, string, error)
	MigrationUpSync() (int, error)
}
