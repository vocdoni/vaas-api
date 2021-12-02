package testcommon

import (
	"math/rand"
	"time"

	"go.vocdoni.io/api/types"
	"go.vocdoni.io/dvote/crypto/ethereum"
)

// CreateEntities a given number of random entities
func CreateOrganization(size int) ([]*ethereum.SignKeys, []*types.Organization) {
	signers := CreateEthRandomKeysBatch(size)
	mp := make([]*types.Organization, size)
	for i := 0; i < size; i++ {
		// retrieve entity ID
		mp[i] = &types.Organization{
			// ID: signers[i].Address().Bytes(),
			// EntityInfo: types.EntityInfo{
			// 	Email: randomdata.Email(),
			// 	Name:  randomdata.FirstName(2),
			// 	Size:  randomdata.Number(1001),
			// },
		}
	}
	return signers, mp
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
