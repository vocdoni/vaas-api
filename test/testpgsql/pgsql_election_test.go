package testpgsql

import (
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/google/uuid"
	"go.vocdoni.io/api/test/testcommon"
)

func TestElection(t *testing.T) {
	c := qt.New(t)
	integrators := testcommon.CreateIntegrators(1)
	var err error
	integrators[0].ID, err = API.DB.CreateIntegrator(integrators[0].SecretApiKey, integrators[0].CspPubKey,
		integrators[0].CspUrlPrefix, integrators[0].Name, integrators[0].Email)
	c.Assert(err, qt.IsNil)

	organizations := testcommon.CreateOrganizations(1)
	organizations[0].ID, err = API.DB.CreateOrganization(integrators[0].SecretApiKey, organizations[0].EthAddress,
		organizations[0].EthPrivKeyCicpher, organizations[0].QuotaPlanID, organizations[0].PublicAPIQuota,
		organizations[0].PublicAPIToken, organizations[0].HeaderURI, organizations[0].AvatarURI)
	c.Assert(err, qt.IsNil)

	elections := testcommon.CreateElections(2)
	id, err := API.DB.CreateElection(integrators[0].SecretApiKey, organizations[0].EthAddress, elections[0].ProcessID,
		elections[0].Title, elections[0].StartDate, elections[0].EndDate, uuid.NullUUID{}, 0, 0, true, true)
	c.Assert(err, qt.IsNil)
	c.Assert(int(id), qt.Not(qt.Equals), 0)
	elections[0].ID = id

	// integrator, err := API.DB.GetIntegrator(elections[0].ID)
	// log.Infof("%w", integrator)
	// c.Assert(err, qt.IsNil)
	// c.Assert(fmt.Sprintf("%x", integrator.SecretApiKey), qt.DeepEquals, fmt.Sprintf("%x", elections[0].SecretApiKey))

	// keys, err := API.DB.GetIntegratorApiKeysList()
	// log.Infof("%s", keys)
	// cleaning up
	for _, integrator := range integrators {
		if err := API.DB.DeleteIntegrator(integrator.ID); err != nil {
			t.Errorf("error deleting test integrator: %w", err)
		}

	}
}