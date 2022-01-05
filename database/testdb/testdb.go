package testdb

import (
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	migrate "github.com/rubenv/sql-migrate"
	"go.vocdoni.io/api/types"
)

var Signers = []struct {
	Pub  string
	Priv string
}{
	// User() no rows
	{"03f9d1e41906436bf2e8aab319383dea6a4c06426266955293fd92b41f6346256f", "1c1c5c24be0d76e5f7c853902e9e23ced013a597aca7573861c8cd0a160ca375"},
	// user no rows failed AddUser()
	{"03733ca0d2462ef3cd4dbd331d5ec27a63eeb13afbaf03f236847479c3e8d7fd94", "1c1c5c24be0d76e5f7c853902e9e23ced013a597aca7573861c8cd0a160ca355"},
	// failed User()
	{"0399d0ad8447520e66df7db954b0936f4b141a01ba6213dda88c9df7293b66262e", "1c1c5c24be0d76e5f7c853902e9e23ced013a597aca7573861c8cd0a160ca357"},
	// MemberPubKey() no rows
	{"026163a9bc3425426bbb7f0fde6c9bb4504493415a34b99a84162fe01640a784a3", "1c1c5c24be0d76e5f7c853902e9e23ced013a597aca7573861c8cd0a160ca372"},
}

type Database struct {
}

func New() (*Database, error) {
	return &Database{}, nil
}

func (d *Database) CreateIntegrator(secretApiKey, cspPubKey []byte, cspUrlPrefix, name, email string) (int, error) {
	return 1, nil
}

func (d *Database) UpdateIntegrator(id int, newCspPubKey []byte, newCspUrlPrefix, newName string) (int, error) {
	return 1, nil
}

func (d *Database) UpdateIntegratorApiKey(id int, newSecretApiKey []byte) (int, error) {
	return 1, nil

}

func (d *Database) GetIntegrator(id int) (*types.Integrator, error) {
	return nil, nil
}

func (d *Database) GetIntegratorByKey(secretApiKey []byte) (*types.Integrator, error) {
	return nil, nil

}

func (d *Database) DeleteIntegrator(id int) error {
	return nil
}

func (d *Database) CountIntegrators() (int, error) {
	return 1, nil

}

func (d *Database) GetIntegratorApiKeysList() ([][]byte, error) {
	return nil, nil
}

func (d *Database) CreatePlan(name string, maxCensusSize, maxProcessCount int) (uuid.UUID, error) {
	return uuid.UUID{}, nil
}

func (d *Database) GetPlan(id uuid.UUID) (*types.QuotaPlan, error) {
	return nil, nil
}

func (d *Database) GetPlanByName(name string) (*types.QuotaPlan, error) {
	return nil, nil
}

func (d *Database) DeletePlan(id uuid.UUID) error {
	return nil

}

func (d *Database) UpdatePlan(id uuid.UUID, newMaxCensusSize, neWMaxProcessCount int, newName string) (int, error) {
	return 1, nil
}

func (d *Database) GetPlansList() ([]types.QuotaPlan, error) {
	return nil, nil
}

func (d *Database) CreateOrganization(integratorAPIKey, ethAddress, ethPrivKeyCipher []byte, planID uuid.NullUUID, publiApiQuota int, publicApiToken, headerUri, avatarUri string) (int, error) {
	return 1, nil
}

func (d *Database) UpdateOrganization(integratorAPIKey, ethAddress []byte, headerUri, avatarUri string) (int, error) {
	return 1, nil
}

func (d *Database) UpdateOrganizationPlan(integratorAPIKey, ethAddress []byte, planID uuid.NullUUID, apiQuota int) (int, error) {
	return 1, nil
}

func (d *Database) UpdateOrganizationEthPrivKeyCipher(integratorAPIKey, ethAddress, newEthPrivKeyCipher []byte) (int, error) {
	return 1, nil
}

func (d *Database) UpdateOrganizationPublicAPIToken(integratorAPIKey, ethAddress []byte, newPublicApiToken string) (int, error) {
	return 1, nil
}

func (d *Database) GetOrganization(integratorAPIKey, ethAddress []byte) (*types.Organization, error) {
	return nil, nil
}

func (d *Database) DeleteOrganization(integratorAPIKey, ethAddress []byte) error {
	return nil

}

func (d *Database) ListOrganizations(integratorAPIKey []byte, filter *types.ListOptions) ([]types.Organization, error) {
	return nil, nil
}

func (d *Database) CountOrganizations(integratorAPIKey []byte) (int, error) {
	return 1, nil
}

func (d *Database) CreateElection(integratorAPIKey, orgEthAddress, processID, encryptedMetadataKey []byte, title string, startDate, endDate time.Time, censusID uuid.NullUUID, startBlock, endBlock int, confidential, hiddenResults bool) (int, error) {
	return 1, nil
}

func (d *Database) GetElection(integratorAPIKey, orgEthAddress, processID []byte) (*types.Election, error) {
	return nil, nil
}

func (d *Database) GetElectionPublic(organizationEthAddress, processID []byte) (*types.Election, error) {
	return nil, nil
}

func (d *Database) GetElectionPrivate(organizationEthAddress, processID []byte) (*types.Election, error) {
	return nil, nil
}

func (d *Database) ListElections(integratorAPIKey, orgEthAddress []byte) ([]types.Election, error) {
	return nil, nil
}

func (d *Database) Ping() error {
	return nil
}

func (d *Database) Close() error {
	return nil
}

func (d *Database) Migrate(dir migrate.MigrationDirection) (int, error) {
	return 1, nil
}

func (d *Database) MigrateStatus() (int, int, string, error) {
	return 1, 0, "", nil
}

func (d *Database) MigrationUpSync() (int, error) {
	return 1, nil
}
