package main

import (
	"context"
	"time"

	"github.com/juju/cmd"
	"github.com/juju/gnuflag"
	errgo "gopkg.in/errgo.v1"
	"gopkg.in/macaroon-bakery.v2/bakery"
	"gopkg.in/macaroon-bakery.v2/bakery/checkers"
)

type newCommand struct {
	ops    []bakery.Op
	expiry time.Duration
}

func init() {
	register(&newCommand{})
}

func (c *newCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "new",
		Args:    "operation ...",
		Purpose: "Create a new macaroon and print it",
		Doc: `
The new command mints a new macaroon and prints it to stdout.

`[1:],
	}
}

func (c *newCommand) SetFlags(f *gnuflag.FlagSet) {
	// TODO allow specification of root key and id?
	f.DurationVar(&c.expiry, "expiry", 0, "expiry time of macaroon as a duration (default unlimited)")
}

func (c *newCommand) IsSuperCommand() bool {
	return false
}

func (c *newCommand) AllowInterspersedFlags() bool {
	return false
}

func (c *newCommand) Init(args []string) error {
	if len(args) < 1 {
		return errgo.Newf("at least one operation (in action:entity form) must be specified")
	}
	ops, err := parseOps(args)
	if err != nil {
		return errgo.Mask(err)
	}
	c.ops = ops
	return nil
}

func (c *newCommand) Run(cmdCtx *cmd.Context) error {
	oven, err := newOven(cmdCtx)
	if err != nil {
		return errgo.Mask(err)
	}
	ctx := context.Background()
	m, err := oven.NewMacaroon(ctx, bakery.Version3, nil, c.ops...)
	if err != nil {
		return errgo.Mask(err)
	}
	if c.expiry != 0 {
		err := m.AddCaveat(ctx, checkers.TimeBeforeCaveat(time.Now().Add(c.expiry).Round(time.Millisecond)), nil, nil)
		if err != nil {
			return errgo.Mask(err)
		}
	}
	data, err := formatJSON.marshalUnbound(bakery.Slice{m})
	if err != nil {
		return errgo.Mask(err)
	}
	cmdCtx.Stdout.Write(data)
	return nil
}
