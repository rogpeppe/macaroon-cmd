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

// unboundPrefix is the prefix that unbound macaroons get so we can
// easily tell the difference even when base-64 encoded. Note that we
// can't use a colon because then it wouldn't be easy to distinguish
// between operations and macaroons.
const unboundPrefix = "unbound%"

func newOven(ctx *cmd.Context) (*bakery.Oven, error) {
	rks, err := newRootKeyStore()
	if err != nil {
		return nil, errgo.Mask(err)
	}
	return bakery.NewOven(bakery.OvenParams{
		RootKeyStoreForOps: func([]bakery.Op) bakery.RootKeyStore {
			return rks
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
//
// It accepts one of:
// - a JSON object in bakery.Macaroon or macaroon.Macaroon format.
// - a JSON string containing an array of macaroons in bakery.Macaroon format.
// - a base64-encoded string containing any of the above.
// - any of the above prefixed with unboundPrefix.
//
// On success, there will always be at least one macaroon
// in the returned slice.
func parseUnboundMacaroons(s string) (bakery.Slice, error) {
	// TODO would it be useful to support a binary-encoded single
	// macaroon too.
	s = strings.TrimPrefix(s, unboundPrefix)
	if s == "" {
		return nil, errgo.Newf("no macaroons found")
	}
	data, err := maybeBase64Decode(s)
	if err != nil {
		return nil, errgo.Mask(err)
	}
	if data[0] == '{' {
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

// maybeBase64Decode decodes s as base64 if it doesn't look
// like a JSON object.
func maybeBase64Decode(s string) ([]byte, error) {
	if len(s) > 0 && (s[0] == '[' || s[0] == '{') {
		return []byte(s), nil
	}
	// It's base64-encoded.
	data, err := macaroon.Base64Decode([]byte(s))
	if err != nil {
		return nil, errgo.Notef(err, "invalid base64-encoding of macaroon")
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
		if len(ms) == 1 {
			// A slice with a single element formats as that element.
			data, err = json.Marshal(ms[0])
		} else {
			data, err = json.Marshal(ms)
		}
		if err != nil {
			return nil, errgo.Mask(err)
		}
	case formatBinary:
		return nil, errgo.Newf("cannot format unbound macaroons in binary format")
	default:
		panic(errgo.Newf("unknown format %d", f))
	}
	if f&formatRaw != 0 {
		// JSON gets a newline even if is raw.
		if f&^formatRaw == formatJSON {
			data = append(data, '\n')
		}
		return data, nil
	}
	return []byte(unboundPrefix + base64.RawStdEncoding.EncodeToString(data) + "\n"), nil
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
