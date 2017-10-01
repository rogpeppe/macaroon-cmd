package main

import (
	"context"
	"time"

	"github.com/juju/cmd"
	"github.com/juju/gnuflag"
	errgo "gopkg.in/errgo.v1"
	"gopkg.in/macaroon-bakery.v2-unstable/bakery"
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
		Doc:     `doc TODO`,
	}
}

func (c *newCommand) SetFlags(f *gnuflag.FlagSet) {
	// TODO allow specification of root key and id?
	f.DurationVar(&c.expiry, "expiry", time.Hour, "expiry time of macaroon as a duration")
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
	m, err := oven.NewMacaroon(ctx, bakery.Version3, time.Now().Add(c.expiry).Round(time.Millisecond), nil, c.ops...)
	if err != nil {
		return errgo.Mask(err)
	}
	if err := printMacaroon(cmdCtx.Stdout, m); err != nil {
		return errgo.Mask(err)
	}
	return nil
}
