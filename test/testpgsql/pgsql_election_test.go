package testpgsql

import (
	"fmt"
	"math/rand"
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/google/uuid"
	"go.vocdoni.io/api/test/testcommon"
	"go.vocdoni.io/api/types"
)

func TestElection(t *testing.T) {
	t.Parallel()
	c := qt.New(t)
	integrators := testcommon.CreateIntegrators(1)
	var err error
	integrators[0].SecretApiKey = []byte(fmt.Sprintf("key%d", rand.Intn(10000)))
	integrators[0].ID, err = API.DB.CreateIntegrator(integrators[0].SecretApiKey, integrators[0].CspPubKey,
		integrators[0].CspUrlPrefix, integrators[0].Name, integrators[0].Email)
	c.Assert(err, qt.IsNil)

	organizations := testcommon.CreateDbOrganizations(1)
	organizations[0].ID, err = API.DB.CreateOrganization(integrators[0].SecretApiKey, organizations[0].EthAddress,
		organizations[0].EthPrivKeyCipher, organizations[0].QuotaPlanID, organizations[0].PublicAPIQuota,
		organizations[0].PublicAPIToken, organizations[0].HeaderURI, organizations[0].AvatarURI)
	c.Assert(err, qt.IsNil)

	elections := testcommon.CreateDbElections(t, 2)
	id, err := API.DB.CreateElection(integrators[0].SecretApiKey, organizations[0].EthAddress, elections[0].ProcessID,
		elections[0].MetadataPrivKey, elections[0].Title, string(types.PROOF_TYPE_BLIND), elections[0].StartDate,
		elections[0].EndDate, uuid.NullUUID{}, 0, 0, true, true)
	c.Assert(err, qt.IsNil)
	c.Assert(int(id), qt.Not(qt.Equals), 0)
	elections[0].ID = id

	election, err := API.DB.GetElectionPublic(organizations[0].EthAddress, elections[0].ProcessID)
	c.Assert(err, qt.IsNil)
	c.Assert(election.ID, qt.Not(qt.Equals), elections[0].ID)

	list, err := API.DB.ListElections(integrators[0].SecretApiKey, organizations[0].EthAddress)
	c.Assert(err, qt.IsNil)
	c.Assert(len(list), qt.Equals, 1)
	c.Assert(list[0].Title, qt.DeepEquals, elections[0].Title)
	// integrator, err := API.DB.GetIntegrator(elections[0].ID)
	// t.Logf("%w", integrator)
	// c.Assert(err, qt.IsNil)
	// c.Assert(fmt.Sprintf("%x", integrator.SecretApiKey), qt.DeepEquals, fmt.Sprintf("%x", elections[0].SecretApiKey))

	// keys, err := API.DB.GetIntegratorApiKeysList()
	// t.Logf("%s", keys)
	// cleaning up
	for _, integrator := range integrators {
		if err := API.DB.DeleteIntegrator(integrator.ID); err != nil {
			t.Errorf("error deleting test integrator: %w", err)
		}

	}
}
