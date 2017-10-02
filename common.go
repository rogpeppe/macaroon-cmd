package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/juju/cmd"
	"github.com/juju/loggo"
	errgo "gopkg.in/errgo.v1"
	"gopkg.in/macaroon-bakery.v2-unstable/bakery"
	"gopkg.in/macaroon-bakery.v2-unstable/httpbakery"
	"gopkg.in/macaroon.v2-unstable"
)

var logger = loggo.GetLogger("macaroon-cmd")

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
// - a JSON string containing a object in bakery.Macaroon or macaroon.Macaroon format.
// - a JSON string containing an array of macaroons in bakery.Macaroon format.
// - a base64-encoded string containing any of the above.
// - any of the above prefixed with "unbound:".
//
// On success, there will always be at least one macaroon
// in the returned slice.
func parseUnboundMacaroons(s string) (bakery.Slice, error) {
	// TODO would it be useful to support a binary-encoded single
	// macaroon too.
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
	for _, m := range ms {
		if m.Version() < bakery.Version3 {
			return nil, errgo.Newf("array of bound macaroons is not allowed (got version %d, want %d)", m.Version(), bakery.Version3)
		}
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

const (
	formatJSON formatFlag = iota
	formatBinary

	formatRaw formatFlag = 1 << 3
)

type formatFlag int

func (f formatFlag) String() string {
	s := ""
	switch f &^ formatRaw {
	case formatJSON:
		s = "json"
	case formatBinary:
		s = "binary"
	default:
		s = "unknown"
	}
	if f&formatRaw != 0 {
		s = "raw" + s
	}
	return s
}

func (f *formatFlag) Set(s string) error {
	var fv formatFlag
	s1 := strings.TrimPrefix(s, "raw")
	if len(s1) != len(s) {
		fv |= formatRaw
	}
	switch s1 {
	case "json":
		fv |= formatJSON
	case "binary":
		fv |= formatBinary
	default:
		return errgo.Newf("unrecognized format %q", s)
	}
	*f = fv
	return nil
}

func (f formatFlag) marshalUnbound(ms bakery.Slice) ([]byte, error) {
	var data []byte
	switch f &^ formatRaw {
	case formatJSON:
		var err error
		data, err = json.Marshal(ms)
		if err != nil {
			return nil, errgo.Mask(err)
		}
	case formatBinary:
		return nil, errgo.Newf("cannot format unbound macaroons in binary format")
	default:
		panic(errgo.Newf("unknown format %d", f))
	}
	if f&formatRaw != 0 {
		return data, nil
	}
	return []byte("unbound:" + base64.RawStdEncoding.EncodeToString(data) + "\n"), nil
}

func (f formatFlag) marshalBound(ms macaroon.Slice) ([]byte, error) {
	var data []byte
	switch f &^ formatRaw {
	case formatJSON:
		var err error
		data, err = json.Marshal(ms)
		if err != nil {
			return nil, errgo.Mask(err)
		}
	case formatBinary:
		var err error
		data, err = ms.MarshalBinary()
		if err != nil {
			return nil, errgo.Mask(err)
		}
	default:
		panic(errgo.Newf("unknown format %d", f))
	}
	if f&formatRaw != 0 {
		return data, nil
	}
	return []byte(base64.RawStdEncoding.EncodeToString(data) + "\n"), nil
}
