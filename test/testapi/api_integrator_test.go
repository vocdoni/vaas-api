package testapi

import (
	"encoding/json"
	"fmt"
	"testing"

	qt "github.com/frankban/quicktest"
	"go.vocdoni.io/api/test/testcommon"
	"go.vocdoni.io/api/types"
	"go.vocdoni.io/dvote/log"
)

func TestIntegrator(t *testing.T) {
	integrators := testcommon.CreateIntegrators(1)
	req := types.APIRequest{CspUrlPrefix: API.CSP.UrlPrefix,
		CspPubKey: fmt.Sprintf("%x", API.CSP.CspPubKey),
		Name:      integrators[0].Name}
	respBody, statusCode := DoRequest(t, API.URL+"/v1/admin/accounts", API.AuthToken, "POST", req)
	if statusCode != 200 {
		log.Errorf("error response %s", string(respBody))
		t.FailNow()
	}
	var resp types.APIResponse
	err := json.Unmarshal(respBody, &resp)
	qt.Check(t, err, qt.IsNil)
	qt.Check(t, resp.ID, qt.Not(qt.Equals), 0)
	qt.Check(t, len(resp.APIKey) > 0, qt.IsTrue)
	integrators[0].ID = resp.ID
	log.Infof("%s", respBody)
	// cleaning up
	for _, integrator := range integrators {
		if err := API.DB.DeleteIntegrator(integrator.ID); err != nil {
			t.Errorf("error deleting test integrator: %v", err)
		}
	}
}
