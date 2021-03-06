package urlapi

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"go.vocdoni.io/api/types"
	"go.vocdoni.io/api/util"
	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/crypto/saltedkey"
	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/httprouter/bearerstdapi"
	"go.vocdoni.io/dvote/log"
)

func (u *URLAPI) enablePublicHandlers() error {
	if err := u.api.RegisterMethod(
		"/pub/censuses/{censusId}/token",
		"POST",
		bearerstdapi.MethodAccessTypePublic,
		u.registerPublicKeyHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/pub/organizations/{organizationId}/elections/{type}",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		u.listProcessesHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/pub/organizations/{organizationId}/elections",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		u.listProcessesHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/pub/elections/{electionId}",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		u.getProcessInfoPublicHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/pub/elections/{electionId}/vote",
		"POST",
		bearerstdapi.MethodAccessTypePublic,
		u.submitVotePublicHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/pub/elections/{electionId}/auth/{signature}",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		u.getProcessInfoConfidentialHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/pub/organizations/{organizationId}",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		u.getOrganizationHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/pub/nullifiers/{nullifier}",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		u.getVoteHandler,
	); err != nil {
		return err
	}
	return nil
}

// POST https://server/v1/pub/censuses/<censusId>/token
// registerPublicKeyHandler registers a voter's public key with a census token
func (u *URLAPI) registerPublicKeyHandler(
	msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// GET https://server/v1/pub/organizations/<organizationId>/elections/signed
// GET https://server/v1/pub/organizations/<organizationId>/elections/blind
// GET https://server/v1/pub/organizations/<organizationId>/elections/paused
// GET https://server/v1/pub/organizations/<organizationId>/elections/canceled
// GET https://server/v1/pub/organizations/<organizationId>/elections/active
// GET https://server/v1/pub/organizations/<organizationId>/elections/upcoming
// GET https://server/v1/pub/organizations/<organizationId>/elections/ended
// listProcessesHandler' lists signed, blind, active, ended, or upcoming processes
func (u *URLAPI) listProcessesHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	entityId, err := util.GetBytesID(ctx, "organizationId")
	if err != nil {
		return err
	}
	list, err :=
		u.getProcessList(ctx.URLParam("type"), []byte{}, entityId, false)
	if err != nil {
		return err
	}
	return sendResponse(list, ctx)
}

// GET https://server/v1/pub/elections/<processId>
// getProcessInfoPublicHandler gets public process info
func (u *URLAPI) getProcessInfoPublicHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	processId, err := util.GetBytesID(ctx, "electionId")
	if err != nil {
		return err
	}

	// Fetch process from vochain
	vochainProcess, err := u.vocClient.GetProcess(processId)
	if err != nil {
		return fmt.Errorf("unable to get process: %w", err)
	}

	// Fetch results
	var results *types.VochainResults
	if vochainProcess.HaveResults {
		if results, err = u.vocClient.GetResults(processId); err != nil {
			return fmt.Errorf("unable to get results %w", err)
		}
	}

	dbElection, err := u.db.GetElectionPublic(vochainProcess.EntityID, processId)
	if err != nil {
		return fmt.Errorf("could not fetch election %x from db: %w", processId, err)
	}

	if dbElection.Confidential {
		return fmt.Errorf("process %x is confidential, use authenticated API", processId)
	}

	// Fetch metadata
	processMetadata, err := u.vocClient.FetchProcessMetadata(vochainProcess.Metadata)
	if err != nil {
		return fmt.Errorf("unable to get metadata: %w", err)
	}

	// Parse all the information
	resp, err := u.parseProcessInfo(vochainProcess, results, processMetadata, types.ProofType(dbElection.ProofType))
	if err != nil {
		return fmt.Errorf("could not parse information for process %x: %w", processId, err)
	}

	return sendResponse(resp, ctx)
}

// GET https://server/v1/pub/elections/<processId>/auth/<signature>
// getProcessInfoConfidentialHandler gets process info, including private metadata,
//  checking the voter's signature for inclusion in the census
func (u *URLAPI) getProcessInfoConfidentialHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	processId, err := util.GetBytesID(ctx, "electionId")
	if err != nil {
		return err
	}
	cspSignature, err := util.GetBytesID(ctx, "signature")
	if err != nil {
		return err
	}

	// Fetch process from vochain
	vochainProcess, err := u.vocClient.GetProcess(processId)
	if err != nil {
		return fmt.Errorf("unable to get process: %w", err)
	}

	// Fetch results
	var results *types.VochainResults
	if vochainProcess.HaveResults {
		if results, err = u.vocClient.GetResults(processId); err != nil {
			return fmt.Errorf("unable to get results %w", err)
		}
	}

	dbElection, err := u.db.GetElectionPrivate(vochainProcess.EntityID, processId)
	if err != nil {
		return fmt.Errorf("could not fetch election %x from db: %w", processId, err)
	}

	if err = verifyCspSharedSignature(processId, cspSignature, vochainProcess.CensusRoot); err != nil {
		return fmt.Errorf("shared key not valid to decrypt process %x: %w", processId, err)
	}

	processMetadata, err := u.getProcessMetadataPriv(
		dbElection.Confidential, dbElection.MetadataPrivKey, vochainProcess.Metadata)
	if err != nil {
		return err
	}

	// Parse all the information
	resp, err := u.parseProcessInfo(vochainProcess, results, processMetadata, types.ProofType(dbElection.ProofType))
	if err != nil {
		return fmt.Errorf("could not parse information for process %x: %w", processId, err)
	}

	return sendResponse(resp, ctx)
}

// GET https://server/v1/pub/account/organizations/<organizationId>
// getOrganizationHandler fetches an entity
func (u *URLAPI) getOrganizationHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	ethAddress, err := hex.DecodeString(ctx.URLParam("organizationId"))
	if err != nil {
		return fmt.Errorf("invalid organizationId: %w", err)
	}
	// Fetch process from vochain
	metaUri, _, _, err := u.vocClient.GetAccount(ethAddress)
	if err != nil {
		return fmt.Errorf("unable to get account: %w", err)
	}

	// Fetch metadata
	organizationMetadata, err := u.vocClient.FetchOrganizationMetadata(metaUri)
	if err != nil {
		return fmt.Errorf("could not get organization metadata with URI\"%s\": %w", metaUri, err)
	}
	resp := types.APIResponse{
		Name:        organizationMetadata.Name["default"],
		Description: organizationMetadata.Description["default"],
		Avatar:      organizationMetadata.Media.Avatar,
		Header:      organizationMetadata.Media.Header,
	}
	return sendResponse(resp, ctx)
}

// POST https://server/v1/pub/elections/<processId>/vote
func (u *URLAPI) submitVotePublicHandler(
	msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	log.Debugf("query to submit vote for process %s", ctx.URLParam("electionId"))
	req, err := util.UnmarshalRequest(msg)
	if err != nil {
		return err
	}
	var votePkg []byte
	if votePkg, err = base64.StdEncoding.DecodeString(req.Vote); err != nil {
		return fmt.Errorf("could not decode vote pkg to base64: %w", err)
	}
	var resp types.APIResponse
	if resp.Nullifier, err = u.vocClient.RelayVote(votePkg); err != nil {
		return fmt.Errorf("could not submit vote tx: %w", err)
	}

	return sendResponse(resp, ctx)
}

// GET https://server/v1//pub/nullifiers/{nullifier}
// getVoteHandler returns a single vote envelope for the given nullifier
func (u *URLAPI) getVoteHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	nullifier, err := util.GetBytesID(ctx, "nullifier")
	if err != nil {
		return err
	}
	var resp types.APIResponse
	resp.Registered = new(bool)
	if resp.ElectionID, *resp.Registered, err = u.vocClient.GetVoteStatus(nullifier); err != nil {
		return fmt.Errorf("could not get envelope status for vote with nullifier %x: %w", nullifier, err)
	}
	if *resp.Registered {
		resp.ExplorerUrl = fmt.Sprintf("%s%x", u.config.ExplorerVoteUrl, nullifier)
	}
	return sendResponse(resp, ctx)
}

func verifyCspSharedSignature(processId, cspSignature, censusRoot []byte) error {
	saltedCspPubKey, err := ethereum.PubKeyFromSignature(processId, cspSignature)
	if err != nil {
		return fmt.Errorf("could not extract csp pubKey from signature: %w", err)
	}
	decompressedCspKey, err := ethereum.DecompressPubKey(saltedCspPubKey)
	if err != nil {
		return fmt.Errorf("could not decompress csp public key: %w", err)
	}
	rootPub, err := ethereum.DecompressPubKey(censusRoot)
	if err != nil {
		return fmt.Errorf("could not decompress census root key: %w", err)
	}
	ecdsaKey, err := ethcrypto.UnmarshalPubkey(rootPub)
	if err != nil {
		return fmt.Errorf("could not decode csp public key from election configuration: %w", err)
	}
	saltedKey, err := saltedkey.SaltECDSAPubKey(ecdsaKey, processId)
	if err != nil {
		return fmt.Errorf("could not salt csp public key: %w", err)
	}
	if !bytes.Equal(decompressedCspKey, ethcrypto.FromECDSAPub(saltedKey)) {
		return fmt.Errorf("signature pubKey %x does not match election census root %x",
			decompressedCspKey, ethcrypto.FromECDSAPub(saltedKey))
	}
	return nil
}
