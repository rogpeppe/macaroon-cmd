package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/juju/cmd"
	"github.com/juju/gnuflag"
	errgo "gopkg.in/errgo.v1"
	"gopkg.in/macaroon-bakery.v2/bakery"
	"gopkg.in/macaroon-bakery.v2/bakery/checkers"
	macaroon "gopkg.in/macaroon.v2"
)

type checkCommand struct {
	ops          []bakery.Op
	macaroonArgs []string
}

func init() {
	register(&checkCommand{})
}

func (c *checkCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "check",
		Args:    "op... [macaroons...]",
		Purpose: "Check validity of macaroons",
		Doc:     `doc TODO`,
	}
}

func (c *checkCommand) SetFlags(f *gnuflag.FlagSet) {
	// TODO allow a namespace to be specified.
}

func (c *checkCommand) Init(args []string) error {
	for i, arg := range args {
		if !strings.Contains(arg, ":") {
			// Don't parse the macaroons here, because we can still succeed
			// even when some macaroons are invalid.
			c.macaroonArgs = args[i:]
			break
		}
		op, err := parseOp(arg)
		if err != nil {
			return errgo.Notef(err, "invalid operation at argument %d", i+1)
		}
		c.ops = append(c.ops, op)
	}
	if len(c.ops) == 0 {
		return errgo.Newf("no operations specified")
	}
	return nil
}

func (c *checkCommand) Run(cmdCtx *cmd.Context) error {
	ctx := context.Background()
	var mss []macaroon.Slice
	for i, arg := range c.macaroonArgs {
		bms, ms, err := parseEither(arg)
		if err != nil {
			fmt.Fprintf(cmdCtx.Stderr, "cannot parse macaroon argument %d: %v\n", i+1, err)
			continue
		}
		if bms != nil {
			fmt.Fprintf(cmdCtx.Stderr, "unbound macaroon provided - check requires bound macaroon (see use command)\n")
			continue
		}
		mss = append(mss, ms)
	}
	// TODO provide more first party caveat checkers, and the facility to invoke
	// commands to check caveats.
	fpChecker := &firstPartyChecker{
		underlying: checkers.New(nil),
	}
	oven, err := newOven(cmdCtx)
	if err != nil {
		return errgo.Mask(err)
	}
	checker := bakery.NewChecker(bakery.CheckerParams{
		MacaroonVerifier: oven,
		Checker:          fpChecker,
	}).Auth(mss...)
	if checker.FirstPartyCaveatChecker != fpChecker {
		panic(errgo.Newf("unexpected checker %T", checker.FirstPartyCaveatChecker))
	}
	if _, err := checker.Allow(ctx, c.ops...); err != nil {
		return errgo.Mask(err)
	}
	if len(fpChecker.unknownConditions) == 0 {
		return nil
	}
	for _, cond := range fpChecker.unknownConditions {
		fmt.Fprintf(cmdCtx.Stdout, "caveat: %s\n", cond)
	}
	return nil
}

func (c *checkCommand) IsSuperCommand() bool {
	return false
}

func (c *checkCommand) AllowInterspersedFlags() bool {
	return false
}

// firstPartyChecker wraps a bakery.FirstPartyCaveatChecker by
// appending any unknown caveats to unknownCaveats.
type firstPartyChecker struct {
	unknownConditions []string
	underlying        bakery.FirstPartyCaveatChecker
}

func (c *firstPartyChecker) CheckFirstPartyCaveat(ctx context.Context, cav string) error {
	err := c.underlying.CheckFirstPartyCaveat(ctx, cav)
	if err == nil {
		return nil
	}
	if errgo.Cause(err) == checkers.ErrCaveatNotRecognized {
		c.unknownConditions = append(c.unknownConditions, cav)
		return nil
	}
	return errgo.Mask(err, errgo.Is(checkers.ErrCaveatNotRecognized))
}

func (c *firstPartyChecker) Namespace() *checkers.Namespace {
	return c.underlying.Namespace()
}
