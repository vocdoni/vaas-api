package vocclient

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.vocdoni.io/api/types"
	apiUtil "go.vocdoni.io/api/util"
	"go.vocdoni.io/dvote/api"
	"go.vocdoni.io/dvote/client"
	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/log"
	dvoteTypes "go.vocdoni.io/dvote/types"
	"go.vocdoni.io/dvote/util"
	"go.vocdoni.io/dvote/vochain/scrutinizer/indexertypes"
	"go.vocdoni.io/proto/build/go/models"
	"google.golang.org/protobuf/proto"
)

const (
	TIMEOUT_TIME        = time.Minute
	HEIGHT_REQUEST_TIME = 10 * time.Second
	// VOCHAIN_BLOCK_MARGIN is the number of blocks a
	//  process should be set in the future to ensure its creation
	VOCHAIN_BLOCK_MARGIN = 5
)

type vocBlockHeight struct {
	height    uint32
	timestamp int32
	avgTimes  [5]int32
	lock      sync.RWMutex
}

type Client struct {
	gw          *client.Client
	signingKey  *ethereum.SignKeys
	blockHeight *vocBlockHeight
}

// New initializes a new gatewayPool with the gatewayUrls, in order of health
// returns the new Client
func New(gatewayUrl string, signingKey *ethereum.SignKeys) (*Client, error) {
	gw, err := DiscoverGateway(gatewayUrl)
	if err != nil {
		return nil, err
	}

	c := &Client{
		gw:          gw,
		signingKey:  signingKey,
		blockHeight: &vocBlockHeight{},
	}

	go func() {
		for {
			time.Sleep(HEIGHT_REQUEST_TIME)
			err = c.getVocHeight()
			if err != nil {
				log.Warnf("could not update blockHeight cache: %v", err)
			}
		}
	}()

	return c, nil
}

// ActiveEndpoint returns the address of the current active endpoint, if one exists
func (c *Client) ActiveEndpoint() string {
	if c.gw == nil {
		return ""
	}
	return c.gw.Addr
}

func (c *Client) request(req api.APIrequest,
	signer *ethereum.SignKeys) (*api.APIresponse, error) {
	resp, err := c.gw.Request(req, signer)
	if err != nil {
		return nil, err
	}
	if !resp.Ok {
		return nil, fmt.Errorf(resp.Message)
	}
	return resp, nil
}

// FETCHING INFO APIS

// GetVoteStatus returns the processID and registration
//  status for a given nullifier from the vochain
func (c *Client) GetVoteStatus(nullifier []byte) ([]byte, bool, error) {
	req := api.APIrequest{
		Method:    "getEnvelopeStatus",
		Nullifier: nullifier,
	}
	resp, err := c.request(req, c.signingKey)
	if err != nil {
		return nil, false, err
	}
	if !resp.Ok {
		return nil, false, fmt.Errorf("could not get vote status: %s", resp.Message)
	}
	if resp.Registered == nil {
		return nil, false, fmt.Errorf("vote registered is nil")
	}
	return resp.ProcessID, *resp.Registered, nil
}

// GetCurrentBlock returns the height of the current vochain block
func (c *Client) GetCurrentBlock() (uint32, error) {
	req := api.APIrequest{Method: "getBlockHeight"}
	resp, err := c.request(req, c.signingKey)
	if err != nil {
		return 0, err
	}
	if !resp.Ok {
		return 0, fmt.Errorf("could not get current block: %s", resp.Message)
	}
	if resp.Height == nil {
		return 0, fmt.Errorf("height is nil")
	}
	return *resp.Height, nil
}

// GetBlockTimes returns the current block height, average block times, and recent block timestamp
// from the vochain. It queries the blockHeight cache updated by the client
func (c *Client) GetBlockTimes() (uint32, [5]int32, int32) {
	c.blockHeight.lock.RLock()
	defer c.blockHeight.lock.RUnlock()
	return c.blockHeight.height, c.blockHeight.avgTimes, c.blockHeight.timestamp
}

// GetBlock fetches the vochain block at the given height and returns its summary
func (c *Client) GetBlock(height uint32) (*indexertypes.BlockMetadata, error) {
	req := api.APIrequest{Method: "getBlock", Height: height}
	resp, err := c.request(req, c.signingKey)
	if err != nil {
		return nil, err
	}
	if !resp.Ok {
		return nil, fmt.Errorf("could not get block: %s", resp.Message)
	}
	return resp.Block, nil
}

// GetProcess returns the process parameters for the given process id
func (c *Client) GetProcess(pid []byte) (*indexertypes.Process, error) {
	req := api.APIrequest{Method: "getProcessInfo", ProcessID: pid}
	resp, err := c.request(req, c.signingKey)
	if err != nil {
		return nil, err
	}
	if !resp.Ok || resp.Process == nil {
		return nil, fmt.Errorf("cannot getProcessInfo: %v", resp.Message)
	}
	if resp.Process.Metadata == "" {
		return nil, fmt.Errorf("election metadata not yet set")
	}
	return resp.Process, nil
}

// GetProcessKeys returns the encryption pubKeys for a process
func (c *Client) GetProcessPubKeys(pid []byte) ([]api.Key, error) {
	req := api.APIrequest{Method: "getProcessKeys", ProcessID: pid}
	resp, err := c.request(req, c.signingKey)
	if err != nil {
		return nil, err
	}
	if !resp.Ok {
		return nil, fmt.Errorf("could not get process keys: %s", resp.Message)
	}
	return resp.EncryptionPublicKeys, nil
}

// GetAccount returns the metadata URI, token balance, and nonce for the
//  given account ID on the vochain
func (c *Client) GetAccount(entityId []byte) (string, uint64, uint32, error) {
	req := api.APIrequest{Method: "getAccount", EntityId: entityId}
	resp, err := c.request(req, c.signingKey)
	if err != nil {
		return "", 0, 0, err
	}
	if !resp.Ok {
		return "", 0, 0, fmt.Errorf("could not get account: %s", resp.Message)
	}
	if resp.Balance == nil {
		resp.Balance = new(uint64)
	}
	if resp.Nonce == nil {
		resp.Nonce = new(uint32)
	}
	if resp.InfoURI == "" {
		return "", 0, 0, fmt.Errorf("account info URI not yet set")
	}
	return resp.InfoURI, *resp.Balance, *resp.Nonce, nil
}

// GetResults returns the results for the given processID, if available
func (c *Client) GetResults(pid []byte) (*types.VochainResults, error) {
	req := api.APIrequest{Method: "getResults", ProcessID: pid}
	resp, err := c.request(req, c.signingKey)
	if err != nil {
		return nil, err
	}
	if !resp.Ok {
		return nil, fmt.Errorf("could not get results: %s", resp.Message)
	}
	if resp.Height == nil {
		resp.Height = new(uint32)
	}
	results := &types.VochainResults{
		Results: resp.Results,
		State:   resp.State,
		Type:    resp.Type,
		Height:  *resp.Height,
	}
	return results, nil
}

// GetProcessList queries for a list of process ids from the vochain.
// filters include entityID, status ("READY", "ENDED", "CANCELED", "PAUSED", "RESULTS"),
//  source network ID (for processes created on ethereum or other source-of-truth blockchains),
//  searchTerm (partial or whole processID match), namespace, and results availability.
// listSize can be between 0 and 64. To query for a process list longer than 64,
//  iteratively increment `from` by `listSize` until no more processes are retrieved
func (c *Client) GetProcessList(entityId []byte, status, srcNetId, searchTerm string,
	namespace uint32, withResults bool, from, listSize int) ([]string, error) {
	req := api.APIrequest{
		Method:      "getProcessList",
		EntityId:    entityId,
		SearchTerm:  searchTerm,
		Namespace:   namespace,
		SrcNetId:    srcNetId,
		Status:      status,
		WithResults: withResults,
		From:        from,
		ListSize:    listSize,
	}
	resp, err := c.request(req, c.signingKey)
	if err != nil {
		return nil, err
	}
	if !resp.Ok {
		return nil, fmt.Errorf("could not get results: %s", resp.Message)
	}
	if resp.Message == "no results yet" {
		return nil, nil
	}
	return resp.ProcessList, nil
}

// FILE APIS

// SetEntityMetadata pins the given entity metadata to IPFS and returns its URI
func (c *Client) SetEntityMetadata(meta types.EntityMetadata,
	entityID []byte) (string, error) {
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return "", fmt.Errorf("could not marshal entity metadata: %v", err)
	}
	metaURI, err := c.AddFile(metaBytes, "ipfs",
		fmt.Sprintf("%X entity metadata", entityID))
	if err != nil {
		return "", fmt.Errorf("could not post metadata to ipfs: %v", err)
	}
	return metaURI, nil
}

// SetProcessMetadata pins the given process metadata to IPFS and returns its URI
func (c *Client) SetProcessMetadata(meta types.ProcessMetadata,
	processId, metadataPrivKey []byte) (string, error) {
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return "", fmt.Errorf("could not marshal process metadata: %v", err)
	}
	if len(metadataPrivKey) > 0 {
		log.Debugf("encrypting metatadata for %x", processId)
		encryptedMeta, err := apiUtil.EncryptSymmetric(metaBytes, metadataPrivKey)
		if err != nil {
			return "", fmt.Errorf("could not encrypt private metadata: %w", err)
		}
		metaBytes, err = json.Marshal(types.RawFile{Payload: encryptedMeta, Version: "1.0"})
		if err != nil {
			return "", fmt.Errorf("could not marshal encrypted bytes: %v", err)
		}
	}
	return c.AddFile(metaBytes, "ipfs",
		fmt.Sprintf("%X process metadata", processId))

}

// SetProcessMetadataConfidential encrypts with metadataPrivKey and then pins
//  the given process metadata to IPFS and returns its URI
func (c *Client) SetProcessMetadataConfidential(meta types.ProcessMetadata, metadataPrivKey,
	processId []byte) (string, error) {
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return "", fmt.Errorf("could not marshal process metadata: %v", err)
	}
	log.Debugf("encrypting metatadata for %x", processId)
	encryptedMeta, err := apiUtil.EncryptSymmetric(metaBytes, metadataPrivKey)
	if err != nil {
		return "", fmt.Errorf("could not encrypt private metadata: %w", err)
	}
	metaBytes, err = json.Marshal(types.RawFile{Payload: encryptedMeta, Version: "1.0"})
	if err != nil {
		return "", fmt.Errorf("could not marshal encrypted bytes: %v", err)
	}
	return c.AddFile(metaBytes, "ipfs",
		fmt.Sprintf("%X process metadata (encrypted)", processId))
}

// AddFile pins the given content to the gateway's storage mechanism,
//  specified by contentType, and returns its URI
func (c *Client) AddFile(content []byte, contentType, name string) (string, error) {
	resp, err := c.request(api.APIrequest{
		Method:  "addFile",
		Content: content,
		Type:    contentType,
		Name:    name,
	}, c.signingKey)
	if err != nil {
		return "", fmt.Errorf("could not AddFile %s: %v", name, err)
	}
	if !resp.Ok {
		return "", fmt.Errorf("could not AddFile %s: %s", name, resp.Message)
	}
	return resp.URI, nil
}

// FetchProcessMetadata fetches and attempts to unmarshal & return
//  the process metadata from the given URI
func (c *Client) FetchProcessMetadata(URI string) (*types.ProcessMetadata, error) {
	content, err := c.FetchFile(URI)
	if err != nil {
		return nil, err
	}
	var process types.ProcessMetadata
	if err = json.Unmarshal(content, &process); err != nil {
		return nil, err
	}
	return &process, nil
}

// FetchProcessMetadataConfidential fetches and attempts to decrypt, unmarshal & return
//  the process metadata from the given URI
func (c *Client) FetchProcessMetadataConfidential(URI string,
	metadataPrivKey []byte) (*types.ProcessMetadata, error) {
	content, err := c.FetchFile(URI)
	if err != nil {
		return nil, err
	}
	var file types.RawFile
	if err = json.Unmarshal(content, &file); err != nil {
		return nil, err
	}
	decrypted, ok := apiUtil.DecryptSymmetric(file.Payload, metadataPrivKey)
	if !ok {
		return nil, fmt.Errorf("could not decrypt private metadata")
	}
	var process types.ProcessMetadata
	return &process, json.Unmarshal(decrypted, &process)
}

// FetchOrganizationMetadata fetches and attempts to unmarshal & return
//   the organization metadata from the given URI
func (c *Client) FetchOrganizationMetadata(URI string) (*types.EntityMetadata, error) {
	content, err := c.FetchFile(URI)
	if err != nil {
		return nil, err
	}
	var entity types.EntityMetadata
	if err = json.Unmarshal(content, &entity); err != nil {
		return nil, err
	}
	return &entity, nil
}

// FetchFile fetches and returns a raw file from the given URI, via the gateway
func (c *Client) FetchFile(URI string) ([]byte, error) {
	resp, err := c.request(api.APIrequest{
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
func (c *Client) AddCensus() (string, error) {
	req := api.APIrequest{
		Method:     "addCensus",
		CensusType: models.Census_ARBO_BLAKE2B,
		CensusID:   fmt.Sprintf("census%d", util.RandomInt(0, 2<<32)),
	}
	resp, err := c.request(req, c.signingKey)
	if err != nil {
		return "", err
	}
	if !resp.Ok {
		return "", fmt.Errorf(resp.Message)
	}
	return resp.CensusID, nil
}

// AddClaim adds a new publickey to the existing census specified by censusID and returns its root
func (c *Client) AddClaim(censusID string, censusSigner *ethereum.SignKeys, censusPubKey string,
	censusValue []byte) (dvoteTypes.HexBytes, error) {
	req := api.APIrequest{
		Method:   "addClaim",
		Digested: false,
		CensusID: censusID,
	}
	var hexpub string
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
	resp, err := c.request(req, c.signingKey)
	if err != nil {
		return dvoteTypes.HexBytes{}, err
	}
	if !resp.Ok {
		return nil, fmt.Errorf(resp.Message)
	}
	return resp.Root, nil
}

// AddClaimBulk adds many new publickeys to the existing census specified by censusID
//  and returns the census root and returns the number of invalid claims
func (c *Client) AddClaimBulk(censusID string, censusSigners []*ethereum.SignKeys,
	censusPubKeys []string, censusValues []*dvoteTypes.BigInt) (dvoteTypes.HexBytes, []int, error) {
	req := api.APIrequest{
		CensusID:  censusID,
		Method:    "addClaimBulk",
		CensusKey: []byte{},
		Digested:  false,
	}
	censusSize := 0
	if censusSigners != nil {
		censusSize = len(censusSigners)
	} else {
		censusSize = len(censusPubKeys)
	}
	log.Infof("add bulk claims (size %d)", censusSize)
	currentSize := censusSize
	totalRequests := 0
	var hexpub string
	var root dvoteTypes.HexBytes
	var invalidClaims []int
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
		resp, err := c.request(req, c.signingKey)
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
	rootHash dvoteTypes.HexBytes) (string, error) {
	req := api.APIrequest{
		Method:   "publish",
		CensusID: censusID,
		RootHash: rootHash,
	}
	resp, err := c.request(req, c.signingKey)
	if err != nil {
		return "", err
	}
	if !resp.Ok {
		return "", fmt.Errorf(resp.Message)
	}
	if len(resp.URI) < 40 {
		return "", fmt.Errorf("got invalid URI")
	}
	return resp.URI, nil
}

// GetRoot returns the root for the given censusID
func (c *Client) GetRoot(censusID string) (dvoteTypes.HexBytes, error) {
	req := api.APIrequest{
		Method:   "getRoot",
		CensusID: censusID,
	}
	resp, err := c.request(req, c.signingKey)
	if err != nil {
		return dvoteTypes.HexBytes{}, err
	}
	if !resp.Ok {
		return []byte{}, fmt.Errorf(resp.Message)
	}
	return resp.Root, nil
}

// Transaction APIs

// SetAccountInfo submits a transaction to set an account with the given
//  ethereum wallet address and metadata URI on the vochain
func (c *Client) SetAccountInfo(signer *ethereum.SignKeys, uri string, nonce uint32) error {
	req := api.APIrequest{Method: "submitRawTx"}
	tx := models.Tx_SetAccountInfo{SetAccountInfo: &models.SetAccountInfoTx{
		Txtype:  models.TxType_SET_ACCOUNT_INFO,
		Nonce:   nonce,
		InfoURI: uri,
	}}
	var err error
	stx := new(models.SignedTx)
	stx.Tx, err = proto.Marshal(&models.Tx{Payload: &tx})
	if err != nil {
		return fmt.Errorf("could not marshal set account info tx")
	}
	stx.Signature, err = signer.Sign(stx.Tx)
	if err != nil {
		return fmt.Errorf("could not sign account transaction: %v", err)
	}
	if req.Payload, err = proto.Marshal(stx); err != nil {
		return err
	}
	resp, err := c.request(req, c.signingKey)
	if err != nil {
		return err
	}
	if !resp.Ok {
		return fmt.Errorf(resp.Message)
	}
	return nil
}

// CreateProcess submits a transaction to the vochain to
//  create a process with the given configuration and returns its starting block height
func (c *Client) CreateProcess(process *models.Process,
	signingKey *ethereum.SignKeys, nonce uint32) error {
	req := api.APIrequest{Method: "submitRawTx"}
	p := &models.NewProcessTx{
		Txtype:  models.TxType_NEW_PROCESS,
		Process: process,
		Nonce:   make([]byte, 4),
	}
	binary.LittleEndian.PutUint32(p.Nonce, nonce)
	var err error
	stx := &models.SignedTx{}
	stx.Tx, err = proto.Marshal(&models.Tx{Payload: &models.Tx_NewProcess{NewProcess: p}})
	if err != nil {
		return err
	}
	if stx.Signature, err = signingKey.Sign(stx.Tx); err != nil {
		return err
	}
	if req.Payload, err = proto.Marshal(stx); err != nil {
		return err
	}
	resp, err := c.request(req, c.signingKey)
	if err != nil {
		return err
	}
	if !resp.Ok {
		return fmt.Errorf(resp.Message)
	}
	return nil
}

// SetProcessStatus updates the process given by `pid` status to `status`
//  using the organization's `signkeys`
func (c *Client) SetProcessStatus(pid []byte,
	status *models.ProcessStatus, signingKey *ethereum.SignKeys, nonce uint32) error {
	req := api.APIrequest{Method: "submitRawTx"}
	p := &models.SetProcessTx{
		Txtype:    models.TxType_SET_PROCESS_STATUS,
		ProcessId: pid,
		Status:    status,
		Nonce:     make([]byte, 4),
	}
	binary.LittleEndian.PutUint32(p.Nonce, nonce)
	stx := &models.SignedTx{}
	var err error
	stx.Tx, err = proto.Marshal(&models.Tx{Payload: &models.Tx_SetProcess{SetProcess: p}})
	if err != nil {
		return err
	}
	if stx.Signature, err = signingKey.Sign(stx.Tx); err != nil {
		return err
	}
	if req.Payload, err = proto.Marshal(stx); err != nil {
		return err
	}

	resp, err := c.request(req, nil)
	if err != nil {
		return err
	}
	if !resp.Ok {
		return fmt.Errorf("%s failed: %s", req.Method, resp.Message)
	}
	return nil
}

// RelayVote relays a given raw vote transaction to the vochain and returns its nullifier
func (c *Client) RelayVote(signedTx []byte) (string, error) {
	resp, err := c.request(api.APIrequest{
		Method:  "submitRawTx",
		Payload: signedTx,
	}, nil)
	if err != nil {
		return "", err
	}
	if !resp.Ok {
		return "", fmt.Errorf("could not cast vote: %s", resp.Message)
	}
	if len(resp.Payload) < 1 {
		return "", fmt.Errorf("did not receive nullifier")
	}
	return resp.Payload, nil
}

func (c *Client) getVocHeight() error {
	resp, err := c.request(api.APIrequest{
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

	resp, err = c.request(api.APIrequest{
		Method: "getBlockStatus",
	}, c.signingKey)
	if err != nil {
		return err
	}
	if !resp.Ok {
		return fmt.Errorf(resp.Message)
	}
	if resp.BlockTime == nil {
		return fmt.Errorf("blockTime is nil")
	}
	c.blockHeight.avgTimes = *resp.BlockTime
	if resp.BlockTimestamp == 0 {
		return fmt.Errorf("blockTimestamp is 0")
	}
	c.blockHeight.timestamp = resp.BlockTimestamp
	log.Debugf("block info %v", c.blockHeight)
	return nil
}
