package util

import (
	"fmt"
	"strings"

	"go.vocdoni.io/dvote/crypto/ethereum"
	dvoteutil "go.vocdoni.io/dvote/util"
)

// PubKeyToEntityID retrieves entity ID from a public key
func PubKeyToEntityID(pubKey []byte) ([]byte, error) {
	address, err := ethereum.AddrFromPublicKey(pubKey)
	if err != nil {
		return nil, fmt.Errorf("cannot get entityID: %w", err)
	}
	return address.Bytes(), nil
}

func ValidPubKey(pubKey []byte) bool {
	return len(pubKey) == ethereum.PubKeyLengthBytes
}

func HexPrefixed(s string) string {
	if !strings.HasPrefix(s, "0x") {
		return fmt.Sprintf("0x%s", s)
	}
	return s
}

func GenerateBearerToken() string {
	return dvoteutil.RandomHex(32)
}
