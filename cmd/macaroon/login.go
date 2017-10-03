package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/juju/cmd"
	"github.com/juju/errors"
	"github.com/juju/gnuflag"
	"github.com/rogpeppe/macaroon-cmd/cmd/macaroond/macaroondclient"
	"github.com/rogpeppe/macaroon-cmd/params"
	"golang.org/x/crypto/ssh/terminal"
	errgo "gopkg.in/errgo.v1"
	"gopkg.in/macaroon-bakery.v2-unstable/bakery"
)

type loginCommand struct {
	network string
	addr    string
}

func init() {
	register(&loginCommand{})
}

func (c *loginCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "login",
		Purpose: "Log in to macaroond server",
		Doc:     `doc TODO`,
	}
}

func (c *loginCommand) SetFlags(f *gnuflag.FlagSet) {
	f.StringVar(&c.network, "t", "unix", "network to use to connect to server (unix, tcp or file)")
	f.StringVar(&c.addr, "addr", "/tmp/macaroond.socket", "address or socket path to connect to, or file path for local")
}

func (c *loginCommand) Init(args []string) error {
	return nil
}

func (c *loginCommand) Run(cmdCtx *cmd.Context) error {
	ctx := context.Background()
	// TODO Check whether we're already logged in ?
	client := macaroondclient.New(c.network, c.addr, nil)
	// Try to log in with no password in case the initial password has
	// not been set yet.
	_, err := client.Login(ctx, "")
	if err == nil {
		return errgo.Newf("unexpected success logging in with empty password")
	}
	var password string
	if errgo.Cause(err) == params.ErrInitialPasswordNeeded {
		fmt.Fprintf(cmdCtx.Stdout, "Choose initial password for macaroon root keys\n")
		pw1, err := readPassword(cmdCtx, "Password: ")
		if err != nil {
			return errgo.Mask(err)
		}
		pw2, err := readPassword(cmdCtx, "Same password: ")
		if err != nil {
			return errgo.Mask(err)
		}
		if pw1 != pw2 {
			return errgo.Newf("Password mismatch")
		}
		if err := client.SetPassword(ctx, &params.SetPasswordRequest{
			NewPassword: pw1,
		}); err != nil {
			return errgo.Notef(err, "cannot set password")
		}
		password = pw1
	} else {
		pw, err := readPassword(cmdCtx, "Password: ")
		if err != nil {
			return errgo.Mask(err)
		}
		password = pw
	}
	m, err := client.Login(ctx, password)
	if err != nil {
		return errgo.Notef(err, "cannot log in with new password")
	}
	m.M().SetLocation(c.network + " " + c.addr)
	tok, err := formatJSON.marshalUnbound(bakery.Slice{m})
	if err != nil {
		return errgo.Mask(err)
	}
	fmt.Fprintf(cmdCtx.Stdout, "export %s=%s\n", envToken, tok)
	return nil
}

func (c *loginCommand) IsSuperCommand() bool {
	return false
}

func (c *loginCommand) AllowInterspersedFlags() bool {
	return false
}

func readPassword(cmdCtx *cmd.Context, prompt string) (string, error) {
	fmt.Fprintf(cmdCtx.Stderr, "%s", prompt)
	stdin := cmdCtx.Stdin
	if f, ok := stdin.(*os.File); ok && terminal.IsTerminal(int(f.Fd())) {
		password, err := terminal.ReadPassword(int(f.Fd()))
		fmt.Fprintf(cmdCtx.Stderr, "\n")
		return string(password), err
	}
	return readLine(stdin)
}

func readLine(stdin io.Reader) (string, error) {
	// Read one byte at a time to avoid reading beyond the delimiter.
	line, err := bufio.NewReader(byteAtATimeReader{stdin}).ReadString('\n')
	if err != nil {
		return "", errors.Trace(err)
	}
	return line[:len(line)-1], nil
}

type byteAtATimeReader struct {
	io.Reader
}

// Read is part of the io.Reader interface.
func (r byteAtATimeReader) Read(out []byte) (int, error) {
	return r.Reader.Read(out[:1])
}
