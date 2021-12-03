package util

import (
	"crypto/rand"
	"fmt"
	"io"

	"go.vocdoni.io/dvote/crypto/nacl"
	"golang.org/x/crypto/nacl/secretbox"
)

func GeneratePrivateMetaKey() (privateKey []byte, _ error) {
	privKey, err := nacl.Generate(nil)
	if err != nil {
		return []byte{}, fmt.Errorf("could not generate private metadata key: %v", err)
	}
	return privKey.Bytes(), nil
}

// encrypt using symetric key
func EncryptSymmetric(msg, key []byte) ([]byte, error) {
	var nonce [24]byte
	var paddedKey [32]byte
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		return nil, err
	}
	if len(key) <= 32 {
		copy(paddedKey[:], key)
	} else {
		copy(paddedKey[:], key[0:32])
	}
	return secretbox.Seal(nonce[:], msg, &nonce, &paddedKey), nil
}

// decrypt using symetric key
func DecryptSymmetric(msg, key []byte) ([]byte, bool) {
	var paddedKey [32]byte
	if msg == nil {
		return nil, false
	}
	var decryptNonce [24]byte
	copy(decryptNonce[:], msg[:24])
	if len(key) <= 32 {
		copy(paddedKey[:], key)
	} else {
		copy(paddedKey[:], key[0:32])
	}
	return secretbox.Open(nil, msg[24:], &decryptNonce, &paddedKey)
}
