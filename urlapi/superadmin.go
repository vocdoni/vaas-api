package urlapi

import (
	"encoding/hex"
	"fmt"

	"go.vocdoni.io/api/types"
	"go.vocdoni.io/api/util"
	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/httprouter/bearerstdapi"
	"go.vocdoni.io/dvote/log"
	dvoteUtil "go.vocdoni.io/dvote/util"
)

const (
	// TODO determine correct max requests for integrators
	INTEGRATOR_MAX_REQUESTS = 2 << 16
)

func (u *URLAPI) enableSuperadminHandlers(adminToken string) error {
	u.api.SetAdminToken(adminToken)
	log.Infof("admin token;: %s", adminToken)
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
func (u *URLAPI) createIntegratorAccountHandler(
	msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	req, err := util.UnmarshalRequest(msg)
	if err != nil {
		return err
	}
	resp := types.APIResponse{APIKey: util.GenerateBearerToken()}
	apiKey, err := hex.DecodeString(resp.APIKey)
	if err != nil {
		return err
	}

	cspPubKey, err := hex.DecodeString(dvoteUtil.TrimHex(req.CspPubKey))
	if err != nil {
		return fmt.Errorf("error decoding csp pub key %s", req.CspPubKey)
	}
	if resp.ID, err = u.db.CreateIntegrator(apiKey,
		cspPubKey, req.CspUrlPrefix, req.Name, req.Email); err != nil {
		return err
	}
	return sendResponse(resp, ctx)
}

// PUT https://server/v1/admin/accounts/<id>
// updateIntegratorAccountHandler updates an existing integrator account
func (u *URLAPI) updateIntegratorAccountHandler(
	msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	id, err := util.GetIntID(ctx, "id")
	if err != nil {
		return err
	}
	req, err := util.UnmarshalRequest(msg)
	if err != nil {
		return err
	}

	cspPubKey, err := hex.DecodeString(dvoteUtil.TrimHex(req.CspPubKey))
	if err != nil {
		return err
	}
	if _, err = u.db.UpdateIntegrator(id, cspPubKey, req.CspUrlPrefix, req.Name); err != nil {
		return err
	}
	return sendResponse(types.APIResponse{}, ctx)
}

// PATCH https://server/v1/admin/accounts/<id>/key
// resetIntegratorKeyHandler resets an integrator api key
func (u *URLAPI) resetIntegratorKeyHandler(
	msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	id, err := util.GetIntID(ctx, "id")
	if err != nil {
		return err
	}

	// Now generate a new api key & update integrator
	resp := types.APIResponse{APIKey: util.GenerateBearerToken(), ID: id}
	apiKey, err := hex.DecodeString(resp.APIKey)
	if err != nil {
		return err
	}
	if _, err = u.db.UpdateIntegratorApiKey(id, apiKey); err != nil {
		return err
	}
	return sendResponse(resp, ctx)
}

// GET https://server/v1/admin/accounts/<id>
// getIntegratorAccountHandler fetches an integrator account
func (u *URLAPI) getIntegratorAccountHandler(
	msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	id, err := util.GetIntID(ctx, "id")
	if err != nil {
		return err
	}
	integrator, err := u.db.GetIntegrator(id)
	if err != nil {
		return err
	}
	resp := types.APIResponse{
		Name:         integrator.Name,
		CspPubKey:    integrator.CspPubKey,
		CspUrlPrefix: integrator.CspUrlPrefix,
	}
	return sendResponse(resp, ctx)
}

// DELETE https://server/v1/admin/accounts/<id>
// deleteIntegratorAccountHandler deletes an integrator account
func (u *URLAPI) deleteIntegratorAccountHandler(
	msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	id, err := util.GetIntID(ctx, "id")
	if err != nil {
		return err
	}
	if err = u.db.DeleteIntegrator(id); err != nil {
		return err
	}
	return sendResponse(types.APIResponse{}, ctx)
}
