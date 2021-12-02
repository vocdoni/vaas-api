package metadata

import (
	"fmt"

	"go.vocdoni.io/dvote/crypto/nacl"
)

func GeneratePrivateMetaKey() (privateKey []byte, _ error) {
	privKey, err := nacl.Generate(nil)
	if err != nil {
		return []byte{}, fmt.Errorf("could not generate private metadata key: %v", err)
	}
	return privKey.Bytes(), nil
}
