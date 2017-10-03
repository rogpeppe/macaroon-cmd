package main

import (
	"crypto/sha256"

	"github.com/rogpeppe/fastuuid"
	"golang.org/x/crypto/nacl/secretbox"
	"gopkg.in/errgo.v1"
)

var nonceGen = fastuuid.MustNewGenerator()

func encrypt(data []byte, password string) []byte {
	nonce := nonceGen.Next()
	key := sha256.Sum256([]byte(password))
	return secretbox.Seal(nonce[:], data, &nonce, &key)
}

func decrypt(data []byte, password string) ([]byte, error) {
	if len(data) < 24 {
		return nil, errgo.Newf("encrypted data is too small")
	}
	key := sha256.Sum256([]byte(password))
	var nonce [24]byte
	copy(nonce[:], data)
	plain, ok := secretbox.Open(nil, data[len(nonce):], &nonce, &key)
	if !ok {
		return nil, errgo.Newf("bad password %q", password)
	}
	return plain, nil
}
