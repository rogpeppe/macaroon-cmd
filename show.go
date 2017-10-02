package main

import (
	"encoding/json"
	"strings"

	"github.com/juju/cmd"
	"github.com/juju/gnuflag"
	errgo "gopkg.in/errgo.v1"
	"gopkg.in/macaroon-bakery.v2-unstable/bakery"
	macaroon "gopkg.in/macaroon.v2-unstable"
)

type showCommand struct {
	format           formatFlag
	unboundMacaroons bakery.Slice
	boundMacaroons   macaroon.Slice
}

func init() {
	register(&showCommand{})
}

func (c *showCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "show",
		Args:    "macaroons",
		Purpose: "Print macaroons in different formats",
		Doc:     `doc TODO`,
	}
}

func (c *showCommand) SetFlags(f *gnuflag.FlagSet) {
	c.format = formatJSON | formatRaw

	f.Var(&c.format, "f", "Format to print bound macaroons in")
	f.Var(&c.format, "format", "")
}

func (c *showCommand) Init(args []string) error {
	if len(args) != 1 {
		return errgo.New("need macaroon argument")
	}
	var err error
	c.unboundMacaroons, c.boundMacaroons, err = parseEither(args[0])
	if err != nil {
		return errgo.Mask(err)
	}
	return nil
}

func (c *showCommand) Run(cmdCtx *cmd.Context) error {
	// TODO provide a way of formatting the JSON prettily.
	var data []byte
	switch {
	case len(c.boundMacaroons) > 0:
		var err error
		data, err = c.format.marshalBound(c.boundMacaroons)
		if err != nil {
			return errgo.Mask(err)
		}
	case len(c.unboundMacaroons) > 0:
		var err error
		data, err = c.format.marshalUnbound(c.unboundMacaroons)
		if err != nil {
			return errgo.Mask(err)
		}
	default:
		panic("unreachable")
	}
	cmdCtx.Stdout.Write(data)
	return nil
}

func (c *showCommand) IsSuperCommand() bool {
	return false
}

func (c *showCommand) AllowInterspersedFlags() bool {
	return false
}

// parseEither parses macaroons in either bound or unbound format.
func parseEither(s string) (bakery.Slice, macaroon.Slice, error) {
	if strings.HasPrefix(s, "unbound:") {
		ms, err := parseUnboundMacaroons(s)
		if err != nil {
			return nil, nil, errgo.Mask(err)
		}
		return ms, nil, nil
	}
	if s == "" {
		return nil, nil, errgo.Newf("no macaroons found")
	}
	data, err := maybeBase64Decode(s)
	if err != nil {
		return nil, nil, errgo.Mask(err)
	}
	var bms bakery.Slice
	switch data[0] {
	case '{':
		// It's a single macaroon.
		var m bakery.Macaroon
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, nil, errgo.Notef(err, "cannot unmarshal single JSON macaroon")
		}
		bms = bakery.Slice{&m}
	case '[':
		// It's a slice of macaroons.
		if err := json.Unmarshal(data, &bms); err != nil {
			return nil, nil, errgo.Notef(err, "cannot unmarshal JSON macaroon slice")
		}
	default:
		// It's probably in binary format - it must be bound.
		// Note that the binary format for a slice is that of
		// multiple concatenated single macaroons, so we
		// can use UnmarshalBinary to parse both possibilities.
		var ms macaroon.Slice
		if err := ms.UnmarshalBinary(data); err != nil {
			return nil, nil, errgo.Notef(err, "cannot unmarshal binary macaroons")
		}
		return nil, ms, nil
	}
	// If none of the macaroons are bakery V3 macaroons, then treat the
	// slice as bound.
	for _, m := range bms {
		if m.Version() >= bakery.Version3 {
			return bms, nil, nil
		}
	}
	ms := make(macaroon.Slice, len(bms))
	for i, m := range bms {
		ms[i] = m.M()
	}
	return nil, ms, nil
}
