package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/juju/cmd"
	errgo "gopkg.in/errgo.v1"
	"gopkg.in/macaroon-bakery.v2-unstable/bakery"
	"gopkg.in/macaroon-bakery.v2-unstable/httpbakery"
	"gopkg.in/macaroon.v2-unstable"
)

const rootKeyFilePath = "/tmp/maca-root-key"

func newOven(ctx *cmd.Context) (*bakery.Oven, error) {
	return bakery.NewOven(bakery.OvenParams{
		RootKeyStoreForOps: func([]bakery.Op) bakery.RootKeyStore {
			// TODO use connection to server etc.
			return newFileRootKeyStore(rootKeyFilePath)
		},
		// TODO Namespace
		// TODO OpsStore - store the ops in the server too
		// TODO Key - store the key in the server too
		Locator: httpbakery.NewThirdPartyLocator(nil, nil),
	}), nil
}

func parseOp(s string) (bakery.Op, error) {
	p := strings.SplitN(s, ":", 2)
	if len(p) < 2 {
		return bakery.Op{}, errgo.Newf("invalid operation %q (must be in action:entity form)")
	}
	op := bakery.Op{
		Entity: p[1],
		Action: p[0],
	}
	if op.Entity == "" {
		return bakery.Op{}, errgo.Newf("operation %q has empty entity")
	}
	if op.Action == "" {
		return bakery.Op{}, errgo.Newf("operation %q has empty action")
	}
	return op, nil
}

func parseOps(args []string) ([]bakery.Op, error) {
	if len(args) == 0 {
		return nil, nil
	}
	ops := make([]bakery.Op, len(args))
	for i, arg := range args {
		op, err := parseOp(arg)
		if err != nil {
			return nil, errgo.Mask(err)
		}
		ops[i] = op
	}
	return ops, nil
}

// parseUnboundMacaroons parses a macaroon or macaroons from the given
// string. The macaroons are expected to be unbound.
// It accepts:
// - a JSON string containing a single macaroon.
// - a JSON string containing an array of macaroons.
// - a base64-encoded string containing either of the above.
// - any of the above prefixed with "unbound:".
//
// On success, there will always be at least one macaroon
// in the returned slice.
func parseUnboundMacaroons(s string) (bakery.Slice, error) {
	s = strings.TrimPrefix(s, "unbound:")
	if s == "" {
		return nil, errgo.Newf("no macaroons found")
	}
	var data []byte
	if s[0] != '[' && s[0] != '{' {
		// It's base64-encoded.
		data1, err := macaroon.Base64Decode([]byte(s))
		if err != nil {
			return nil, errgo.Notef(err, "invalid base64-encoding of macaroon")
		}
		data = data1
	} else {
		data = []byte(s)
	}
	if s[0] == '{' {
		var m bakery.Macaroon
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, errgo.Mask(err)
		}
		return bakery.Slice{&m}, nil
	}
	var ms bakery.Slice
	if err := json.Unmarshal(data, &ms); err != nil {
		return nil, errgo.Mask(err)
	}
	if len(ms) == 0 {
		return nil, errgo.Newf("empty macaroon array")
	}
	return ms, nil
}

func randomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, fmt.Errorf("cannot generate %d random bytes: %v", n, err)
	}
	return b, nil
}

func printUnboundMacaroons(w io.Writer, ms bakery.Slice) error {
	var x interface{} = ms
	if len(ms) == 1 {
		x = ms[0]
	}
	data, err := json.Marshal(x)
	if err != nil {
		return errgo.Mask(err)
	}
	if _, err := w.Write([]byte("unbound:" + base64.RawStdEncoding.EncodeToString(data) + "\n")); err != nil {
		return errgo.Notef(err, "cannot write macaroon")
	}
	return nil
}

type publicKeyFlag struct {
	key *bakery.PublicKey
}

func (f *publicKeyFlag) String() string {
	if f.key == nil {
		return ""
	}
	return f.key.String()
}

func (f *publicKeyFlag) Set(s string) error {
	if s == "" {
		f.key = nil
		return nil
	}
	var k bakery.PublicKey
	if err := k.UnmarshalText([]byte(s)); err != nil {
		return errgo.Mask(err)
	}
	f.key = &k
	return nil
}
