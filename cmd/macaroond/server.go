package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/juju/httprequest"
	errgo "gopkg.in/errgo.v1"
	"gopkg.in/macaroon-bakery.v2-unstable/bakery"
	macaroon "gopkg.in/macaroon.v2-unstable"
)

var serverParams = httprequest.Server{
	ErrorMapper: errorToResponse,
}

type server struct {
	dir    string
	bakery *bakery.Bakery

	mu                 sync.Mutex
	encryptedMasterKey []byte
	masterKey          []byte
}

// needsPassword reports wh
func (srv *server) needsPassword() bool {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	return srv.encryptedMasterKey == nil
}

// checkPassword checks that the password is valid by decrypting
// the master key. It also sets the srv.masterKey from the decrypted key.
func (srv *server) checkPassword(password string) error {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if srv.encryptedMasterKey == nil {
		return errgo.Newf("no password set yet")
	}
	masterKey, err := decrypt(srv.encryptedMasterKey, password)
	if err != nil {
		return errgo.Mask(err)
	}
	if srv.masterKey == nil {
		srv.masterKey = masterKey
	} else {
		// Sanity check that the decrypted key is the same as
		// the one we already have.
		if !bytes.Equal(masterKey, srv.masterKey) {
			return errgo.Newf("key mismatch after decryption (should never happen)")
		}
	}
	return nil
}

func (srv *server) getMasterKey() ([]byte, error) {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if srv.masterKey == nil {
		return nil, errgo.Newf("locked - no password supplied yet")
	}
	return srv.masterKey, nil
}

func (srv *server) setPassword(oldPassword, newPassword string) error {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	return srv.setPassword0(oldPassword, newPassword)
}

func (srv *server) setPassword0(oldPassword, newPassword string) error {
	log.Printf("changing password from %q to %q", oldPassword, newPassword)
	encryptedMasterKey := srv.encryptedMasterKey
	if encryptedMasterKey == nil {
		// We don't have a key, so generate one.
		masterKey, err := randomBytes(24)
		if err != nil {
			return errgo.Mask(err)
		}
		encryptedMasterKey = encrypt(masterKey, "")
	}
	masterKey, err := decrypt(encryptedMasterKey, oldPassword)
	if err != nil {
		return errgo.Mask(err)
	}
	log.Printf("re-encrypting with new password %q", newPassword)
	// Re-encrypt with new password and write it.
	encryptedMasterKey = encrypt(masterKey, newPassword)
	if err := srv.writeEncryptedKey(encryptedMasterKey); err == nil {
		srv.masterKey = masterKey
		srv.encryptedMasterKey = encryptedMasterKey
		log.Printf("encrypted master key now %x", srv.encryptedMasterKey)
		return nil
	}
	if !os.IsExist(errgo.Cause(err)) {
		return errgo.Mask(err)
	}
	// The file already exists (someone must have been creating it at the same
	// time), so try again.
	return srv.setPassword0(oldPassword, newPassword)
}

func (srv *server) writeEncryptedKey(key []byte) error {
	return writeFile(srv.masterKeyPath(), key)
}

// readEncryptedMasterKey reads the root key and decrypts it.
func (srv *server) readEncryptedMasterKey() error {
	data, err := readFile(srv.masterKeyPath())
	if err != nil {
		return errgo.Mask(err)
	}
	srv.encryptedMasterKey = data
	return nil
}

func (srv *server) masterKeyPath() string {
	return filepath.Join(srv.dir, "masterkey")
}

func writeFile(path string, data []byte) error {
	// TODO write file atomically
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_EXCL|os.O_SYNC|os.O_CREATE, 0600)
	if err != nil {
		return errgo.Mask(err, os.IsExist)
	}
	defer f.Close()
	if _, err := f.Write([]byte(base64.RawStdEncoding.EncodeToString(data))); err != nil {
		return errgo.Mask(err)
	}
	return nil
}

// readFile reads the contents of the given file.
// It returns nil if the file does not exist.
func readFile(path string) ([]byte, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, errgo.Mask(err)
		}
		return nil, nil
	}
	data = bytes.TrimSpace(data)
	data, err = macaroon.Base64Decode(data)
	if err != nil {
		return nil, errgo.Notef(err, "invalid root key contents")
	}
	return data, nil
}

func randomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, fmt.Errorf("cannot generate %d random bytes: %v", n, err)
	}
	return b, nil
}
