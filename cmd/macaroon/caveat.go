package main

import (
	"context"

	"github.com/juju/cmd"
	"github.com/juju/gnuflag"
	errgo "gopkg.in/errgo.v1"
	"gopkg.in/macaroon-bakery.v2/bakery"
	"gopkg.in/macaroon-bakery.v2/bakery/checkers"
	"gopkg.in/macaroon-bakery.v2/httpbakery"
)

type caveatCommand struct {
	location  string
	publicKey publicKeyFlag
	macaroons bakery.Slice
	insecure  bool
	condition string
	version   bakery.Version
}

func init() {
	register(&caveatCommand{})
}

func (c *caveatCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "caveat",
		Args:    "macaroons condition",
		Purpose: "Add a caveat to a macaroon",
		Doc:     `doc TODO`,
	}
}

func (c *caveatCommand) SetFlags(f *gnuflag.FlagSet) {
	// TODO allow specification of root key and id?
	f.StringVar(&c.location, "3", "", "Add third party caveat with the specified location")
	f.Var(&c.publicKey, "public-key", "For third party caveat, base64 public key of third party (discovered automatically if not specified)")
	f.IntVar((*int)(&c.version), "version", int(bakery.Version2), "bakery version of third party") // TODO use Version3?
	f.BoolVar(&c.insecure, "insecure", false, "allow non-secure public key retrieval (intended only for testing)")
	// TODO allow specification of namespace for third party caveat?
}

func (c *caveatCommand) IsSuperCommand() bool {
	return false
}

func (c *caveatCommand) AllowInterspersedFlags() bool {
	return false
}

func (c *caveatCommand) Init(args []string) error {
	if len(args) != 2 {
		return errgo.New("need macaroon and condition arguments")
	}
	ms, err := parseUnboundMacaroons(args[0])
	if err != nil {
		return errgo.Mask(err)
	}
	c.macaroons = ms
	c.condition = args[1]
	return nil
}

func (c *caveatCommand) Run(cmdCtx *cmd.Context) error {
	ctx := context.Background()
	cav := checkers.Caveat{
		Condition: c.condition,
		Location:  c.location,
	}
	var key *bakery.KeyPair
	var loc bakery.ThirdPartyLocator
	if cav.Location != "" {
		if c.publicKey.key == nil {
			loc1 := httpbakery.NewThirdPartyLocator(nil, nil)
			if c.insecure {
				loc1.AllowInsecure()
			}
			loc = loc1
		} else {
			loc1 := bakery.NewThirdPartyStore()
			loc1.AddInfo(c.location, bakery.ThirdPartyInfo{
				PublicKey: *c.publicKey.key,
				Version:   c.version,
			})
			loc = loc1
		}
		key1, err := bakery.GenerateKey()
		if err != nil {
			return errgo.Mask(err)
		}
		key = key1
	}
	if err := c.macaroons[0].AddCaveat(ctx, cav, key, loc); err != nil {
		return errgo.Mask(err)
	}
	data, err := formatJSON.marshalUnbound(c.macaroons)
	if err != nil {
		return errgo.Mask(err)
	}
	cmdCtx.Stdout.Write(data)
	return nil
}
