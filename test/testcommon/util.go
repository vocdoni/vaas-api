package testcommon

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/google/uuid"
	sk "github.com/vocdoni/blind-csp/saltedkey"
	"go.vocdoni.io/api/types"
	"go.vocdoni.io/dvote/api"
	"go.vocdoni.io/dvote/crypto/ethereum"
	dvotetypes "go.vocdoni.io/dvote/types"
	dvoteutil "go.vocdoni.io/dvote/util"
)

type TestOrganization struct {
	APIToken       string
	Name           string
	Description    string
	HeaderURI      string
	AvatarURI      string
	CreationTxHash string
	ID             int
	EthAddress     dvotetypes.HexBytes
}

type TestElection struct {
	Confidential       bool
	CreationTxHash     string
	Description        string
	ElectionID         dvotetypes.HexBytes
	EncryptionPubKeys  []api.Key
	EndDate            time.Time
	Header             string
	HiddenResults      bool
	OrganizationID     dvotetypes.HexBytes
	Questions          []types.Question
	Results            []types.Result
	ResultsAggregation string
	ResultsDisplay     string
	StartDate          time.Time
	Status             string
	StreamURI          string
	Title              string
	Type               string
	VoteCount          uint32
}

func CreateIntegrators(size int) []*types.Integrator {
	mp := make([]*types.Integrator, size)
	for i := 0; i < size; i++ {
		randomID := rand.Intn(10000000)
		cspPub := dvoteutil.RandomHex(32)
		cspPubKey, _ := hex.DecodeString(cspPub)
		// retrieve entity ID
		mp[i] = &types.Integrator{
			Name:         fmt.Sprintf("Test%d", randomID),
			Email:        fmt.Sprintf("mail%d@mail.org", randomID),
			CspUrlPrefix: "csp.vocdoni.net",
			CspPubKey:    cspPubKey,
		}
	}
	return mp
}

// Create a given number of random organizations
func CreateOrganizations(size int) []*TestOrganization {
	mp := make([]*TestOrganization, size)
	for i := 0; i < size; i++ {
		randomID := rand.Intn(10000000)
		mp[i] = &TestOrganization{
			Name:        fmt.Sprintf("Test%d", randomID),
			Description: fmt.Sprintf("Description%d", randomID),
			HeaderURI:   "https://headeruri",
			AvatarURI:   "https://avataruri",
		}
	}
	return mp
}

// Create a given number of random Elections
func CreateElections(size int, confidential, encrypted bool) []*TestElection {
	mp := make([]*TestElection, size)
	for i := 0; i < size; i++ {
		randomID := rand.Intn(10000000)
		mp[i] = &TestElection{
			Title:         fmt.Sprintf("Test%d", randomID),
			Description:   fmt.Sprintf("Description%d", randomID),
			Header:        fmt.Sprintf("Header%d", randomID),
			StreamURI:     fmt.Sprintf("Stream%d", randomID),
			EndDate:       time.Now().Add(24 * time.Hour),
			Confidential:  confidential,
			HiddenResults: encrypted,
		}
		for j := 0; j <= i; j++ {
			mp[i].Questions = append(mp[i].Questions, types.Question{
				Title:       fmt.Sprintf("Title%d", j),
				Description: fmt.Sprintf("Description%d", j),
			})
			for k := 0; k <= j; k++ {
				mp[i].Questions[j].Choices = append(mp[i].Questions[j].Choices, fmt.Sprintf("Choice%d", k))
			}
		}
	}
	return mp
}

// Create a given number of random organizations
func CreateDbOrganizations(size int) []*types.Organization {
	signers := CreateEthRandomKeysBatch(size)
	mp := make([]*types.Organization, size)
	for i := 0; i < size; i++ {
		mp[i] = &types.Organization{
			EthAddress:       signers[i].Address().Bytes(),
			EthPrivKeyCipher: []byte("ff"),
			HeaderURI:        "https://",
			AvatarURI:        "https://",
			PublicAPIToken:   signers[i].Address().String(),
			PublicAPIQuota:   1000,
			QuotaPlanID:      uuid.NullUUID{},
		}
	}
	return mp
}

// Create a given number of random Elections
func CreateDbElections(t *testing.T, size int) []*types.Election {
	var duration time.Duration
	var err error
	if duration, err = time.ParseDuration("1.5h"); err != nil {
		t.Error("unexpected, cannot parse duration")
	}
	mp := make([]*types.Election, size)
	for i := 0; i < size; i++ {
		randomID := rand.Intn(10000000)
		mp[i] = &types.Election{
			ProcessID:       []byte(fmt.Sprintf("%d", randomID)),
			Title:           fmt.Sprintf("Test%d", randomID),
			StartDate:       time.Now(),
			EndDate:         time.Now().Add(duration),
			Confidential:    true,
			HiddenResults:   true,
			MetadataPrivKey: nil,
		}
	}
	return mp
}

// CreateEthRandomKeysBatch creates a set of eth random signing keys
func CreateEthRandomKeysBatch(n int) []*ethereum.SignKeys {
	s := make([]*ethereum.SignKeys, n)
	for i := 0; i < n; i++ {
		s[i] = ethereum.NewSignKeys()
		if err := s[i].Generate(); err != nil {
			return nil
		}
	}
	return s
}

// RandDate creates a random date
func RandDate() time.Time {
	min := time.Date(1970, 1, 0, 0, 0, 0, 0, time.UTC).Unix()
	max := time.Date(2070, 1, 0, 0, 0, 0, 0, time.UTC).Unix()
	delta := max - min
	sec := rand.Int63n(delta) + min
	return time.Unix(sec, 0)
}

// RandBool creates a random bool
func RandBool() bool {
	return rand.Float32() < 0.5
}

func GetCSPSignature(t *testing.T, processId []byte, signer *ethereum.SignKeys) []byte {
	// extract public key as hexString, decode
	_, priv := signer.HexString()

	// create saltable private key
	saltedPrivKey, err := sk.NewSaltedKey(priv)
	if err != nil {
		t.Error(err)
	}
	salt := [sk.SaltSize]byte{}
	copy(salt[:], processId)
	// generate salted signature with compressed private key
	signature, err := saltedPrivKey.SignECDSA(salt, processId)
	if err != nil {
		t.Error(err)
	}
	return signature
}
