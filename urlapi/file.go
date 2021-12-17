package urlapi

import (
	"go.vocdoni.io/api/types"
	"go.vocdoni.io/api/util"
	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/httprouter/bearerstdapi"
)

func (u *URLAPI) enableFileHandlers() error {
	if err := u.api.RegisterMethod(
		"/pub/file/{hash}",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		u.createGetFileHandler,
	); err != nil {
		return err
	}
	return nil
}

// GET https://server/v1/pub/file/{hash}
// createOrganizationHandler creates a new entity
func (u *URLAPI) createGetFileHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	hash, err := util.GetBytesID(ctx, "hash")
	if err != nil {
		return err
	}
	var content []byte
	// u.db.GetFile(hash)
	return sendResponse(types.APIResponse{Content: content}, ctx)
}
