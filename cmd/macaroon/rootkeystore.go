package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"io/ioutil"
	"os"
	"strings"
	"sync"

	errgo "gopkg.in/errgo.v1"
	"gopkg.in/macaroon-bakery.v2/bakery"
	macaroon "gopkg.in/macaroon.v2"

	"github.com/rogpeppe/macaroon-cmd/cmd/macaroond/macaroondclient"
)

var errNoAccessToken = errgo.Newf(`no macaroon access token found - use "macaroon login" to obtain one`)

var _ bakery.RootKeyStore = (*fileRootKeyStore)(nil)

const envToken = "MACAROON_ACCESS_TOKEN"

func newRootKeyStore() (bakery.RootKeyStore, error) {
	tok := os.Getenv(envToken)
	if tok == "" {
		return nil, errNoAccessToken
	}
	if path := strings.TrimPrefix(tok, "localfile:"); len(path) != len(tok) {
		return newFileRootKeyStore(path), nil
	}
	ms, err := parseUnboundMacaroons(tok)
	if err != nil {
		return nil, errgo.Notef(err, "invalid macaroon access token")
	}
	macLoc := ms[0].M().Location()
	loc := strings.SplitN(macLoc, " ", 2)
	if len(loc) != 2 {
		return nil, errgo.Notef(err, "access token location %q in incorrect format", macLoc)
	}
	netw, addr := loc[0], loc[1]
	// TODO discharge macaroons, as someone may have added 3rd party caveats to them.
	return macaroondclient.New(netw, addr, ms.Bind()), nil
}

// newFileRootKeyStore returns an implementation of
// Store that stores a single key inside a path with
// the given string.
func newFileRootKeyStore(path string) bakery.RootKeyStore {
	return &fileRootKeyStore{
		path: path,
	}
}

// TODO encrypt key at rest.
// TODO use a server that implements an oven-like API, so
// the command line apps never need to see the root keys.

type fileRootKeyStore struct {
	path string

	mu  sync.Mutex
	key []byte
}

var rootKeyId = []byte{'0'}

// Get implements Store.Get.
func (s *fileRootKeyStore) Get(_ context.Context, id []byte) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !bytes.Equal(id, rootKeyId) {
		return nil, bakery.ErrNotFound
	}
	if s.key != nil {
		return s.key, nil
	}
	key, err := s.readKey()
	if err != nil {
		if os.IsNotExist(errgo.Cause(err)) {
			return nil, bakery.ErrNotFound
		}
		return nil, errgo.Mask(err)
	}
	s.key = key
	return s.key, nil
}

func (s *fileRootKeyStore) readKey() ([]byte, error) {
	// TODO limit read size?
	data, err := ioutil.ReadFile(s.path)
	if err != nil {
		return nil, errgo.Mask(err, os.IsNotExist)
	}
	data = bytes.TrimSpace(data)
	data, err = macaroon.Base64Decode(data)
	if err != nil {
		return nil, errgo.Notef(err, "invalid root key contents")
	}
	return data, err
}

// RootKey implements Store.RootKey by always returning the same root key.
func (s *fileRootKeyStore) RootKey(context.Context) (rootKey, id []byte, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.key != nil {
		return s.key, []byte("0"), nil
	}
	key, err := s.readKey()
	if err == nil || !os.IsNotExist(errgo.Cause(err)) {
		return key, rootKeyId, errgo.Mask(err)
	}
	rootKey, err = randomBytes(24)
	if err != nil {
		return nil, nil, errgo.Mask(err)
	}
	f, err := os.OpenFile(s.path, os.O_WRONLY|os.O_EXCL|os.O_SYNC|os.O_CREATE, 0600)
	if err != nil {
		if !os.IsExist(err) {
			return nil, nil, errgo.Mask(err)
		}
		// The file already exists (someone must have been creating it at the same
		// time), so read it back.
		rootKey, err = s.readKey()
		if err != nil {
			return nil, nil, errgo.Mask(err)
		}
		s.key = rootKey
		return rootKey, rootKeyId, nil
	}
	defer f.Close()
	if _, err := f.Write([]byte(base64.RawStdEncoding.EncodeToString(rootKey))); err != nil {
		return nil, nil, errgo.Mask(err)
	}
	s.key = rootKey
	return rootKey, rootKeyId, nil
}
