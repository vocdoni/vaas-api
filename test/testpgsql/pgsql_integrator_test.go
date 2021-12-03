package testpgsql

import (
	"testing"

	qt "github.com/frankban/quicktest"
	"go.vocdoni.io/api/test/testcommon"
)

func TestIntegrator(t *testing.T) {
	c := qt.New(t)
	integrators := testcommon.CreateIntegrators(1)
	id, err := API.DB.CreateIntegrator(integrators[0].SecretApiKey, integrators[0].CspPubKey,
		integrators[0].CspUrlPrefix, integrators[0].Name, integrators[0].Email)
	c.Assert(err, qt.IsNil)
	c.Assert(int(id), qt.Not(qt.Equals), 0)
	integrators[0].ID = id
	// cleaning up
	for _, integrator := range integrators {
		if err := API.DB.DeleteIntegrator(integrator.ID); err != nil {
			t.Errorf("error deleting test entity: %w", err)
		}

	}
}
