package main

import (
	"crypto/rand"
	"fmt"
	"strings"

	"github.com/juju/cmd"
	errgo "gopkg.in/errgo.v1"
	"gopkg.in/macaroon-bakery.v2-unstable/bakery"
	"gopkg.in/macaroon-bakery.v2-unstable/httpbakery"
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

func randomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, fmt.Errorf("cannot generate %d random bytes: %v", n, err)
	}
	return b, nil
}
