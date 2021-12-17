package urlapi

import (
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.vocdoni.io/api/types"
	"go.vocdoni.io/api/util"
	"go.vocdoni.io/api/vocclient"
	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/httprouter/bearerstdapi"
	"go.vocdoni.io/dvote/log"
	dvoteutil "go.vocdoni.io/dvote/util"
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
		"/priv/organizations/{organizationId}/elections/{type}",
		"GET",
		bearerstdapi.MethodAccessTypePrivate,
		u.listProcessesPrivateHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/organizations/{organizationId}/elections",
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
		"/priv/elections/{electionId}/{status}",
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
	if err := u.api.RegisterMethod(
		"/priv/transactions/{transactionHash}",
		"GET",
		bearerstdapi.MethodAccessTypePrivate,
		u.getTxStatusHandler,
	); err != nil {
		return err
	}
	return nil
}

// POST https://server/v1/priv/account/organizations
// createOrganizationHandler creates a new entity
func (u *URLAPI) createOrganizationHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	// var organizationMetadataKey []byte
	req, err := util.UnmarshalRequest(msg)
	if err != nil {
		return err
	}
	integratorPrivKey, err := util.GetAuthToken(msg)
	if err != nil {
		return err
	}
	orgApiToken := util.GenerateBearerToken()

	ethSignKeys := ethereum.NewSignKeys()
	if err = ethSignKeys.Generate(); err != nil {
		return fmt.Errorf("could not generate ethereum keys: %w", err)
	}

	// Encrypt private key to store in db
	_, priv := ethSignKeys.HexString()
	entityPrivKey, err := hex.DecodeString(priv)
	if err != nil {
		return fmt.Errorf("could not decode entity private key: %w", err)
	}

	// If there is a global entity encryption key that can be decoded,
	//  use it to encrypt the entityPrivKey before storing
	encryptedPrivKey := entityPrivKey
	if len(u.globalOrganizationKey) > 0 {
		if encryptedPrivKey, err = util.EncryptSymmetric(
			entityPrivKey, u.globalOrganizationKey); err != nil {
			return fmt.Errorf("could not encrypt entity private key: %w", err)
		}
	}

	// Post metadata to ipfs
	metaURI, err := u.vocClient.SetEntityMetadata(types.EntityMetadata{
		Version:     "1.0",
		Languages:   []string{},
		Name:        map[string]string{"default": req.Name},
		Description: map[string]string{"default": req.Description},
		NewsFeed:    map[string]string{},
		Media: types.EntityMedia{
			Avatar: req.Avatar,
			Header: req.Header,
		},
	}, ethSignKeys.Address().Bytes())
	if err != nil {
		return fmt.Errorf("could not set entity metadata: %w", err)
	}

	// Create the new account on the Vochain
	if err = u.vocClient.SetAccountInfo(ethSignKeys, metaURI); err != nil {
		return fmt.Errorf("could not create account on the vochain: %w", err)
	}

	// TODO fetch actual transaction hash
	txHash := dvoteutil.RandomHex(32)
	u.txWaitMap.Store(txHash, time.Now())
	u.dbTransactions.Store(txHash, createOrganizationQuery{
		integratorPrivKey: integratorPrivKey,
		ethAddress:        ethSignKeys.Address().Bytes(),
		ethPrivKeyCipher:  encryptedPrivKey,
		planID:            uuid.NullUUID{},
		publicApiQuota:    0,
		publicApiToken:    orgApiToken,
		headerUri:         req.Header,
		avatarUri:         req.Avatar,
	})

	resp := types.APIResponse{APIToken: orgApiToken, OrganizationID: ethSignKeys.Address().Bytes(), TxHash: txHash}

	return sendResponse(resp, ctx)
}

// GET https://server/v1/priv/account/organizations/<organizationId>
// getOrganizationPrivateHandler fetches an entity
func (u *URLAPI) getOrganizationPrivateHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	// authenticate integrator has permission to edit this entity
	orgInfo, err := u.authEntityPermissions(msg, ctx)
	if err != nil {
		return err
	}

	// Fetch process from vochain
	metaUri, _, _, err := u.vocClient.GetAccount(orgInfo.organization.EthAddress)
	if err != nil {
		return err
	}

	// Fetch metadata
	organizationMetadata, err := u.vocClient.FetchOrganizationMetadata(metaUri)
	if err != nil {
		return fmt.Errorf("could not get organization metadata with URI\"%s\": %w", metaUri, err)
	}

	resp := types.APIResponse{
		APIToken:    orgInfo.organization.PublicAPIToken,
		Name:        organizationMetadata.Name["default"],
		Description: organizationMetadata.Description["default"],
		Avatar:      organizationMetadata.Media.Avatar,
		Header:      organizationMetadata.Media.Header,
	}
	return sendResponse(resp, ctx)
}

// DELETE https://server/v1/priv/account/organizations/<organizationId>
// deleteOrganizationHandler deletes an entity
func (u *URLAPI) deleteOrganizationHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	// authenticate integrator has permission to edit this entity
	orgInfo, err := u.authEntityPermissions(msg, ctx)
	if err != nil {
		return err
	}

	if err = u.db.DeleteOrganization(orgInfo.integratorPrivKey, orgInfo.entityID); err != nil {
		log.Warn(err)
		return sendResponse(types.APIResponse{}, ctx)
	}
	return sendResponse(types.APIResponse{}, ctx)
}

// PATCH https://server/v1/account/organizations/<id>/key
// resetOrganizationKeyHandler resets an entity's api key
func (u *URLAPI) resetOrganizationKeyHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	// authenticate integrator has permission to edit this entity
	orgInfo, err := u.authEntityPermissions(msg, ctx)
	if err != nil {
		return err
	}

	// Now generate a new api key & update integrator
	resp := types.APIResponse{APIToken: util.GenerateBearerToken()}
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
	// authenticate integrator has permission to edit this entity
	orgInfo, err := u.authEntityPermissions(msg, ctx)
	if err != nil {
		return err
	}
	req, err := util.UnmarshalRequest(msg)
	if err != nil {
		return err
	}
	// Post metadata to ipfs
	metaURI, err := u.vocClient.SetEntityMetadata(types.EntityMetadata{
		Version:     "1.0",
		Languages:   []string{},
		Name:        map[string]string{"default": req.Name},
		Description: map[string]string{"default": req.Description},
		NewsFeed:    map[string]string{},
		Media: types.EntityMedia{
			Avatar: req.Avatar,
			Header: req.Header,
		},
	}, orgInfo.entityID)
	if err != nil {
		return fmt.Errorf("could not set entity metadata: %w", err)
	}

	entitySignKeys, err := decryptEntityKeys(
		orgInfo.organization.EthPrivKeyCipher, u.globalOrganizationKey)
	if err != nil {
		return err
	}
	if err := u.vocClient.SetAccountInfo(entitySignKeys, metaURI); err != nil {
		return fmt.Errorf("could not update account metadata uri: %w", err)
	}

	// TODO fetch actual transaction hash
	txHash := dvoteutil.RandomHex(32)
	u.txWaitMap.Store(txHash, time.Now())
	u.dbTransactions.Store(txHash,
		updateOrganizationQuery{
			integratorPrivKey: orgInfo.organization.IntegratorApiKey,
			ethAddress:        orgInfo.organization.EthAddress,
			headerUri:         req.Header,
			avatarUri:         req.Avatar,
		})
	resp := types.APIResponse{
		OrganizationID: orgInfo.entityID,
		ContentURI:     metaURI,
		TxHash:         txHash,
	}
	return sendResponse(resp, ctx)
}

// POST https://server/v1/priv/organizations/<organizationId>/elections/signed
// POST https://server/v1/priv/organizations/<organizationId>/elections/blind
// createProcessHandler creates a process with
//  the given metadata, either with signed or blind signature voting
func (u *URLAPI) createProcessHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	// TODO use blind/signed
	// authenticate integrator has permission to edit this entity
	orgInfo, err := u.authEntityPermissions(msg, ctx)
	if err != nil {
		return err
	}

	req, err := util.UnmarshalRequest(msg)
	if err != nil {
		return err
	}

	processID, err := hex.DecodeString(dvoteutil.RandomHex(32))
	if err != nil {
		return fmt.Errorf("could not decode process ID: %w", err)
	}
	entitySignKeys, err := decryptEntityKeys(
		orgInfo.organization.EthPrivKeyCipher, u.globalOrganizationKey)
	if err != nil {
		return err
	}

	var startBlock uint32
	startDate := time.Unix(0, 0)
	// If start date is empty, do not attempt to parse it. Set startBlock to 0, starting the
	//  process immediately. Otherwise, ensure the startBlock is in the future
	if req.StartDate != "" {
		if startDate, err = time.Parse("2006-01-02T15:04:05.000Z", req.StartDate); err != nil {
			return fmt.Errorf("could not parse startDate: %w", err)
		}
		if startBlock, err = u.estimateBlockHeight(startDate); err != nil {
			return fmt.Errorf("unable to estimate startDate block height: %w", err)
		}
	}

	endDate, err := time.Parse("2006-01-02T15:04:05.000Z", req.EndDate)
	if err != nil {
		return fmt.Errorf("could not parse endDate: %w", err)
	}

	if endDate.Before(time.Now()) {
		return fmt.Errorf("election end date cannot be in the past")
	}
	endBlock, err := u.estimateBlockHeight(endDate)
	if err != nil {
		return fmt.Errorf("unable to estimate endDate block height: %w", err)
	}
	if endDate.Before(startDate) {
		return fmt.Errorf("end date must be after start date")
	}

	metadata := types.ProcessMetadata{
		Description: map[string]string{"default": req.Description},
		Media: types.ProcessMedia{
			Header:    req.Header,
			StreamURI: req.StreamURI,
		},
		Meta:      nil,
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

	voteOptions := &models.ProcessVoteOptions{
		MaxCount:          uint32(len(req.Questions)),
		MaxValue:          uint32(maxChoiceValue),
		MaxVoteOverwrites: 0,
		MaxTotalCost:      uint32(len(req.Questions) * maxChoiceValue),
		CostExponent:      1,
	}

	var metaUri string
	var metaPrivKeyBytes []byte
	var encryptedMetaKey []byte
	// If election is confidential, generate a private metadata key and encrypt it.
	// store this key with the election
	if req.Confidential {
		metadataPrivKey := dvoteutil.RandomHex(32)
		if metaPrivKeyBytes, err = hex.DecodeString(metadataPrivKey); err != nil {
			return fmt.Errorf("could not decode metadata private key: %w", err)
		}
		encryptedMetaKey = metaPrivKeyBytes
		// If there is a global meta key, encrypt the meta priv key
		if len(u.globalMetadataKey) > 0 {
			if encryptedMetaKey, err = util.EncryptSymmetric(
				metaPrivKeyBytes, u.globalMetadataKey); err != nil {
				return fmt.Errorf("could not encrypt metadata private key: %w", err)
			}
		}

		if metaUri, err = u.vocClient.SetProcessMetadataConfidential(
			metadata, metaPrivKeyBytes, processID); err != nil {
			return fmt.Errorf("could not set confidential process metadata: %w", err)
		}
	} else {
		if metaUri, err = u.vocClient.SetProcessMetadata(metadata, processID); err != nil {
			return fmt.Errorf("could not set process metadata: %w", err)
		}
	}

	integrator, err := u.db.GetIntegratorByKey(orgInfo.integratorPrivKey)
	if err != nil {
		return fmt.Errorf("could not retrieve integrator from db: %w", err)
	}

	currentBlockHeight, avgTimes, _ := u.vocClient.GetBlockTimes()
	if startBlock > 1 && startBlock < currentBlockHeight+vocclient.VOCHAIN_BLOCK_MARGIN {
		return fmt.Errorf("cannot create process: startDate needs to be at least %ds in the future", vocclient.VOCHAIN_BLOCK_MARGIN*avgTimes[0]/1000)
	}

	if startBlock, err = u.vocClient.CreateProcess(&models.Process{
		ProcessId:     processID,
		EntityId:      orgInfo.entityID,
		StartBlock:    startBlock,
		BlockCount:    endBlock - startBlock,
		CensusRoot:    integrator.CspPubKey,
		CensusURI:     new(string),
		Status:        models.ProcessStatus_READY,
		EnvelopeType:  envelopeType,
		Mode:          processMode,
		VoteOptions:   voteOptions,
		CensusOrigin:  models.CensusOrigin_OFF_CHAIN_CA,
		Metadata:      &metaUri,
		MaxCensusSize: &u.config.MaxCensusSize,
	}, entitySignKeys); err != nil {
		return fmt.Errorf("could not create process on the vochain: %w", err)
	}

	// TODO fetch actual transaction hash
	txHash := dvoteutil.RandomHex(32)
	u.txWaitMap.Store(txHash, time.Now())
	u.dbTransactions.Store(txHash, createElectionQuery{
		integratorPrivKey: orgInfo.integratorPrivKey,
		ethAddress:        orgInfo.entityID,
		encryptedMetaKey: encryptedMetaKey,
		electionID:        processID,
		title:             req.Title,
		startDate:         startDate,
		endDate:           endDate,
		censusID:          uuid.NullUUID{},
		startBlock:        int(startBlock),
		endBlock:          int(endBlock),
		confidential:      req.Confidential,
		hiddenResults:     req.HiddenResults,
	})
	return sendResponse(types.APIResponse{ElectionID: processID, TxHash: txHash}, ctx)
}

// GET https://server/v1/priv/organizations/<organizationId>/elections/signed
// GET https://server/v1/priv/organizations/<organizationId>/elections/blind
// GET https://server/v1/priv/organizations/<organizationId>/elections/active
// GET https://server/v1/pprivub/organizations/<organizationId>/elections/ended
// GET https://server/v1/priv/organizations/<organizationId>/elections/upcoming
// listProcessesPrivateHandler' lists signed, blind, active, ended, or upcoming processes
func (u *URLAPI) listProcessesPrivateHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {

	orgInfo, err := u.authEntityPermissions(msg, ctx)
	if err != nil {
		return err
	}

	list, err := u.getProcessList(ctx.URLParam("type"),
		orgInfo.integratorPrivKey, orgInfo.entityID, true)
	if err != nil {
		return err
	}
	return sendResponse(list, ctx)
}

// GET https://server/v1/priv/elections/<processId>
// getProcessHandler gets the entirety of a process, including metadata
// confidential processes need no extra step, only the api key
func (u *URLAPI) getProcessHandler(
	msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	processId, err := util.GetBytesID(ctx, "electionId")
	if err != nil {
		return err
	}

	// Fetch process from vochain
	vochainProcess, err := u.vocClient.GetProcess(processId)
	if err != nil {
		return fmt.Errorf("unable to fetch process from the vochain: %w", err)
	}
	if vochainProcess == nil {
		return fmt.Errorf("process does not exist")
		
	integratorApiKey, err := hex.DecodeString(msg.AuthToken)
	if err != nil {
		return fmt.Errorf("could not decode bearer token: %w", err)
	}

	// Fetch election from database
	dbElection, err := u.db.GetElection(integratorApiKey, vochainProcess.EntityID, processId)
	if err != nil {
		return fmt.Errorf("could not get election from the database")
	}

	// Fetch results
	var results *types.VochainResults
	if vochainProcess.HaveResults {
		if results, err = u.vocClient.GetResults(processId); err != nil {
			return fmt.Errorf("could not get results: %w", err)
		}
	}

	// Fetch metadata
	var processMetadata *types.ProcessMetadata
	if dbElection.Confidential {
		metaKey := dbElection.MetadataPrivKey
		// If globalMetadataKey exists, try to decrypt metadata private key
		if len(u.globalMetadataKey) > 0 {
			var ok bool
			metaKey, ok = util.DecryptSymmetric(dbElection.MetadataPrivKey, u.globalMetadataKey)
			if !ok {
				return fmt.Errorf("could not decrypt election private metadata key")
			}
		}
		if processMetadata, err = u.vocClient.FetchProcessMetadataConfidential(
			vochainProcess.Metadata, metaKey); err != nil {
			return fmt.Errorf("could not get process metadata: %w", err)
		}
	} else {
		if processMetadata, err = u.vocClient.FetchProcessMetadata(vochainProcess.Metadata); err != nil {
			return fmt.Errorf("could not get process metadata: %w", err)
		}
	}
	// Parse all the information
	resp, err := u.parseProcessInfo(vochainProcess, results, processMetadata)
	if err != nil {
		return fmt.Errorf("could not parse information for process %x: %w", processId, err)
	}

	return sendResponse(resp, ctx)
}

// POST https://server/v1/priv/censuses
// createCensusHandler creates a census where public keys or
//  token slots (that will eventually contain a public key) are stored.
// A census can start with 0 items, and public keys can be imported later on.
// If census tokens are allocated, users will need to generate a wallet on
//  the frontend and register the public key by themselves.
// This prevents both the API and the integrator from gaining access to the private key.
func (u *URLAPI) createCensusHandler(
	msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// POST https://server/v1/priv/censuses/<censusId>/tokens/flat
// POST https://server/v1/priv/censuses/<censusId>/tokens/weighted
// addCensusTokensHandler adds N (weight 1 or weighted)
//  census tokens for voters to register their public keys
func (u *URLAPI) addCensusTokensHandler(
	msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// GET https://server/v1/priv/censuses/<censusId>/tokens/<tokenId>
// getCensusTokenHandler gets the given census
//  token with weight and assigned public key, if applicable
func (u *URLAPI) getCensusTokenHandler(
	msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// DELETE https://server/v1/priv/censuses/<censusId>/tokens/<tokenId>
// deleteCensusTokenHandler deletes the given token(s) from the given census
func (u *URLAPI) deleteCensusTokenHandler(
	msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// DELETE https://server/v1/priv/censuses/<censusId>/keys/<publicKey>
// deletePublicKeyHandler deletes the given public key(s) from the given census
func (u *URLAPI) deletePublicKeyHandler(
	msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// POST https://server/v1/priv/censuses/<censusId>/import/flat
// POST https://server/v1/priv/censuses/<censusId>/import/weighted
// importPublicKeysHandler imports a group of public keys
//  into the existing census, weighted or weight 1
func (u *URLAPI) importPublicKeysHandler(
	msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// PUT https://server/v1/priv/elections/<electionId>/status
// setProcessStatusHandler sets the process status (READY, PAUSED, ENDED, CANCELED)
func (u *URLAPI) setProcessStatusHandler(
	msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	processID, err := util.GetBytesID(ctx, "electionId")
	if err != nil {
		return fmt.Errorf("could not get electionId: %w", err)
	}
	process, err := u.vocClient.GetProcess(processID)
	if err != nil {
		return fmt.Errorf("could not fetch election %x from the vochain: %w", processID, err)
	}
	integratorPrivKey, err := util.GetAuthToken(msg)
	if err != nil {
		return fmt.Errorf("could not get integrator api token: %w", err)
	}
	organization, err := u.db.GetOrganization(integratorPrivKey, process.EntityID)
	if err != nil {
		return fmt.Errorf("organization %X could not be fetched from the db: %w",
			organization.EthAddress, err)
	}
	entityPrivKey := organization.EthPrivKeyCipher
	if len(u.globalOrganizationKey) > 0 {
		var ok bool
		if entityPrivKey, ok = util.DecryptSymmetric(
			organization.EthPrivKeyCipher, u.globalOrganizationKey); !ok {
			return fmt.Errorf("could not decrypt entity private key")
		}
	}
	entitySignKeys := ethereum.NewSignKeys()
	if err = entitySignKeys.AddHexKey(hex.EncodeToString(entityPrivKey)); err != nil {
		return fmt.Errorf("could not convert entity private key to signKey: %w", err)
	}

	var status models.ProcessStatus
	switch strings.ToUpper(ctx.URLParam("status")) {
	case "READY":
		status = models.ProcessStatus_READY
	case "PAUSED":
		status = models.ProcessStatus_PAUSED
	case "ENDED":
		status = models.ProcessStatus_ENDED
	case "CANCELLED":
		status = models.ProcessStatus_CANCELED
	}

	if err = u.vocClient.SetProcessStatus(processID, &status, entitySignKeys); err != nil {
		return fmt.Errorf("could not set process status %d: %w", status, err)
	}

	// TODO fetch actual transaction hash
	txHash := dvoteutil.RandomHex(32)
	u.txWaitMap.Store(txHash, time.Now())

	return sendResponse(types.APIResponse{TxHash: txHash}, ctx)
}
func decryptEntityKeys(privKeyCipher, globalOrganizationKey []byte) (*ethereum.SignKeys, error) {
	entityPrivKey := privKeyCipher
	if len(globalOrganizationKey) > 0 {
		var ok bool
		if entityPrivKey, ok = util.DecryptSymmetric(
			privKeyCipher, globalOrganizationKey); !ok {
			return nil, fmt.Errorf("could not decrypt entity private key")
		}
	}
	entitySignKeys := ethereum.NewSignKeys()
	if err := entitySignKeys.AddHexKey(hex.EncodeToString(entityPrivKey)); err != nil {
		return nil, fmt.Errorf("could not convert entity private key to signKey: %w", err)
	}
	return entitySignKeys, nil
}
