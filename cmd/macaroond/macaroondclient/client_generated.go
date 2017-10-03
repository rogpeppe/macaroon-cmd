// The code in this file was automatically generated by running httprequest-generate-client.
// DO NOT EDIT

package client

import (
	"github.com/juju/httprequest"
	"github.com/rogpeppe/macaroon-cmd/internal/params"
	"golang.org/x/net/context"
)

type client struct {
	Client httprequest.Client
}

func (c *client) Access(ctx context.Context, p *params.AccessRequest) (*params.AccessResponse, error) {
	var r *params.AccessResponse
	err := c.Client.Call(ctx, p, &r)
	return r, err
}

func (c *client) FindRootKey(ctx context.Context, p *params.FindRootKeyRequest) (*params.FindRootKeyResponse, error) {
	var r *params.FindRootKeyResponse
	err := c.Client.Call(ctx, p, &r)
	return r, err
}

func (c *client) NewRootKey(ctx context.Context, p *params.NewRootKeyRequest) (*params.NewRootKeyResponse, error) {
	var r *params.NewRootKeyResponse
	err := c.Client.Call(ctx, p, &r)
	return r, err
}

func (c *client) SetPassword(ctx context.Context, p *params.SetPasswordRequest) error {
	return c.Client.Call(ctx, p, nil)
}