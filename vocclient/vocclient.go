package vocclient

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"go.vocdoni.io/api/types"
	"go.vocdoni.io/dvote/api"
	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/log"
	dvoteTypes "go.vocdoni.io/dvote/types"
	"go.vocdoni.io/dvote/util"
	"go.vocdoni.io/dvote/vochain/scrutinizer/indexertypes"
	"go.vocdoni.io/proto/build/go/models"
	"google.golang.org/protobuf/proto"
)

const TIMEOUT_TIME = 1 * time.Minute
const HEIGHT_REQUEST_TIME = 10 * time.Second

var MAX_CENSUS_SIZE = uint64(1024)

type vocBlockHeight struct {
	height    uint32
	timestamp int32
	avgTimes  [5]int32
	lock      sync.RWMutex
}

type Client struct {
	pool        GatewayPool
	signingKey  *ethereum.SignKeys
	blockHeight *vocBlockHeight
}

// New initializes a new gatewayPool with the gatewayUrls, in order of health
// returns the new Client
func New(gatewayUrls []string, signingKey *ethereum.SignKeys) (*Client, error) {
	gwPool, err := DiscoverGateways(gatewayUrls)
	if err != nil {
		return nil, err
	}

	c := &Client{
		pool:        gwPool,
		signingKey:  signingKey,
		blockHeight: &vocBlockHeight{},
	}

	go func() {
		for {
			time.Sleep(HEIGHT_REQUEST_TIME)
			c.getVocHeight()
		}
	}()

	return c, nil
}

// ActiveEndpoint returns the address of the current active endpoint, if one exists
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

// GetVoteStatus returns the processID and registration
//  status for a given nullifier from the vochain
func (c *Client) GetVoteStatus(nullifier []byte) ([]byte, bool, error) {
	var req api.APIrequest
	req.Method = "getEnvelopeStatus"
	req.Nullifier = nullifier
	resp, err := c.pool.Request(req, c.signingKey)
	if err != nil {
		return nil, false, err
	}
	if resp.Registered == nil {
		return nil, false, fmt.Errorf("vote registered is nil")
	}
	return resp.ProcessID, *resp.Registered, nil
}

// GetCurrentBlock returns the height of the current vochain block
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

// GetBlockTimes returns the current block height, average block times, and recent block timestamp
// from the vochain. It queries the blockHeight cache updated by the client
func (c *Client) GetBlockTimes() (blockHeight uint32,
	blockTimes [5]int32, blockTimestamp int32, _ error) {
	c.blockHeight.lock.RLock()
	defer c.blockHeight.lock.RUnlock()
	return c.blockHeight.height, c.blockHeight.avgTimes, c.blockHeight.timestamp, nil
}

// GetBlock fetches the vochain block at the given height and returns its summary
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

// GetProcess returns the process parameters for the given process id
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

// GetAccount returns the metadata URI, token balance, and nonce for the
//  given account ID on the vochain
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

// GetResults returns the results for the given processID, if available
func (c *Client) GetResults(pid []byte) (results *types.VochainResults, _ error) {
	var req api.APIrequest
	results = new(types.VochainResults)
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

// GetProcessList queries for a list of process ids from the vochain.
// filters include entityID, status ("READY", "ENDED", "CANCELED", "PAUSED", "RESULTS"),
//  source network ID (for processes created on ethereum or other source-of-truth blockchains),
//  searchTerm (partial or whole processID match), namespace, and results availability.
// listSize can be between 0 and 64. To query for a process list longer than 64,
//  iteratively increment `from` by `listSize` until no more processes are retrieved
func (c *Client) GetProcessList(entityId []byte, status, srcNetId, searchTerm string,
	namespace uint32, withResults bool, from, listSize int) (processList []string, _ error) {
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

// SetEntityMetadata pins the given entity metadata to IPFS and returns its URI
func (c *Client) SetEntityMetadata(meta types.EntityMetadata,
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

// SetProcessMetadata pins the given process metadata to IPFS and returns its URI
func (c *Client) SetProcessMetadata(meta types.ProcessMetadata,
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

// AddFile pins the given content to the gateway's storage mechanism,
//  specified by contentType, and returns its URI
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

// FetchProcessMetadata fetches and attempts to unmarshal & return
//  the process metadata from the given URI
func (c *Client) FetchProcessMetadata(URI string) (process *types.ProcessMetadata, _ error) {
	content, err := c.FetchFile(URI)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(content, &process); err != nil {
		return nil, err
	}
	return process, nil
}

// FetchOrganizationMetadata fetches and attempts to unmarshal & return
//   the organization metadata from the given URI
func (c *Client) FetchOrganizationMetadata(URI string) (entity *types.EntityMetadata, _ error) {
	content, err := c.FetchFile(URI)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(content, &entity); err != nil {
		return nil, err
	}
	return entity, nil
}

// FetchFile fetches and returns a raw file from the given URI, via the gateway
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

// AddCensus creates a new census and returns its ID
func (c *Client) AddCensus() (CensusID string, _ error) {
	var req api.APIrequest

	// Create census
	log.Infof("Create census")
	req.Method = "addCensus"
	req.CensusType = models.Census_ARBO_BLAKE2B
	req.CensusID = fmt.Sprintf("census%d", rand.Int())
	resp, err := c.pool.Request(req, c.signingKey)
	if err != nil {
		return "", err
	}
	return resp.CensusID, nil
}

// AddClaim adds a new publickey to the existing census specified by censusID and returns its root
func (c *Client) AddClaim(censusID string, censusSigner *ethereum.SignKeys, censusPubKey string,
	censusValue []byte) (root dvoteTypes.HexBytes, _ error) {
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
		return dvoteTypes.HexBytes{}, err
	}
	req.CensusKey = pub
	req.CensusValue = censusValue
	resp, err := c.pool.Request(req, c.signingKey)
	if err != nil {
		return dvoteTypes.HexBytes{}, err
	}
	return resp.Root, nil
}

// AddClaimBulk adds many new publickeys to the existing census specified by censusID
//  and returns the census root and returns the number of invalid claims
func (c *Client) AddClaimBulk(censusID string, censusSigners []*ethereum.SignKeys,
	censusPubKeys []string, censusValues []*dvoteTypes.BigInt) (root dvoteTypes.HexBytes,
	invalidClaims []int, _ error) {
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
		values := []*dvoteTypes.BigInt{}
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
				return dvoteTypes.HexBytes{}, []int{}, err
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
			return dvoteTypes.HexBytes{}, []int{}, err
		}
		root = resp.Root
		invalidClaims = append(invalidClaims, resp.InvalidClaims...)
		totalRequests++
		log.Infof("census creation progress: %d%%", (totalRequests*100*100)/(censusSize))
	}
	return root, invalidClaims, nil
}

// PublishCensus publishes the census with the given rootHash and returns its URI
func (c *Client) PublishCensus(censusID string,
	rootHash dvoteTypes.HexBytes) (uri string, _ error) {
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

// GetRoot returns the root for the given censusID
func (c *Client) GetRoot(censusID string) (root dvoteTypes.HexBytes, _ error) {
	var req api.APIrequest
	req.Method = "getRoot"
	req.CensusID = censusID
	resp, err := c.pool.Request(req, c.signingKey)
	if err != nil {
		return dvoteTypes.HexBytes{}, err
	}
	return resp.Root, nil
}

// Transaction APIs

// SetAccountInfo submits a transaction to set an account with the given
//  ethereum wallet address and metadata URI on the vochain
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

// CreateProcess submits a transaction to the vochain to
//  create a process with the given configuration and returns its starting block height
func (c *Client) CreateProcess(
	pid []byte, entityID []byte, startBlock, duration uint32, censusRoot []byte, censusURI string,
	envelopeType *models.EnvelopeType, processMode *models.ProcessMode,
	voteOptions *models.ProcessVoteOptions, censusOrigin models.CensusOrigin, metadataUri string,
	signingKey *ethereum.SignKeys) (blockHeight uint32, _ error) {
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

// RelayVote relays a raw vote transaction to the vochain and returns its nullifier
func (c *Client) RelayVote(signedTx []byte) (string, error) {
	var err error
	var resp *api.APIresponse
	if resp, err = c.pool.Request(api.APIrequest{
		Method:  "submitRawTx",
		Payload: signedTx,
	}, c.signingKey); err != nil {
		return "", err
	}

	if !resp.Ok {
		return "", fmt.Errorf("could not cast voteTx to the vochain: %s", resp.Message)
	}

	if len(resp.Nullifier) < 1 {
		return "", fmt.Errorf("RelayVote: did not revieve nullifier")
	}

	return resp.Nullifier, nil
}

func (c *Client) getVocHeight() error {
	resp, err := c.pool.Request(api.APIrequest{
		Method: "getBlockHeight",
	}, c.signingKey)
	if err != nil {
		return err
	}
	if resp.Height == nil {
		return fmt.Errorf("height is nil")
	}
	c.blockHeight.lock.Lock()
	defer c.blockHeight.lock.Unlock()
	c.blockHeight.height = *resp.Height

	resp, err = c.pool.Request(api.APIrequest{
		Method: "getBlockStatus",
	}, c.signingKey)
	if err != nil {
		return err
	}
	if resp.BlockTime == nil {
		return fmt.Errorf("blockTime is nil")
	}
	c.blockHeight.avgTimes = *resp.BlockTime
	if resp.BlockTimestamp == 0 {
		return fmt.Errorf("blockTimestamp is 0")
	}
	c.blockHeight.timestamp = resp.BlockTimestamp
	log.Debugf("Block info %v", c.blockHeight)
	return nil
}
