package urlapi

import (
	"encoding/json"
	"fmt"

	"go.vocdoni.io/api/types"
	"go.vocdoni.io/api/util"
	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/httprouter/bearerstdapi"
	"go.vocdoni.io/dvote/vochain/scrutinizer/indexertypes"
	"go.vocdoni.io/proto/build/go/models"
)

func (u *URLAPI) enableVoterHandlers() error {
	if err := u.api.RegisterMethod(
		"/pub/censuses/{censusId}/token",
		"POST",
		bearerstdapi.MethodAccessTypeQuota,
		u.registerPublicKeyHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/pub/processes/{processId}",
		"GET",
		bearerstdapi.MethodAccessTypeQuota,
		u.getProcessInfoPublicHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/pub/processes/{processId}/auth/{signature}",
		"GET",
		bearerstdapi.MethodAccessTypeQuota,
		u.getProcessInfoConfidentialHandler,
	); err != nil {
		return err
	}
	return nil
}

// POST https://server/v1/pub/censuses/<censusId>/token
// registerPublicKeyHandler registers a voter's public key with a census token
func (u *URLAPI) registerPublicKeyHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// GET https://server/v1/pub/processes/<processId>
// getProcessInfoPublicHandler gets public process info
func (u *URLAPI) getProcessInfoPublicHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var err error
	var resp types.APIProcess
	var processId []byte
	var process *types.Election
	var vochainProcess *indexertypes.Process
	var results *types.VochainResults
	var processMetadata *models.Process
	if processId, err = util.GetBytesID(ctx, "processId"); err != nil {
		return err
	}

	// Fetch process from db
	if process, err = u.db.GetElection([]byte{}, []byte{}, processId); err != nil {
		return err
	}

	// Fetch process from vochain
	if vochainProcess, err = u.vocClient.GetProcess(processId); err != nil {
		return err
	}

	// Fetch results
	if vochainProcess.HaveResults {
		if results, err = u.vocClient.GetResults(processId); err != nil {
			return err
		}
	}

	// Fetch metadata
	metadataUri := vochainProcess.Metadata
	if processMetadata, err = u.vocClient.FetchProcessMetadata(metadataUri); err != nil {
		return err
	}

	// Parse all the information
	resp = parseProcessInfo(process, vochainProcess, results, processMetadata)

	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("error marshaling JSON: %w", err)
	}
	if err = ctx.Send(data); err != nil {
		return err
	}
	return nil
}

// GET https://server/v1/pub/processes/<processId>/auth/<signature>
// getProcessInfoConfidentialHandler gets process info, including private metadata,
//  checking the voter's signature for inclusion in the census
func (u *URLAPI) getProcessInfoConfidentialHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// TODO add listProcessesInfoHandler

func parseProcessInfo(db *types.Election, vc *indexertypes.Process,
	results *types.VochainResults, meta *models.Process) (process types.APIProcess) {
	// TODO implement this function
	// TODO update when blind is added to election
	// if db.Blind {
	process.Type = "blind-"
	// 	} else {
	// 	resp.Type = "signed-"
	// }
	if db.Confidential {
		process.Type += "confidential-"
	} else {
		process.Type += "plain-"
	}
	if db.HiddenResults {
		process.Type += "hidden-results"
	} else {
		process.Type += "rolling-results"
	}
	return process
}
