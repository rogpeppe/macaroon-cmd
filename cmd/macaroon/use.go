package main

import (
	"github.com/juju/cmd"
	"github.com/juju/gnuflag"
	errgo "gopkg.in/errgo.v1"
	"gopkg.in/macaroon-bakery.v2/bakery"
)

type useCommand struct {
	format    formatFlag
	force     bool
	macaroons bakery.Slice
}

func init() {
	register(&useCommand{})
}

func (c *useCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "use",
		Args:    "macaroons",
		Purpose: "Print macaroons suitable for using in a request",
		Doc:     `doc TODO`,
	}
}

func (c *useCommand) SetFlags(f *gnuflag.FlagSet) {
	c.format = formatBinary
	f.Var(&c.format, "f", "Format to print bound macaroons in")
	f.Var(&c.format, "format", "")
	f.BoolVar(&c.force, "force", false, "Continue even if all discharges are not present")
}

func (c *useCommand) Init(args []string) error {
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

func (c *useCommand) Run(cmdCtx *cmd.Context) error {
	if !c.force {
		if err := checkDischargesIncluded(c.macaroons); err != nil {
			return errgo.Mask(err)
		}
	}
	data, err := c.format.marshalBound(c.macaroons.Bind())
	if err != nil {
		return errgo.Mask(err)
	}
	cmdCtx.Stdout.Write(data)
	return nil
}

func (c *useCommand) IsSuperCommand() bool {
	return false
}

func (c *useCommand) AllowInterspersedFlags() bool {
	return false
}

func checkDischargesIncluded(ms bakery.Slice) error {
	ids := make(map[string]bool)
	for _, m := range ms[1:] {
		ids[string(m.M().Id())] = true
	}
	missing := 0
	for _, m := range ms {
		for _, cav := range m.M().Caveats() {
			if len(cav.VerificationId) != 0 && !ids[string(cav.Id)] {
				missing++
			}
		}
	}
	if missing > 0 {
		return errgo.Newf("%d discharge macaroons are missing - use discharge to acquire them", missing)
	}
	return nil
}
