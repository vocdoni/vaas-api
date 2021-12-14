package urlapi

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.vocdoni.io/api/types"
	"go.vocdoni.io/api/util"
	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/httprouter/bearerstdapi"
	"go.vocdoni.io/dvote/log"
	dvoteutil "go.vocdoni.io/dvote/util"
	"go.vocdoni.io/dvote/vochain/scrutinizer/indexertypes"
	"go.vocdoni.io/proto/build/go/models"
)

func (u *URLAPI) enableEntityHandlers() error {
	if err := u.api.RegisterMethod(
		"/priv/account/organizations",
		"POST",
		bearerstdapi.MethodAccessTypePrivate,
		u.createOrganizationHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/account/organizations/{organizationId}",
		"GET",
		bearerstdapi.MethodAccessTypePrivate,
		u.getOrganizationPrivateHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/account/organizations/{organizationId}",
		"DELETE",
		bearerstdapi.MethodAccessTypePrivate,
		u.deleteOrganizationHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/account/organizations/{organizationId}/key",
		"PATCH",
		bearerstdapi.MethodAccessTypePrivate,
		u.resetOrganizationKeyHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/organizations/{organizationId}/metadata",
		"PUT",
		bearerstdapi.MethodAccessTypePrivate,
		u.setOrganizationMetadataHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/organizations/{organizationId}/elections/{type}",
		"POST",
		bearerstdapi.MethodAccessTypePrivate,
		u.createProcessHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/organizations/{entityId}/elections/{type}",
		"GET",
		bearerstdapi.MethodAccessTypePrivate,
		u.listProcessesPrivateHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/censuses",
		"POST",
		bearerstdapi.MethodAccessTypePrivate,
		u.createCensusHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/censuses/{censusId}/tokens/*",
		"POST",
		bearerstdapi.MethodAccessTypePrivate,
		u.addCensusTokensHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/censuses/{censusId}/tokens/{tokenId}",
		"GET",
		bearerstdapi.MethodAccessTypePrivate,
		u.getCensusTokenHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/censuses/{censusId}/tokens/{tokenId}",
		"DELETE",
		bearerstdapi.MethodAccessTypePrivate,
		u.deleteCensusTokenHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/censuses/{censusId}/tokens/{tokenId}",
		"DELETE",
		bearerstdapi.MethodAccessTypePrivate,
		u.deleteCensusTokenHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/censuses/{censusId}/keys/{publicKey}",
		"DELETE",
		bearerstdapi.MethodAccessTypePrivate,
		u.deletePublicKeyHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/censuses/{censusId}/import/*",
		"POST",
		bearerstdapi.MethodAccessTypePrivate,
		u.importPublicKeysHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/elections/{electionId}/status",
		"PUT",
		bearerstdapi.MethodAccessTypePrivate,
		u.setProcessStatusHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/elections/{electionId}",
		"GET",
		bearerstdapi.MethodAccessTypePrivate,
		u.getProcessHandler,
	); err != nil {
		return err
	}
	return nil
}

// POST https://server/v1/priv/account/organizations
// createOrganizationHandler creates a new entity
func (u *URLAPI) createOrganizationHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	var err error
	var resp types.APIResponse
	var req types.APIRequest
	var entityPrivKey []byte
	var integratorPrivKey []byte
	var orgApiToken string
	var encryptedPrivKey []byte
	var metaURI string
	// var organizationMetadataKey []byte
	if req, err = util.UnmarshalRequest(msg); err != nil {
		return err
	}
	if integratorPrivKey, err = util.GetAuthToken(msg); err != nil {
		return err
	}
	orgApiToken = util.GenerateBearerToken()

	ethSignKeys := ethereum.NewSignKeys()
	if err = ethSignKeys.Generate(); err != nil {
		return fmt.Errorf("could not generate ethereum keys: %w", err)
	}

	// Encrypt private key to store in db
	_, priv := ethSignKeys.HexString()
	if entityPrivKey, err = hex.DecodeString(priv); err != nil {
		return fmt.Errorf("could not decode entity private key: %w", err)
	}

	if encryptedPrivKey, err = util.EncryptSymmetric(entityPrivKey, integratorPrivKey); err != nil {
		return fmt.Errorf("could not encrypt entity private key: %w", err)
	}

	// Post metadata to ipfs
	if metaURI, err = u.vocClient.SetEntityMetadata(types.EntityMetadata{
		Version:     "1.0",
		Languages:   []string{},
		Name:        map[string]string{"default": req.Name},
		Description: map[string]string{"default": req.Description},
		NewsFeed:    map[string]string{},
		Media: types.EntityMedia{
			Avatar: req.Avatar,
			Header: req.Header,
		},
	}, ethSignKeys.Address().Bytes()); err != nil {
		return fmt.Errorf("could not set entity metadata: %w", err)
	}

	// Register organization to database
	if _, err = u.db.CreateOrganization(integratorPrivKey, ethSignKeys.Address().Bytes(),
		encryptedPrivKey, uuid.NullUUID{}, 0, orgApiToken, req.Header, req.Avatar); err != nil {
		return fmt.Errorf("could not create organization: %w", err)
	}

	// Create the new account on the Vochain
	if err = u.vocClient.SetAccountInfo(ethSignKeys, metaURI); err != nil {
		return fmt.Errorf("could not create account on the vochain: %w", err)
	}

	resp.APIToken = orgApiToken
	resp.OrganizationID = ethSignKeys.Address().Bytes()

	return sendResponse(resp, ctx)
}

// GET https://server/v1/priv/account/organizations/<organizationId>
// getOrganizationPrivateHandler fetches an entity
func (u *URLAPI) getOrganizationPrivateHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	var err error
	var resp types.APIResponse
	var organizationMetadata *types.EntityMetadata
	var orgInfo orgPermissionsInfo
	var metaUri string
	// authenticate integrator has permission to edit this entity
	if orgInfo, err = u.authEntityPermissions(msg, ctx); err != nil {
		return err
	}

	// Fetch process from vochain
	if metaUri, _, _, err = u.vocClient.GetAccount(orgInfo.organization.EthAddress); err != nil {
		return err
	}

	// Fetch metadata
	if organizationMetadata, err = u.vocClient.FetchOrganizationMetadata(metaUri); err != nil {
		return fmt.Errorf("could not get organization metadata with URI\"%s\": %w", metaUri, err)
	}

	resp.APIToken = orgInfo.organization.PublicAPIToken
	resp.Name = organizationMetadata.Name["default"]
	resp.Description = organizationMetadata.Description["default"]
	resp.Avatar = organizationMetadata.Media.Avatar
	resp.Header = organizationMetadata.Media.Header
	return sendResponse(resp, ctx)
}

// DELETE https://server/v1/priv/account/organizations/<organizationId>
// deleteOrganizationHandler deletes an entity
func (u *URLAPI) deleteOrganizationHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	var err error
	var resp types.APIResponse
	var orgInfo orgPermissionsInfo

	// authenticate integrator has permission to edit this entity
	if orgInfo, err = u.authEntityPermissions(msg, ctx); err != nil {
		return err
	}

	if err = u.db.DeleteOrganization(orgInfo.integratorPrivKey, orgInfo.entityID); err != nil {
		log.Warn(err)
		return sendResponse(resp, ctx)
	}
	return sendResponse(resp, ctx)
}

// PATCH https://server/v1/account/organizations/<id>/key
// resetOrganizationKeyHandler resets an entity's api key
func (u *URLAPI) resetOrganizationKeyHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	var err error
	var resp types.APIResponse
	var orgInfo orgPermissionsInfo

	// authenticate integrator has permission to edit this entity
	if orgInfo, err = u.authEntityPermissions(msg, ctx); err != nil {
		return err
	}

	// Now generate a new api key & update integrator
	resp.APIToken = util.GenerateBearerToken()
	if _, err = u.db.UpdateOrganizationPublicAPIToken(
		orgInfo.integratorPrivKey, orgInfo.entityID, resp.APIToken); err != nil {
		return fmt.Errorf("could not update public api token %w", err)
	}
	return sendResponse(resp, ctx)
}

// PUT https://server/v1/priv/organizations/<organizationId>/metadata
// setOrganizationMetadataHandler sets an entity's metadata
func (u *URLAPI) setOrganizationMetadataHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	var err error
	var resp types.APIResponse
	var req types.APIRequest
	var orgInfo orgPermissionsInfo
	var metaURI string

	// authenticate integrator has permission to edit this entity
	if orgInfo, err = u.authEntityPermissions(msg, ctx); err != nil {
		return err
	}

	// Post metadata to ipfs
	if metaURI, err = u.vocClient.SetEntityMetadata(types.EntityMetadata{
		Version:     "1.0",
		Languages:   []string{},
		Name:        map[string]string{"default": req.Name},
		Description: map[string]string{"default": req.Description},
		NewsFeed:    map[string]string{},
		Media: types.EntityMedia{
			Avatar: req.Avatar,
			Header: req.Header,
		},
	}, orgInfo.entityID); err != nil {
		return fmt.Errorf("could not set entity metadata: %w", err)
	}

	// Update organization in the db to make sure it matches the metadata
	if _, err = u.db.UpdateOrganization(orgInfo.organization.IntegratorApiKey, orgInfo.organization.EthAddress,
		req.Header, req.Avatar); err != nil {
		return fmt.Errorf("could not update organization: %w", err)
	}

	resp.OrganizationID = orgInfo.entityID
	resp.ContentURI = metaURI
	return sendResponse(resp, ctx)
}

// POST https://server/v1/priv/organizations/<organizationId>/elections/signed
// POST https://server/v1/priv/organizations/<organizationId>/elections/blind
// createProcessHandler creates a process with the given metadata, either with signed or blind signature voting
func (u *URLAPI) createProcessHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	var err error
	var resp types.APIResponse
	var req types.APIRequest
	var orgInfo orgPermissionsInfo
	var processID []byte
	var metaUri string

	// TODO use blind/signed

	// ctx.URLParam("type")

	// authenticate integrator has permission to edit this entity
	if orgInfo, err = u.authEntityPermissions(msg, ctx); err != nil {
		return err
	}

	if req, err = util.UnmarshalRequest(msg); err != nil {
		return err
	}

	if req.Confidential {
		return fmt.Errorf("confidential processes are not yet supported")
	}

	pid := dvoteutil.RandomHex(32)
	if processID, err = hex.DecodeString(pid); err != nil {
		return fmt.Errorf("could not decode process ID: %w", err)
	}
	entityPrivKey, ok := util.DecryptSymmetric(orgInfo.organization.EthPrivKeyCicpher, orgInfo.integratorPrivKey)
	if !ok {
		return fmt.Errorf("could not decrypt entity private key")
	}
	entitySignKeys := ethereum.NewSignKeys()
	if err = entitySignKeys.AddHexKey(hex.EncodeToString(entityPrivKey)); err != nil {
		return fmt.Errorf("could not decode entity private key: %w", err)
	}

	startDate, err := time.Parse("2006-01-02T15:04:05.000Z", req.StartDate)
	if err != nil {
		return fmt.Errorf("could not parse startDate: %w", err)
	}
	endDate, err := time.Parse("2006-01-02T15:04:05.000Z", req.EndDate)
	if err != nil {
		return fmt.Errorf("could not parse startDate: %w", err)
	}

	now := time.Now()
	if startDate.Before(now) || endDate.Before(now) {
		return fmt.Errorf("election start and end date cannot be in the past")
	}
	if endDate.Before(startDate) {
		return fmt.Errorf("end date must be after start date")
	}
	startBlock, err := u.estimateBlockHeight(startDate)
	if err != nil {
		return fmt.Errorf("unable to estimate startDate block height: %w", err)
	}
	endBlock, err := u.estimateBlockHeight(endDate)
	if err != nil {
		return fmt.Errorf("unable to estimate endDate block height: %w", err)
	}
	log.Debugf("start block %d end block %d", startBlock, endBlock)

	metadata := types.ProcessMetadata{
		Description: map[string]string{"default": req.Description},
		Media: types.ProcessMedia{
			Header:    req.Header,
			StreamURI: req.StreamURI,
		},
		Meta:      metaUri,
		Questions: []types.QuestionMeta{},
		Results: types.ProcessResultsDetails{
			Aggregation: "discrete-values",
			Display:     "multiple-choice",
		},
		Title:   map[string]string{"default": req.Title},
		Version: "1.0",
	}

	envelopeType := &models.EnvelopeType{
		Serial:         false,
		Anonymous:      false,
		EncryptedVotes: req.HiddenResults,
		UniqueValues:   false,
		CostFromWeight: false,
	}
	processMode := &models.ProcessMode{
		AutoStart:         false,
		Interruptible:     true,
		DynamicCensus:     false,
		EncryptedMetaData: req.Confidential,
		PreRegister:       false,
	}

	maxChoiceValue := 0
	for _, question := range req.Questions {
		if len(question.Choices) > maxChoiceValue {
			maxChoiceValue = len(question.Choices)
		}
		metaQuestion := types.QuestionMeta{
			Choices:     []types.Choice{},
			Description: map[string]string{"default": question.Description},
			Title:       map[string]string{"default": question.Title},
		}
		for i, choice := range question.Choices {
			metaQuestion.Choices = append(metaQuestion.Choices, types.Choice{
				Title: map[string]string{"default": choice},
				Value: uint32(i),
			})
		}
		metadata.Questions = append(metadata.Questions, metaQuestion)
	}
	log.Debugf("req questions: %v", req.Questions)
	log.Debugf("meta qustions: %v", metadata.Questions)

	voteOptions := &models.ProcessVoteOptions{
		MaxCount:          uint32(len(req.Questions)),
		MaxValue:          uint32(maxChoiceValue),
		MaxVoteOverwrites: 0,
		MaxTotalCost:      uint32(len(req.Questions) * maxChoiceValue),
		CostExponent:      1,
	}

	if metaUri, err = u.vocClient.SetProcessMetadata(metadata, processID); err != nil {
		return fmt.Errorf("could not set process metadata: %w", err)
	}

	var MAX_CENSUS_SIZE = uint64(1024)

	// TODO use encryption priv/pub keys if process is encrypted
	if startBlock, err = u.vocClient.CreateProcess(&models.Process{
		ProcessId:     processID,
		EntityId:      orgInfo.entityID,
		StartBlock:    startBlock,
		BlockCount:    endBlock - startBlock,
		CensusRoot:    []byte{},
		CensusURI:     new(string),
		Status:        models.ProcessStatus_READY,
		EnvelopeType:  envelopeType,
		Mode:          processMode,
		VoteOptions:   voteOptions,
		CensusOrigin:  models.CensusOrigin_OFF_CHAIN_CA,
		Metadata:      &metaUri,
		MaxCensusSize: &MAX_CENSUS_SIZE,
	}, entitySignKeys); err != nil {
		return fmt.Errorf("could not create process on the vochain: %w", err)
	}

	if _, err = u.db.CreateElection(orgInfo.integratorPrivKey, orgInfo.entityID, processID, req.Title, startDate,
		endDate, uuid.NullUUID{}, int(startBlock), int(endBlock), req.Confidential, req.HiddenResults); err != nil {
		return fmt.Errorf("could not create election: %w", err)
	}
	resp.ElectionID = processID
	return sendResponse(resp, ctx)
}

// GET https://server/v1/priv/organizations/<organizationId>/elections/signed
// GET https://server/v1/priv/organizations/<organizationId>/elections/blind
// GET https://server/v1/priv/organizations/<organizationId>/elections/active
// GET https://server/v1/pprivub/organizations/<organizationId>/elections/ended
// GET https://server/v1/priv/organizations/<organizationId>/elections/upcoming
// listProcessesPrivateHandler' lists signed, blind, active, ended, or upcoming processes
func (u *URLAPI) listProcessesPrivateHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	var orgInfo orgPermissionsInfo
	var pub []types.APIElectionSummary
	var err error

	if orgInfo, err = u.authEntityPermissions(msg, ctx); err != nil {
		return err
	}

	filter := ctx.URLParam("type")
	if pub, _, err = u.getProcessList(filter, orgInfo.integratorPrivKey, orgInfo.entityID, true); err != nil {
		return err
	}
	return sendResponse(pub, ctx)
}

// GET https://server/v1/priv/elections/<processId>
// getProcessHandler gets the entirety of a process, including metadata
// confidential processes need no extra step, only the api key
func (u *URLAPI) getProcessHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var err error
	var resp types.APIElectionInfo
	var processId []byte
	var vochainProcess *indexertypes.Process
	var results *types.VochainResults
	var processMetadata *types.ProcessMetadata
	if processId, err = util.GetBytesID(ctx, "electionId"); err != nil {
		return err
	}

	// Fetch process from vochain
	if vochainProcess, err = u.vocClient.GetProcess(processId); err != nil {
		return fmt.Errorf("unable to fetch process from the vochain: %w", err)
	}

	// Fetch results
	if vochainProcess.HaveResults {
		if results, err = u.vocClient.GetResults(processId); err != nil {
			return fmt.Errorf("could not get results: %w", err)
		}
	}

	// Fetch metadata
	metadataUri := vochainProcess.Metadata
	if processMetadata, err = u.vocClient.FetchProcessMetadata(metadataUri); err != nil {
		return fmt.Errorf("could not get process metadata: %w", err)
	}

	// Parse all the information
	if resp, err = u.parseProcessInfo(vochainProcess, results, processMetadata); err != nil {
		return fmt.Errorf("could not parse information for process %x: %w", processId, err)
	}

	return sendResponse(resp, ctx)
}

// POST https://server/v1/priv/censuses
// createCensusHandler creates a census where public keys or token slots (that will eventually contain a public key) are stored.
// A census can start with 0 items, and public keys can be imported later on.
// If census tokens are allocated, users will need to generate a wallet on the frontend and register the public key by themselves.
// This prevents both the API and the integrator from gaining access to the private key.
func (u *URLAPI) createCensusHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// POST https://server/v1/priv/censuses/<censusId>/tokens/flat
// POST https://server/v1/priv/censuses/<censusId>/tokens/weighted
// addCensusTokensHandler adds N (weight 1 or weighted) census tokens for voters to register their public keys
func (u *URLAPI) addCensusTokensHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// GET https://server/v1/priv/censuses/<censusId>/tokens/<tokenId>
// getCensusTokenHandler gets the given census token with weight and assigned public key, if applicable
func (u *URLAPI) getCensusTokenHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// DELETE https://server/v1/priv/censuses/<censusId>/tokens/<tokenId>
// deleteCensusTokenHandler deletes the given token(s) from the given census
func (u *URLAPI) deleteCensusTokenHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// DELETE https://server/v1/priv/censuses/<censusId>/keys/<publicKey>
// deletePublicKeyHandler deletes the given public key(s) from the given census
func (u *URLAPI) deletePublicKeyHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// POST https://server/v1/priv/censuses/<censusId>/import/flat
// POST https://server/v1/priv/censuses/<censusId>/import/weighted
// importPublicKeysHandler imports a group of public keys into the existing census, weighted or weight 1
func (u *URLAPI) importPublicKeysHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// PUT https://server/v1/priv/elections/<processId>/status
// setProcessStatusHandler sets the process status (READY, PAUSED, ENDED, CANCELED)
func (u *URLAPI) setProcessStatusHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}
