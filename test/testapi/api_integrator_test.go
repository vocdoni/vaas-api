package testapi

import (
	"testing"

	"go.vocdoni.io/api/test/testcommon"
	"go.vocdoni.io/api/types"
	"go.vocdoni.io/dvote/log"
)

func TestIntegrator(t *testing.T) {
	// c := qt.New(t)
	integrators := testcommon.CreateIntegrators(1)
	req := &types.APIRequest{CspUrlPrefix: API.CSP.UrlPrefix,
		CspPubKey: API.CSP.CspPubKey,
		Name:      integrators[0].Name}
	resp := DoRequest(t, API.URL+"/admin/accounts", API.AuthToken, "POST", *req)

	log.Infof("%s", resp)
	// cleaning up
	for _, integrator := range integrators {
		if err := API.DB.DeleteIntegrator(integrator.ID); err != nil {
			t.Errorf("error deleting test entity: %w", err)
		}

	}
}
