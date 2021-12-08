package testpgsql

import (
	"fmt"
	"testing"

	qt "github.com/frankban/quicktest"
	"go.vocdoni.io/api/test/testcommon"
	"go.vocdoni.io/dvote/log"
)

func TestOrganization(t *testing.T) {
	c := qt.New(t)
	integrators := testcommon.CreateIntegrators(1)
	var err error
	integrators[0].ID, err = API.DB.CreateIntegrator(integrators[0].SecretApiKey, integrators[0].CspPubKey,
		integrators[0].CspUrlPrefix, integrators[0].Name, integrators[0].Email)
	c.Assert(err, qt.IsNil)

	organizations := testcommon.CreateOrganizations(1)
	id, err := API.DB.CreateOrganization(integrators[0].SecretApiKey, organizations[0].EthAddress,
		organizations[0].EthPrivKeyCicpher, organizations[0].QuotaPlanID, organizations[0].PublicAPIQuota,
		organizations[0].PublicAPIToken, organizations[0].HeaderURI, organizations[0].AvatarURI)
	c.Assert(err, qt.IsNil)
	c.Assert(int(id), qt.Not(qt.Equals), 0)
	organizations[0].ID = id

	organization, err := API.DB.GetOrganization(integrators[0].SecretApiKey, organizations[0].EthAddress)
	log.Infof("%w", organization)
	c.Assert(err, qt.IsNil)
	c.Assert(fmt.Sprintf("%x", organization.EthPrivKeyCicpher), qt.DeepEquals, fmt.Sprintf("%x", organizations[0].EthPrivKeyCicpher))

	count, err := API.DB.CountOrganizations(integrators[0].SecretApiKey)
	c.Assert(err, qt.IsNil)
	c.Assert(count, qt.Equals, 1)

	dbOrganizations, err := API.DB.ListOrganizations(integrators[0].SecretApiKey, nil)
	c.Assert(err, qt.IsNil)
	c.Assert(len(dbOrganizations), qt.Equals, 1)
	c.Assert(fmt.Sprintf("%x", dbOrganizations[0].ID), qt.DeepEquals, fmt.Sprintf("%x", organizations[0].ID))

	// cleaning up (cascade delete from integrators)
	for _, integrator := range integrators {
		if err := API.DB.DeleteIntegrator(integrator.ID); err != nil {
			t.Errorf("error deleting test entity: %w", err)
		}

	}
}
