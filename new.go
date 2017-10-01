package main

import (
	"github.com/juju/cmd"
	"github.com/juju/gnuflag"
	errgo "gopkg.in/errgo.v1"
	"gopkg.in/macaroon-bakery.v2-unstable/bakery"
)

type newCommand struct {
	ops []bakery.Op
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

func (c *newCommand) Run(ctx *cmd.Context) error {
	return nil
}
