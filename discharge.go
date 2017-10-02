package main

import (
	"context"

	"github.com/juju/cmd"
	"github.com/juju/gnuflag"
	errgo "gopkg.in/errgo.v1"
	"gopkg.in/macaroon-bakery.v2-unstable/bakery"
	"gopkg.in/macaroon-bakery.v2-unstable/httpbakery"
)

type dischargeCommand struct {
	macaroons bakery.Slice
}

func init() {
	register(&dischargeCommand{})
}

func (c *dischargeCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "discharge",
		Args:    "macaroons",
		Purpose: "Discharge all third party caveats and print resulting macaroons",
		Doc:     `doc TODO`,
	}
}

func (c *dischargeCommand) SetFlags(f *gnuflag.FlagSet) {}

func (c *dischargeCommand) Init(args []string) error {
	if len(args) != 1 {
		return errgo.New("need macaroon argument")
	}
	ms, err := parseUnboundMacaroons(args[0])
	if err != nil {
		return errgo.Mask(err)
	}
	c.macaroons = ms
	return nil
}

func (c *dischargeCommand) Run(cmdCtx *cmd.Context) error {
	ctx := context.Background()
	client := httpbakery.NewClient()
	client.AddInteractor(httpbakery.WebBrowserInteractor{})
	// TODO use local agent key when available.
	ms, err := client.DischargeAllUnbound(ctx, c.macaroons)
	if err != nil {
		return errgo.Mask(err)
	}
	data, err := formatJSON.marshalUnbound(ms)
	if err != nil {
		return errgo.Mask(err)
	}
	cmdCtx.Stdout.Write(data)
	return nil
}

func (c *dischargeCommand) IsSuperCommand() bool {
	return false
}

func (c *dischargeCommand) AllowInterspersedFlags() bool {
	return false
}
