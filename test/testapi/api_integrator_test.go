package testapi

import (
	"bytes"
	"encoding/hex"
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
	failIntegrators := testcommon.CreateIntegrators(1)

	// test integrator creation
	req := types.APIRequest{CspUrlPrefix: integrators[0].CspUrlPrefix,
		CspPubKey: hex.EncodeToString(integrators[0].CspPubKey),
		Name:      integrators[0].Name,
		Email:     integrators[0].Email}
	respBody, statusCode := DoRequest(t, API.URL+"/v1/admin/accounts", API.AuthToken, "POST", req)
	if statusCode != 200 {
		log.Errorf("error response %s", string(respBody))
		t.FailNow()
	}
	var resp types.APIResponse
	err := json.Unmarshal(respBody, &resp)
	qt.Assert(t, err, qt.IsNil)
	qt.Check(t, resp.ID, qt.Not(qt.Equals), 0)
	qt.Check(t, len(resp.APIKey) > 0, qt.IsTrue)
	integrators[0].ID = resp.ID
	integrators[0].SecretApiKey = []byte(resp.APIKey)
	log.Infof("%s", respBody)

	// test failure: invalid api auth token
	req = types.APIRequest{CspUrlPrefix: API.CSP.UrlPrefix,
		CspPubKey: "zzz",
		Name:      failIntegrators[0].Name,
		Email:     failIntegrators[0].Email}
	respBody, statusCode = DoRequest(t, API.URL+"/v1/admin/accounts", "1234", "POST", req)
	qt.Assert(t, statusCode, qt.Equals, 401)
	log.Infof("%s", respBody)

	// test failure: invalid pubKey
	req = types.APIRequest{CspUrlPrefix: API.CSP.UrlPrefix,
		CspPubKey: "zzz",
		Name:      failIntegrators[0].Name,
		Email:     failIntegrators[0].Email}
	respBody, statusCode = DoRequest(t, API.URL+"/v1/admin/accounts", API.AuthToken, "POST", req)
	qt.Assert(t, statusCode, qt.Equals, 400)
	log.Infof("%s", respBody)

	// test failure: missing name, email
	req = types.APIRequest{}
	respBody, statusCode = DoRequest(t, API.URL+"/v1/admin/accounts", API.AuthToken, "POST", req)
	qt.Assert(t, statusCode, qt.Equals, 400)
	log.Infof("%s", respBody)

	// test fetching integrators
	for _, integrator := range integrators {
		req := types.APIRequest{}
		respBody, statusCode := DoRequest(t, fmt.Sprintf("%s/v1/admin/accounts/%d",
			API.URL, integrator.ID), API.AuthToken, "GET", req)
		if statusCode != 200 {
			log.Errorf("error response %s", string(respBody))
			t.FailNow()
		}
		var resp types.APIResponse
		err := json.Unmarshal(respBody, &resp)
		qt.Assert(t, err, qt.IsNil)
		log.Infof("%s", respBody)
		qt.Assert(t, resp.Name, qt.Equals, integrator.Name)
		qt.Assert(t, bytes.Compare(resp.CspPubKey, integrator.CspPubKey), qt.Equals, 0)
		qt.Assert(t, resp.CspUrlPrefix, qt.Equals, integrator.CspUrlPrefix)
	}

	// test fetching nonexistent integrator
	req = types.APIRequest{}
	respBody, statusCode = DoRequest(t, fmt.Sprintf("%s/v1/admin/accounts/222222222222",
		API.URL), API.AuthToken, "GET", req)
	qt.Assert(t, statusCode, qt.Equals, 400)
	err = json.Unmarshal(respBody, &resp)
	if err != nil {
		t.Fatal(err)
	}
	log.Infof("%s", respBody)

	// test resetting integrator api keys
	for _, integrator := range integrators {
		req := types.APIRequest{}
		respBody, statusCode := DoRequest(t, fmt.Sprintf("%s/v1/admin/accounts/%d/key",
			API.URL, integrator.ID), API.AuthToken, "PATCH", req)
		if statusCode != 200 {
			log.Errorf("error response %s", string(respBody))
			t.FailNow()
		}
		var resp types.APIResponse
		err := json.Unmarshal(respBody, &resp)
		qt.Assert(t, err, qt.IsNil)
		log.Infof("%s", respBody)
		qt.Assert(t, resp.ID, qt.Equals, integrator.ID)
		qt.Assert(t, resp.APIKey, qt.Not(qt.Equals), string(integrator.SecretApiKey))
		qt.Assert(t, resp.APIKey, qt.Not(qt.Equals), "")
		integrator.SecretApiKey = []byte(resp.APIKey)
	}

	// test fetching nonexistent integrator
	req = types.APIRequest{}
	respBody, statusCode = DoRequest(t, fmt.Sprintf("%s/v1/admin/accounts/222222222222",
		API.URL), API.AuthToken, "GET", req)
	qt.Assert(t, statusCode, qt.Equals, 400)
	err = json.Unmarshal(respBody, &resp)
	if err != nil {
		t.Fatal(err)
	}
	log.Infof("%s", respBody)

	// cleaning up
	for _, integrator := range integrators {
		if err := API.DB.DeleteIntegrator(integrator.ID); err != nil {
			t.Errorf("error deleting test integrator: %v", err)
		}
	}

	// test fetching integrators
	for _, integrator := range integrators {
		req := types.APIRequest{}
		respBody, statusCode := DoRequest(t, fmt.Sprintf("%s/v1/admin/accounts/%d",
			API.URL, integrator.ID), API.AuthToken, "GET", req)
		qt.Assert(t, statusCode, qt.Equals, 400)
		err = json.Unmarshal(respBody, &resp)
		if err != nil {
			t.Fatal(err)
		}
		log.Infof("%s", respBody)
	}
}
