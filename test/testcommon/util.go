package testcommon

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"go.vocdoni.io/api/types"
	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/log"
)

func CreateIntegrators(size int) []*types.Integrator {
	mp := make([]*types.Integrator, size)
	for i := 0; i < size; i++ {
		randomID := rand.Intn(10000000)
		// retrieve entity ID
		mp[i] = &types.Integrator{
			SecretApiKey: []byte(fmt.Sprintf("%d", randomID)),
			Name:         fmt.Sprintf("Test%d", randomID),
			Email:        fmt.Sprintf("mail%d@mail.org", randomID),
			CspUrlPrefix: "csp.vocdoni.net",
			CspPubKey:    []byte("ff"),
		}
	}
	return mp
}

// Create a given number of random organizations
func CreateOrganizations(size int) []*types.Organization {
	signers := CreateEthRandomKeysBatch(size)
	mp := make([]*types.Organization, size)
	for i := 0; i < size; i++ {
		mp[i] = &types.Organization{
			EthAddress:        signers[i].Address().Bytes(),
			EthPrivKeyCicpher: []byte("ff"),
			HeaderURI:         "https://",
			AvatarURI:         "https://",
			PublicAPIToken:    signers[i].Address().String(),
			PublicAPIQuota:    1000,
			QuotaPlanID:       uuid.NullUUID{},
		}
	}
	return mp
}

// Create a given number of random Elections
func CreateElections(size int) []*types.Election {
	var duration time.Duration
	var err error
	if duration, err = time.ParseDuration("1.5h"); err != nil {
		log.Error("unexpected, cannot parse duration")
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
