package vocclient

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"

	apitypes "go.vocdoni.io/api/types"
	"go.vocdoni.io/dvote/api"
	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/log"
	"go.vocdoni.io/dvote/types"
	"go.vocdoni.io/dvote/util"
	"go.vocdoni.io/dvote/vochain/scrutinizer/indexertypes"
	"go.vocdoni.io/proto/build/go/models"
	"google.golang.org/protobuf/proto"
)

var MAX_CENSUS_SIZE = uint64(1024)

type Client struct {
	pool       GatewayPool
	signingKey *ethereum.SignKeys
}

func New(gatewayUrls []string, signingKey *ethereum.SignKeys) (*Client, error) {
	gwPool, err := discoverGateways(gatewayUrls)
	if err != nil {
		return nil, err
	}
	return &Client{
		pool:       gwPool,
		signingKey: signingKey,
	}, nil
}

func (c *Client) ActiveEndpoint() string {
	gw, err := c.pool.activeGateway()
	if err != nil {
		return ""
	}
	if gw.client == nil {
		return ""
	}
	return gw.client.Addr
}

// FETCHING INFO APIS

func (c *Client) GetCurrentBlock() (blockHeight uint32, _ error) {
	var req api.APIrequest
	req.Method = "getBlockHeight"
	resp, err := c.pool.Request(req, c.signingKey)
	if err != nil {
		return 0, err
	}
	if resp.Height == nil {
		return 0, fmt.Errorf("height is nil")
	}
	return *resp.Height, nil
}

func (c *Client) GetBlockTimes() (blockTimes [5]int32, _ error) {
	var req api.APIrequest
	req.Method = "getBlockStatus"
	resp, err := c.pool.Request(req, c.signingKey)
	if err != nil {
		return [5]int32{}, err
	}
	if resp.BlockTime == nil {
		return [5]int32{}, fmt.Errorf("blockTime is nil")
	}
	return *resp.BlockTime, nil
}

func (c *Client) GetBlock(height uint32) (*indexertypes.BlockMetadata, error) {
	var req api.APIrequest
	req.Method = "getBlock"
	req.Height = height
	resp, err := c.pool.Request(req, c.signingKey)
	if err != nil {
		return nil, err
	}
	return resp.Block, nil
}

func (c *Client) GetProcess(pid []byte) (*indexertypes.Process, error) {
	var req api.APIrequest
	req.Method = "getProcessInfo"
	req.ProcessID = pid
	resp, err := c.pool.Request(req, c.signingKey)
	if err != nil {
		return nil, err
	}
	return resp.Process, nil
}

func (c *Client) GetAccount(entityId []byte) (string, uint64, uint32, error) {
	var req api.APIrequest
	req.Method = "getAccount"
	req.EntityId = entityId
	resp, err := c.pool.Request(req, c.signingKey)
	if err != nil {
		return "", 0, 0, err
	}
	if resp.Balance == nil {
		resp.Balance = new(uint64)
	}
	if resp.Nonce == nil {
		resp.Nonce = new(uint32)
	}
	return resp.InfoURI, *resp.Balance, *resp.Nonce, nil
}

func (c *Client) GetResults(pid []byte) (results *apitypes.VochainResults, _ error) {
	var req api.APIrequest
	results = new(apitypes.VochainResults)
	req.Method = "getResults"
	req.ProcessID = pid
	resp, err := c.pool.Request(req, c.signingKey)
	if err != nil {
		return nil, err
	}
	if resp.Height == nil {
		resp.Height = new(uint32)
	}
	results.Results = resp.Results
	results.State = resp.State
	results.Type = resp.Type
	results.Height = *resp.Height
	return results, nil
}

func (c *Client) GetProcessList(entityId []byte, status, srcNetId, searchTerm string, namespace uint32, withResults bool, from, listSize int) (processList []string, _ error) {
	var req api.APIrequest
	req.Method = "getProcessList"
	req.EntityId = entityId
	req.SearchTerm = searchTerm
	req.Namespace = namespace
	req.SrcNetId = srcNetId
	req.Status = status
	req.WithResults = withResults
	req.From = from
	req.ListSize = listSize
	resp, err := c.pool.Request(req, c.signingKey)
	if err != nil {
		return nil, err
	}
	if resp.Message == "no results yet" {
		return nil, nil
	}
	return resp.ProcessList, nil
}

// FILE APIS

func (c *Client) SetEntityMetadata(meta apitypes.EntityMetadata,
	entityID []byte) (metaURI string, _ error) {
	var metaBytes []byte
	var err error

	if metaBytes, err = json.Marshal(meta); err != nil {
		return "", fmt.Errorf("could not marshal entity metadata: %v", err)
	}
	if metaURI, err = c.AddFile(metaBytes, "ipfs",
		fmt.Sprintf("%X entity metadata", entityID)); err != nil {
		return "", fmt.Errorf("could not post metadata to ipfs: %v", err)
	}
	return metaURI, nil
}

func (c *Client) SetProcessMetadata(meta apitypes.ProcessMetadata,
	processId []byte) (metaURI string, _ error) {
	var metaBytes []byte
	var err error

	if metaBytes, err = json.Marshal(meta); err != nil {
		return "", fmt.Errorf("could not marshal process metadata: %v", err)
	}
	log.Debugf("meta: %s", string(metaBytes))
	if metaURI, err = c.AddFile(metaBytes, "ipfs",
		fmt.Sprintf("%X process metadata", processId)); err != nil {
		return "", fmt.Errorf("could not post metadata to ipfs: %v", err)
	}
	return metaURI, nil
}

func (c *Client) AddFile(content []byte, contentType, name string) (URI string, _ error) {
	resp, err := c.pool.Request(api.APIrequest{
		Method:  "addFile",
		Content: content,
		Type:    contentType,
		Name:    name,
	}, c.signingKey)
	if err != nil {
		return "", fmt.Errorf("could not AddFile %s: %v", name, err)
	}
	return resp.URI, nil
}

func (c *Client) FetchProcessMetadata(URI string) (process *apitypes.ProcessMetadata, _ error) {
	content, err := c.FetchFile(URI)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(content, &process); err != nil {
		return nil, err
	}
	return process, nil
}

func (c *Client) FetchOrganizationMetadata(URI string) (entity *apitypes.EntityMetadata, _ error) {
	content, err := c.FetchFile(URI)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(content, &entity); err != nil {
		return nil, err
	}
	return entity, nil
}

func (c *Client) FetchFile(URI string) (content []byte, _ error) {
	resp, err := c.pool.Request(api.APIrequest{
		Method: "fetchFile",
		URI:    URI,
	}, c.signingKey)
	if err != nil {
		return []byte{}, fmt.Errorf("could not fetch file %s: %v", URI, err)
	}
	if !resp.Ok {
		return []byte{}, fmt.Errorf(resp.Message)
	}
	return resp.Content, nil
}

// CENSUS APIS

func (c *Client) AddCensus() (CensusID string, _ error) {
	var req api.APIrequest

	// Create census
	log.Infof("Create census")
	req.Method = "addCensus"
	req.CensusType = models.Census_ARBO_BLAKE2B
	// TODO does rand.Int provide sufficient entropy?
	req.CensusID = fmt.Sprintf("census%d", rand.Int())
	resp, err := c.pool.Request(req, c.signingKey)
	if err != nil {
		return "", err
	}
	return resp.CensusID, nil
}

func (c *Client) AddClaim(censusID string, censusSigner *ethereum.SignKeys, censusPubKey string, censusValue []byte) (root types.HexBytes, _ error) {
	var req api.APIrequest
	var hexpub string
	req.Method = "addClaim"
	req.Digested = false
	req.CensusID = censusID
	if censusSigner != nil {
		hexpub, _ = censusSigner.HexString()
	} else {
		hexpub = censusPubKey
	}
	pub, err := hex.DecodeString(hexpub)
	if err != nil {
		return types.HexBytes{}, err
	}
	req.CensusKey = pub
	req.CensusValue = censusValue
	resp, err := c.pool.Request(req, c.signingKey)
	if err != nil {
		return types.HexBytes{}, err
	}
	return resp.Root, nil
}

func (c *Client) AddClaimBulk(censusID string, censusSigners []*ethereum.SignKeys, censusPubKeys []string, censusValues []*types.BigInt) (root types.HexBytes, invalidClaims []int, _ error) {
	var req api.APIrequest
	req.CensusID = censusID
	censusSize := 0
	if censusSigners != nil {
		censusSize = len(censusSigners)
	} else {
		censusSize = len(censusPubKeys)
	}
	log.Infof("add bulk claims (size %d)", censusSize)
	req.Method = "addClaimBulk"
	req.CensusKey = []byte{}
	req.Digested = false
	currentSize := censusSize
	totalRequests := 0
	var hexpub string
	for currentSize > 0 {
		claims := [][]byte{}
		values := []*types.BigInt{}
		for j := 0; j < 100; j++ {
			if currentSize < 1 {
				break
			}
			if censusSigners != nil {
				hexpub, _ = censusSigners[currentSize-1].HexString()
			} else {
				hexpub = censusPubKeys[currentSize-1]
			}
			pub, err := hex.DecodeString(hexpub)
			if err != nil {
				return types.HexBytes{}, []int{}, err
			}
			claims = append(claims, pub)
			if len(censusValues) > 0 {
				values = append(values, censusValues[currentSize-1])
			}
			currentSize--
		}
		req.CensusKeys = claims
		req.Weights = values
		resp, err := c.pool.Request(req, c.signingKey)
		if err != nil {
			return types.HexBytes{}, []int{}, err
		}
		root = resp.Root
		invalidClaims = append(invalidClaims, resp.InvalidClaims...)
		totalRequests++
		log.Infof("census creation progress: %d%%", (totalRequests*100*100)/(censusSize))
	}
	return root, invalidClaims, nil
}

func (c *Client) PublishCensus(censusID string, rootHash types.HexBytes) (uri string, _ error) {
	var req api.APIrequest
	req.Method = "publish"
	req.CensusID = censusID
	req.RootHash = rootHash
	resp, err := c.pool.Request(req, c.signingKey)
	if err != nil {
		return "", err
	}
	uri = resp.URI
	if len(uri) < 40 {
		return "", fmt.Errorf("got invalid URI")
	}
	return resp.URI, nil
}

func (c *Client) GetRoot(censusID string) (root types.HexBytes, _ error) {
	var req api.APIrequest
	req.Method = "getRoot"
	req.CensusID = censusID
	resp, err := c.pool.Request(req, c.signingKey)
	if err != nil {
		return types.HexBytes{}, err
	}
	return resp.Root, nil
}

// Transaction APIs

func (c *Client) SetAccountInfo(signer *ethereum.SignKeys, uri string) error {
	var tx models.Tx_SetAccountInfo
	var req api.APIrequest
	var err error
	tx.SetAccountInfo = &models.SetAccountInfoTx{
		Txtype:  models.TxType_SET_ACCOUNT_INFO,
		Nonce:   uint32(util.RandomInt(0, 2<<32)),
		InfoURI: uri,
	}
	stx := &models.SignedTx{}
	stx.Tx, err = proto.Marshal(&models.Tx{Payload: &tx})
	if err != nil {
		return fmt.Errorf("could not marshal set account info tx")
	}
	stx.Signature, err = signer.Sign(stx.Tx)
	if err != nil {
		return fmt.Errorf("could not sign account transaction: %v", err)
	}
	req.Method = "submitRawTx"
	if req.Payload, err = proto.Marshal(stx); err != nil {
		return err
	}
	if _, err = c.pool.Request(req, c.signingKey); err != nil {
		return err
	}
	return nil
}

func (c *Client) CreateProcess(
	pid []byte, entityID []byte, startBlock, duration uint32, censusRoot []byte, censusURI string, envelopeType *models.EnvelopeType,
	processMode *models.ProcessMode, voteOptions *models.ProcessVoteOptions,
	censusOrigin models.CensusOrigin, metadataUri string, signingKey *ethereum.SignKeys) (blockHeight uint32, _ error) {
	var resp *api.APIresponse
	var req api.APIrequest
	var err error
	req.Method = "submitRawTx"
	processData := &models.Process{
		ProcessId:     pid,
		EntityId:      entityID,
		StartBlock:    startBlock,
		BlockCount:    duration,
		CensusRoot:    censusRoot,
		CensusURI:     &censusURI,
		Status:        models.ProcessStatus_READY,
		EnvelopeType:  envelopeType,
		Mode:          processMode,
		VoteOptions:   voteOptions,
		CensusOrigin:  censusOrigin,
		Metadata:      &metadataUri,
		MaxCensusSize: &MAX_CENSUS_SIZE,
	}
	p := &models.NewProcessTx{
		Txtype:  models.TxType_NEW_PROCESS,
		Nonce:   util.RandomBytes(32),
		Process: processData,
	}
	stx := &models.SignedTx{}
	stx.Tx, err = proto.Marshal(&models.Tx{Payload: &models.Tx_NewProcess{NewProcess: p}})
	if err != nil {
		return 0, err
	}
	if stx.Signature, err = signingKey.Sign(stx.Tx); err != nil {
		return 0, err
	}
	if req.Payload, err = proto.Marshal(stx); err != nil {
		return 0, err
	}
	if resp, err = c.pool.Request(req, c.signingKey); err != nil {
		return 0, err
	}
	if !resp.Ok {
		return 0, fmt.Errorf("could not create organization on the vochain: %s", resp.Message)
	}
	return p.Process.StartBlock, nil
}
