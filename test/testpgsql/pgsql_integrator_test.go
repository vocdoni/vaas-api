package testpgsql

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"testing"

	qt "github.com/frankban/quicktest"
	"go.vocdoni.io/api/test/testcommon"
	"go.vocdoni.io/dvote/log"
)

func TestIntegrator(t *testing.T) {
	t.Parallel()
	c := qt.New(t)
	integrators := testcommon.CreateIntegrators(1)
	integrators[0].SecretApiKey = []byte(fmt.Sprintf("key%d", rand.Intn(10000)))
	id, err := API.DB.CreateIntegrator(integrators[0].SecretApiKey, integrators[0].CspPubKey,
		integrators[0].CspUrlPrefix, integrators[0].Name, integrators[0].Email)
	c.Assert(err, qt.IsNil)
	c.Assert(int(id), qt.Not(qt.Equals), 0)
	integrators[0].ID = id

	integrator, err := API.DB.GetIntegrator(integrators[0].ID)
	log.Infof("%w", integrator)
	c.Assert(err, qt.IsNil)
	c.Assert(fmt.Sprintf("%x", integrator.SecretApiKey), qt.DeepEquals, fmt.Sprintf("%x", integrators[0].SecretApiKey))

	count, err := API.DB.UpdateIntegrator(integrators[0].ID, []byte{}, "", "New Name")
	c.Assert(err, qt.IsNil)
	c.Assert(count, qt.Equals, 1)
	integrator, err = API.DB.GetIntegrator(integrators[0].ID)
	c.Assert(err, qt.IsNil)
	c.Assert(integrator.CspUrlPrefix, qt.Equals, integrators[0].CspUrlPrefix)
	c.Assert(integrator.Name, qt.Equals, "New Name")
	c.Assert(fmt.Sprintf("%x", integrator.CspPubKey), qt.Equals, fmt.Sprintf("%x", integrators[0].CspPubKey))

	apiKey, err := hex.DecodeString("bb")
	c.Assert(err, qt.IsNil)
	count, err = API.DB.UpdateIntegratorApiKey(integrators[0].ID, apiKey)
	c.Assert(err, qt.IsNil)
	c.Assert(count, qt.Equals, 1)
	integrator, err = API.DB.GetIntegrator(integrators[0].ID)
	c.Assert(err, qt.IsNil)
	c.Assert(fmt.Sprintf("%x", integrator.SecretApiKey), qt.Equals, fmt.Sprintf("%x", apiKey))

	_, err = API.DB.GetIntegratorApiKeysList()
	c.Assert(err, qt.IsNil)
	// cleaning up
	for _, integrator := range integrators {
		if err := API.DB.DeleteIntegrator(integrator.ID); err != nil {
			t.Errorf("error deleting test entity: %w", err)
		}

	}
}
