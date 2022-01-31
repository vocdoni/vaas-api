package urlapi

import (
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.vocdoni.io/api/database/transactions"
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

// IMMEDIATE_PROCESS_CREATION_OFFSET is the average number of blocks it takes to create a
//  process when its startBlock is set to 0. This is only to be used for estimating timings.
const IMMEDIATE_PROCESS_CREATION_OFFSET = 3

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
	if req.Name == "" {
		return fmt.Errorf("organization name is empty")
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
	txHash := dvoteutil.RandomBytes(32)
	u.kv.StoreTxTime(txHash, time.Now())
	queryTx := transactions.SerializableTx{
		Type:         transactions.CreateOrganization,
		CreationTime: time.Now(),
		Body: transactions.CreateOrganizationTx{
			IntegratorPrivKey: integratorPrivKey,
			EthAddress:        ethSignKeys.Address().Bytes(),
			EthPrivKeyCipher:  encryptedPrivKey,
			PlanID:            uuid.NullUUID{},
			PublicAPIQuota:    0,
			PublicAPIToken:    orgApiToken,
			HeaderURI:         req.Header,
			AvatarURI:         req.Avatar,
		},
	}
	if err = u.kv.StoreTx(txHash, queryTx); err != nil {
		return err
	}

	resp := types.APIResponse{APIToken: orgApiToken,
		OrganizationID: ethSignKeys.Address().Bytes(), TxHash: txHash}

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
	txHash := dvoteutil.RandomBytes(32)
	u.kv.StoreTxTime(txHash, time.Now())
	queryTx := transactions.SerializableTx{
		Type:         transactions.UpdateOrganization,
		CreationTime: time.Now(),
		Body: transactions.UpdateOrganizationTx{
			IntegratorPrivKey: orgInfo.organization.IntegratorApiKey,
			EthAddress:        orgInfo.organization.EthAddress,
			HeaderUri:         req.Header,
			AvatarUri:         req.Avatar,
		},
	}
	if err = u.kv.StoreTx([]byte(txHash), queryTx); err != nil {
		return err
	}
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
	// authenticate integrator has permission to edit this entity
	orgInfo, err := u.authEntityPermissions(msg, ctx)
	if err != nil {
		return err
	}

	electionType := types.ProofType(ctx.URLParam("type"))
	switch electionType {
	case types.PROOF_TYPE_BLIND, types.PROOF_TYPE_ECDSA:
	default:
		return fmt.Errorf("election proof type %s is invalid", electionType)
	}

	req, err := util.UnmarshalRequest(msg)
	if err != nil {
		return err
	}

	processID := dvoteutil.RandomBytes(32)
	entitySignKeys, err := decryptEntityKeys(
		orgInfo.organization.EthPrivKeyCipher, u.globalOrganizationKey)
	if err != nil {
		return err
	}

	var startBlock uint32
	startDate := time.Now()
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
	// If election is confidential, generate a private metadata key and encrypt it.
	// store this key with the election
	if req.Confidential {
		metaPrivKeyBytes = dvoteutil.RandomBytes(32)
		// Encrypt and send the process metadata
		if metaUri, err = u.vocClient.SetProcessMetadata(
			metadata, processID, metaPrivKeyBytes); err != nil {
			return fmt.Errorf("could not set confidential process metadata: %w", err)
		}

		// If there is a global meta key, encrypt the meta priv key
		if len(u.globalMetadataKey) > 0 {
			if metaPrivKeyBytes, err = util.EncryptSymmetric(
				metaPrivKeyBytes, u.globalMetadataKey); err != nil {
				return fmt.Errorf("could not encrypt metadata private key: %w", err)
			}
		}

	} else { // Process is not confidential, no need to touch metadata key
		if metaUri, err = u.vocClient.SetProcessMetadata(metadata, processID, []byte{}); err != nil {
			return fmt.Errorf("could not set process metadata: %w", err)
		}
	}

	integrator, err := u.db.GetIntegratorByKey(orgInfo.integratorPrivKey)
	if err != nil {
		return fmt.Errorf("could not retrieve integrator from db: %w", err)
	}

	currentBlockHeight, avgTimes, _ := u.vocClient.GetBlockTimes()
	if startBlock > 1 && startBlock < currentBlockHeight+vocclient.VOCHAIN_BLOCK_MARGIN {
		return fmt.Errorf("cannot create process: startDate needs to be at least %ds in the future",
			vocclient.VOCHAIN_BLOCK_MARGIN*avgTimes[0]/1000)
	}

	blockCount := endBlock - startBlock
	if startBlock == 0 {
		// If startBlock is set to 0 (process starts asap), set the blockcount to the desired
		//  end block, minus the expected start block of the process
		blockCount = blockCount - currentBlockHeight + 3
	}
	if err = u.vocClient.CreateProcess(&models.Process{
		ProcessId:     processID,
		EntityId:      orgInfo.entityID,
		StartBlock:    startBlock,
		BlockCount:    blockCount,
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

	// If starting immediately, store current block height as startblock in db
	if startBlock <= 1 {
		startBlock = currentBlockHeight + IMMEDIATE_PROCESS_CREATION_OFFSET
	}

	// TODO fetch actual transaction hash
	txHash := dvoteutil.RandomBytes(32)
	u.kv.StoreTxTime(txHash, time.Now().Add(time.Duration(2*int(avgTimes[0]))))
	queryTx := transactions.SerializableTx{
		Type:         transactions.CreateElection,
		CreationTime: time.Now().Add(time.Duration(2 * int(avgTimes[0]))),
		Body: transactions.CreateElectionTx{
			IntegratorPrivKey: orgInfo.integratorPrivKey,
			EthAddress:        orgInfo.entityID,
			EncryptedMetaKey:  metaPrivKeyBytes,
			ElectionID:        processID,
			Title:             req.Title,
			StartDate:         startDate,
			EndDate:           endDate,
			CensusID:          uuid.NullUUID{},
			StartBlock:        startBlock,
			EndBlock:          startBlock + blockCount,
			Confidential:      req.Confidential,
			HiddenResults:     req.HiddenResults,
		},
	}
	if err = u.kv.StoreTx(txHash, queryTx); err != nil {
		return err
	}

	return sendResponse(types.APIResponse{
		ElectionID: processID, TxHash: txHash}, ctx)
}

// GET https://server/v1/priv/organizations/<organizationId>/elections/signed
// GET https://server/v1/priv/organizations/<organizationId>/elections/blind
// GET https://server/v1/priv/organizations/<organizationId>/elections/paused
// GET https://server/v1/priv/organizations/<organizationId>/elections/canceled
// GET https://server/v1/priv/organizations/<organizationId>/elections/active
// GET https://server/v1/priv/organizations/<organizationId>/elections/upcoming
// GET https://server/v1/priv/organizations/<organizationId>/elections/ended
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
	}

	integratorApiKey, err := hex.DecodeString(msg.AuthToken)
	if err != nil {
		return fmt.Errorf("could not decode bearer token: %w", err)
	}

	// Fetch election from database
	dbElection, err := u.db.GetElection(integratorApiKey, vochainProcess.EntityID, processId)
	if err != nil {
		return fmt.Errorf("could not get election from the database: %w", err)
	}

	// Fetch results
	var results *types.VochainResults
	if vochainProcess.HaveResults {
		if results, err = u.vocClient.GetResults(processId); err != nil {
			return fmt.Errorf("could not get results: %w", err)
		}
	}

	// Fetch metadata
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
	case "CANCELED":
		status = models.ProcessStatus_CANCELED
	}

	if err = u.vocClient.SetProcessStatus(processID, &status, entitySignKeys); err != nil {
		return fmt.Errorf("could not set process status %d: %w", status, err)
	}

	// TODO fetch actual transaction hash
	txHash := dvoteutil.RandomBytes(32)
	if err = u.kv.StoreTxTime([]byte(txHash), time.Now()); err != nil {
		return err
	}

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

// Helper function to get process metadata, confidential or not.
func (u *URLAPI) getProcessMetadataPriv(confidential bool,
	metadataPrivKey []byte, uri string) (*types.ProcessMetadata, error) {
	// If election is confidential, fetch private metadata key & decrypt metadata
	var processMetadata *types.ProcessMetadata
	var err error
	if confidential {
		// If globalMetadataKey exists, try to decrypt metadata private key
		if len(u.globalMetadataKey) > 0 {
			var ok bool
			metadataPrivKey, ok = util.DecryptSymmetric(metadataPrivKey, u.globalMetadataKey)
			if !ok {
				return nil, fmt.Errorf("could not decrypt election private metadata key")
			}
		}
		if processMetadata, err = u.vocClient.FetchProcessMetadataConfidential(
			uri, metadataPrivKey); err != nil {
			return nil, fmt.Errorf("could not get process metadata: %w", err)
		}
	} else { // Election is not confidential, no need to decrypt metadata
		if processMetadata, err = u.vocClient.FetchProcessMetadata(uri); err != nil {
			return nil, fmt.Errorf("could not get process metadata: %w", err)
		}
	}
	return processMetadata, nil
}
