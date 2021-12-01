package urlapi

import (
	"go.vocdoni.io/api/types"
	"go.vocdoni.io/api/util"
	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/httprouter/bearerstdapi"
)

const (
	// TODO determine corred max requests for integrators
	INTEGRATOR_MAX_REQUESTS = 2 << 16
)

func (u *URLAPI) enableSuperadminHandlers() error {
	if err := u.api.RegisterMethod(
		"/admin/accounts",
		"POST",
		bearerstdapi.MethodAccessTypeAdmin,
		u.createIntegratorAccountHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/admin/accounts/{id}",
		"PUT",
		bearerstdapi.MethodAccessTypeAdmin,
		u.updateIntegratorAccountHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/admin/accounts/{id}/key",
		"PATCH",
		bearerstdapi.MethodAccessTypeAdmin,
		u.resetIntegratorKeyHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/admin/accounts/{id}",
		"GET",
		bearerstdapi.MethodAccessTypeAdmin,
		u.getIntegratorAccountHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/admin/accounts/{id}",
		"DELETE",
		bearerstdapi.MethodAccessTypeAdmin,
		u.deleteIntegratorAccountHandler,
	); err != nil {
		return err
	}
	return nil
}

// POST https://server/v1/admin/accounts
// createIntegratorAccountHandler creates a new integrator account
func (u *URLAPI) createIntegratorAccountHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var err error
	var req types.APIRequest
	var resp types.APIResponse
	if req, err = util.UnmarshalRequest(ctx); err != nil {
		return err
	}
	resp.APIKey = util.GenerateBearerToken()
	if resp.ID, err = u.db.CreateIntegrator([]byte(resp.APIKey), req.CspPubKey, req.Name, req.CspUrlPrefix); err != nil {
		return err
	}
	u.registerToken(resp.APIKey, INTEGRATOR_MAX_REQUESTS)
	return util.SendResponse(resp, ctx)
}

// PUT https://server/v1/admin/accounts/<id>
// updateIntegratorAccountHandler updates an existing integrator account
func (u *URLAPI) updateIntegratorAccountHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var err error
	var req types.APIRequest
	var resp types.APIResponse
	var id int
	if id, err = util.GetID(ctx); err != nil {
		return err
	}
	if req, err = util.UnmarshalRequest(ctx); err != nil {
		return err
	}
	if resp.ID, err = u.db.UpdateIntegrator(id, req.CspPubKey, req.Name, req.CspUrlPrefix); err != nil {
		return err
	}
	return util.SendResponse(resp, ctx)
}

// PATCH https://server/v1/admin/accounts/<id>/key
// resetIntegratorKeyHandler resets an integrator api key
func (u *URLAPI) resetIntegratorKeyHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var err error
	var resp types.APIResponse
	var id int
	if id, err = util.GetID(ctx); err != nil {
		return err
	}
	// Before updating integrator key, fetch & revoke the old key
	oldIntegrator, err := u.db.GetIntegrator(id)
	if err != nil {
		return err
	}
	u.revokeToken(string(oldIntegrator.SecretApiKey))

	// Now generate a new api key & update integrator
	resp.APIKey = util.GenerateBearerToken()
	if resp.ID, err = u.db.UpdateIntegratorApiKey(id, []byte(resp.APIKey)); err != nil {
		return err
	}
	u.registerToken(resp.APIKey, INTEGRATOR_MAX_REQUESTS)
	return util.SendResponse(resp, ctx)
}

// GET https://server/v1/admin/accounts/<id>
// getIntegratorAccountHandler fetches an integrator account
func (u *URLAPI) getIntegratorAccountHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var err error
	var resp types.APIResponse
	var integrator *types.Integrator
	var id int
	if id, err = util.GetID(ctx); err != nil {
		return err
	}
	if integrator, err = u.db.GetIntegrator(id); err != nil {
		return err
	}
	resp.Name = integrator.Name
	resp.CspPubKey = integrator.CspPubKey
	resp.CspUrlPrefix = integrator.CspUrlPrefix
	return util.SendResponse(resp, ctx)
}

// DELETE https://server/v1/admin/accounts/<id>
// deleteIntegratorAccountHandler deletes an integrator account
func (u *URLAPI) deleteIntegratorAccountHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var err error
	var resp types.APIResponse
	var id int
	if id, err = util.GetID(ctx); err != nil {
		return err
	}
	if err = u.db.DeleteIntegrator(id); err != nil {
		return err
	}
	return util.SendResponse(resp, ctx)
}
