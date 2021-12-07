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

func (c *Client) GetProcessList(entityId []byte, searchTerm string, namespace uint32, status string, withResults bool, srcNetId string, from, listSize int) (processList []string, _ error) {
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

func (c *Client) SetEntityMetadata(avatar, description, header,
	name string, entityID []byte) (metaURI string, _ error) {
	var metaBytes []byte
	var err error
	var entityMetadata apitypes.EntityMetadata
	entityMetadata.Avatar = avatar
	entityMetadata.Description = description
	entityMetadata.Header = header
	entityMetadata.Name = name

	if metaBytes, err = json.Marshal(entityMetadata); err != nil {
		return "", fmt.Errorf("could not marshal entity metadata: %v", err)
	}
	if metaURI, err = c.AddFile(metaBytes, "ipfs",
		fmt.Sprintf("%X entity metadata", entityID)); err != nil {
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

func (c *Client) FetchFile(URI string) (content []byte, _ error) {
	resp, err := c.pool.Request(api.APIrequest{
		Method: "fetchFile",
		URI:    URI,
	}, c.signingKey)
	if err != nil {
		return []byte{}, fmt.Errorf("could not fetch file %s: %v", URI, err)
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
	entityID, censusRoot []byte,
	censusURI string,
	pid []byte,
	envelopeType *models.EnvelopeType,
	censusOrigin models.CensusOrigin,
	duration int) (blockHeight uint32, _ error) {
	var req api.APIrequest
	req.Method = "submitRawTx"
	block, err := c.GetCurrentBlock()
	if err != nil {
		return 0, err
	}
	processData := &models.Process{
		EntityId:     entityID,
		CensusRoot:   censusRoot,
		CensusURI:    &censusURI,
		CensusOrigin: censusOrigin,
		BlockCount:   uint32(duration),
		ProcessId:    pid,
		StartBlock:   block + 4,
		EnvelopeType: envelopeType,
		Mode:         &models.ProcessMode{AutoStart: true, Interruptible: true},
		Status:       models.ProcessStatus_READY,
		VoteOptions:  &models.ProcessVoteOptions{MaxCount: 16, MaxValue: 8},
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
	if stx.Signature, err = c.signingKey.Sign(stx.Tx); err != nil {
		return 0, err
	}
	if req.Payload, err = proto.Marshal(stx); err != nil {
		return 0, err
	}

	if _, err = c.pool.Request(req, c.signingKey); err != nil {
		return 0, err
	}
	return p.Process.StartBlock, nil
}
