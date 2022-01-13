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
)

func TestIntegrator(t *testing.T) {
	t.Parallel()
	integrators := testcommon.CreateIntegrators(1)
	// test integrator creation
	req := types.APIRequest{
		CspUrlPrefix: integrators[0].CspUrlPrefix,
		CspPubKey:    hex.EncodeToString(integrators[0].CspPubKey),
		Name:         integrators[0].Name,
		Email:        integrators[0].Email,
	}
	respBody, statusCode := DoRequest(t,
		fmt.Sprintf("%s/v1/admin/accounts", API.URL), API.AuthToken, "POST", req)
	t.Logf("%s", respBody)
	qt.Assert(t, statusCode, qt.Equals, 200)
	var resp types.APIResponse
	err := json.Unmarshal(respBody, &resp)
	qt.Assert(t, err, qt.IsNil)
	qt.Check(t, resp.ID, qt.Not(qt.Equals), 0)
	qt.Check(t, len(resp.APIKey) > 0, qt.IsTrue)
	integrators[0].ID = resp.ID
	integrators[0].SecretApiKey = []byte(resp.APIKey)

	// test fetching integrators
	for _, integrator := range integrators {
		req := types.APIRequest{}
		respBody, statusCode := DoRequest(t,
			fmt.Sprintf("%s/v1/admin/accounts/%d", API.URL, integrator.ID),
			API.AuthToken, "GET", req)
		t.Logf("%s", respBody)
		qt.Assert(t, statusCode, qt.Equals, 200)
		var resp types.APIResponse
		err := json.Unmarshal(respBody, &resp)
		qt.Assert(t, err, qt.IsNil)
		qt.Assert(t, resp.Name, qt.Equals, integrator.Name)
		qt.Assert(t, bytes.Compare(resp.CspPubKey, integrator.CspPubKey), qt.Equals, 0)
		qt.Assert(t, resp.CspUrlPrefix, qt.Equals, integrator.CspUrlPrefix)
	}

	// test resetting integrator api keys
	for _, integrator := range integrators {
		req := types.APIRequest{}
		respBody, statusCode := DoRequest(t,
			fmt.Sprintf("%s/v1/admin/accounts/%d/key", API.URL, integrator.ID),
			API.AuthToken, "PATCH", req)
		t.Logf("%s", respBody)
		qt.Assert(t, statusCode, qt.Equals, 200)
		var resp types.APIResponse
		err := json.Unmarshal(respBody, &resp)
		qt.Assert(t, err, qt.IsNil)
		qt.Assert(t, resp.ID, qt.Equals, integrator.ID)
		qt.Assert(t, resp.APIKey, qt.Not(qt.Equals), string(integrator.SecretApiKey))
		qt.Assert(t, resp.APIKey, qt.Not(qt.Equals), "")
		integrator.SecretApiKey = []byte(resp.APIKey)
	}

	// cleaning up
	for _, integrator := range integrators {
		respBody, statusCode := DoRequest(t,
			fmt.Sprintf("%s/v1/admin/accounts/%d", API.URL, integrator.ID),
			API.AuthToken, "DELETE", types.APIRequest{})
		t.Logf("%s", respBody)
		qt.Assert(t, statusCode, qt.Equals, 200)
	}

	// test fetching integrators
	for _, integrator := range integrators {
		respBody, statusCode := DoRequest(t,
			fmt.Sprintf("%s/v1/admin/accounts/%d", API.URL, integrator.ID),
			API.AuthToken, "GET", types.APIRequest{})
		t.Logf("%s", respBody)
		qt.Assert(t, statusCode, qt.Equals, 400)
	}
}

func TestCreateIntegratorFail(t *testing.T) {
	t.Parallel()
	failIntegrators := testcommon.CreateIntegrators(1)
	// test failure: invalid api auth token
	req := types.APIRequest{
		CspUrlPrefix: API.CSP.UrlPrefix,
		CspPubKey:    "zzz",
		Name:         failIntegrators[0].Name,
		Email:        failIntegrators[0].Email,
	}
	respBody, statusCode := DoRequest(t,
		fmt.Sprintf("%s/v1/admin/accounts", API.URL), "1234", "POST", req)
	t.Logf("%s", respBody)
	qt.Assert(t, statusCode, qt.Equals, 401)

	// test failure: invalid pubKey
	req = types.APIRequest{
		CspUrlPrefix: API.CSP.UrlPrefix,
		CspPubKey:    "zzz",
		Name:         failIntegrators[0].Name,
		Email:        failIntegrators[0].Email,
	}
	respBody, statusCode = DoRequest(t, fmt.Sprintf("%s/v1/admin/accounts", API.URL),
		API.AuthToken, "POST", req)
	t.Logf("%s", respBody)
	qt.Assert(t, statusCode, qt.Equals, 400)

	// test failure: missing name, email
	respBody, statusCode = DoRequest(t,
		fmt.Sprintf("%s/v1/admin/accounts", API.URL),
		API.AuthToken, "POST", types.APIRequest{})
	t.Logf("%s", respBody)
	qt.Assert(t, statusCode, qt.Equals, 400)
}

func TestFetchIntegratorFail(t *testing.T) {
	t.Parallel()
	// test fetching nonexistent integrator
	respBody, statusCode := DoRequest(t,
		fmt.Sprintf("%s/v1/admin/accounts/222222222222", API.URL),
		API.AuthToken, "GET", types.APIRequest{})
	t.Logf("%s", respBody)
	qt.Assert(t, statusCode, qt.Equals, 400)
}
