package testpgsql

import (
	"encoding/hex"
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

	organizations := testcommon.CreateDbOrganizations(1)
	id, err := API.DB.CreateOrganization(integrators[0].SecretApiKey, organizations[0].EthAddress,
		organizations[0].EthPrivKeyCipher, organizations[0].QuotaPlanID, organizations[0].PublicAPIQuota,
		organizations[0].PublicAPIToken, organizations[0].HeaderURI, organizations[0].AvatarURI)
	c.Assert(err, qt.IsNil)
	c.Assert(int(id), qt.Not(qt.Equals), 0)
	organizations[0].ID = id

	organization, err := API.DB.GetOrganization(integrators[0].SecretApiKey, organizations[0].EthAddress)
	log.Infof("%w", organization)
	c.Assert(err, qt.IsNil)
	c.Assert(fmt.Sprintf("%x", organization.EthAddress), qt.Equals, fmt.Sprintf("%x", organizations[0].EthAddress))
	c.Assert(fmt.Sprintf("%x", organization.EthPrivKeyCipher), qt.Equals, fmt.Sprintf("%x", organizations[0].EthPrivKeyCipher))
	c.Assert(organization.QuotaPlanID, qt.Equals, organizations[0].QuotaPlanID)
	c.Assert(organization.PublicAPIQuota, qt.Equals, organizations[0].PublicAPIQuota)
	c.Assert(organization.PublicAPIToken, qt.Equals, organizations[0].PublicAPIToken)
	c.Assert(organization.HeaderURI, qt.Equals, organizations[0].HeaderURI)
	c.Assert(organization.AvatarURI, qt.Equals, organizations[0].AvatarURI)

	count, err := API.DB.UpdateOrganization(integrators[0].SecretApiKey, organizations[0].EthAddress, "header", "avatar")
	c.Assert(err, qt.IsNil)
	c.Assert(count, qt.Equals, 1)

	count, err = API.DB.CountOrganizations(integrators[0].SecretApiKey)
	c.Assert(err, qt.IsNil)
	c.Assert(count, qt.Equals, 1)

	dbOrganizations, err := API.DB.ListOrganizations(integrators[0].SecretApiKey, nil)
	c.Assert(err, qt.IsNil)
	c.Assert(len(dbOrganizations), qt.Equals, 1)

	// cleaning up (cascade delete from integrators)
	for _, integrator := range integrators {
		if err := API.DB.DeleteIntegrator(integrator.ID); err != nil {
			t.Errorf("error deleting test entity: %w", err)
		}

	}
}

func TestOrganizationUpdate(t *testing.T) {
	c := qt.New(t)
	integrators := testcommon.CreateIntegrators(1)
	var err error
	integrators[0].ID, err = API.DB.CreateIntegrator(integrators[0].SecretApiKey, integrators[0].CspPubKey,
		integrators[0].CspUrlPrefix, integrators[0].Name, integrators[0].Email)
	c.Assert(err, qt.IsNil)

	organizations := testcommon.CreateDbOrganizations(1)
	id, err := API.DB.CreateOrganization(integrators[0].SecretApiKey, organizations[0].EthAddress,
		organizations[0].EthPrivKeyCipher, organizations[0].QuotaPlanID, organizations[0].PublicAPIQuota,
		organizations[0].PublicAPIToken, organizations[0].HeaderURI, organizations[0].AvatarURI)
	c.Assert(err, qt.IsNil)
	c.Assert(int(id), qt.Not(qt.Equals), 0)
	organizations[0].ID = id

	count, err := API.DB.UpdateOrganization(integrators[0].SecretApiKey, organizations[0].EthAddress, "header", "avatar")
	c.Assert(err, qt.IsNil)
	c.Assert(count, qt.Equals, 1)
	organization, err := API.DB.GetOrganization(integrators[0].SecretApiKey, organizations[0].EthAddress)
	c.Assert(err, qt.IsNil)
	c.Assert(fmt.Sprintf("%x", organization.EthAddress), qt.Equals, fmt.Sprintf("%x", organizations[0].EthAddress))
	c.Assert(fmt.Sprintf("%x", organization.EthPrivKeyCipher), qt.Equals, fmt.Sprintf("%x", organizations[0].EthPrivKeyCipher))
	c.Assert(organization.QuotaPlanID, qt.Equals, organizations[0].QuotaPlanID)
	c.Assert(organization.PublicAPIQuota, qt.Equals, organizations[0].PublicAPIQuota)
	c.Assert(organization.PublicAPIToken, qt.Equals, organizations[0].PublicAPIToken)
	c.Assert(organization.HeaderURI, qt.Equals, "header")
	c.Assert(organization.AvatarURI, qt.Equals, "avatar")

	ethPrivKeyCipher, err := hex.DecodeString("bb")
	c.Assert(err, qt.IsNil)
	count, err = API.DB.UpdateOrganizationEthPrivKeyCipher(integrators[0].SecretApiKey, organizations[0].EthAddress, ethPrivKeyCipher)
	c.Assert(err, qt.IsNil)
	c.Assert(count, qt.Equals, 1)
	organization, err = API.DB.GetOrganization(integrators[0].SecretApiKey, organizations[0].EthAddress)
	c.Assert(err, qt.IsNil)
	c.Assert(fmt.Sprintf("%x", organization.EthPrivKeyCipher), qt.Equals, fmt.Sprintf("%x", ethPrivKeyCipher))
	c.Assert(fmt.Sprintf("%x", organization.EthAddress), qt.Equals, fmt.Sprintf("%x", organizations[0].EthAddress))
	c.Assert(organization.QuotaPlanID, qt.Equals, organizations[0].QuotaPlanID)
	c.Assert(organization.PublicAPIQuota, qt.Equals, organizations[0].PublicAPIQuota)
	c.Assert(organization.PublicAPIToken, qt.Equals, organizations[0].PublicAPIToken)
	c.Assert(organization.HeaderURI, qt.Equals, "header")
	c.Assert(organization.AvatarURI, qt.Equals, "avatar")

	count, err = API.DB.UpdateOrganizationPublicAPIToken(integrators[0].SecretApiKey, organizations[0].EthAddress, "bb")
	c.Assert(err, qt.IsNil)
	c.Assert(count, qt.Equals, 1)
	organization, err = API.DB.GetOrganization(integrators[0].SecretApiKey, organizations[0].EthAddress)
	c.Assert(err, qt.IsNil)
	c.Assert(organization.PublicAPIToken, qt.Equals, "bb")
	c.Assert(fmt.Sprintf("%x", organization.EthPrivKeyCipher), qt.Equals, fmt.Sprintf("%x", ethPrivKeyCipher))
	c.Assert(fmt.Sprintf("%x", organization.EthAddress), qt.Equals, fmt.Sprintf("%x", organizations[0].EthAddress))
	c.Assert(organization.QuotaPlanID, qt.Equals, organizations[0].QuotaPlanID)
	c.Assert(organization.PublicAPIQuota, qt.Equals, organizations[0].PublicAPIQuota)
	c.Assert(organization.AvatarURI, qt.Equals, "avatar")
	c.Assert(organization.AvatarURI, qt.Equals, "avatar")

	// count, err = API.DB.UpdateOrganizationPlan(integratorAPIKey []byte, ethAddress []byte, planID uuid.NullUUID, apiQuota int)

	// cleaning up (cascade delete from integrators)
	for _, integrator := range integrators {
		if err := API.DB.DeleteIntegrator(integrator.ID); err != nil {
			t.Errorf("error deleting test entity: %w", err)
		}

	}
}
