package macaroondclient

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net"
	"net/http"

	"github.com/juju/httprequest"
	"github.com/rogpeppe/macaroon-cmd/params"

	errgo "gopkg.in/errgo.v1"
	"gopkg.in/macaroon-bakery.v2/bakery"
	"gopkg.in/macaroon-bakery.v2/httpbakery"
	"gopkg.in/macaroon.v2"
)

//go:generate httprequest-generate-client github.com/rogpeppe/macaroon-cmd/cmd/macaroond handler client

type Client struct {
	client
	accessToken string
}

// New returns a new client that uses the given token for
// access. If accessToken is nil, the only methods
// that may be called are Login and ChangePassword.
func New(netw, addr string, accessToken macaroon.Slice) *Client {
	var c Client
	if netw == "tcp" {
		c.Client.BaseURL = "http://" + addr
	} else {
		// For decent errors only - address is ignored.
		c.Client.BaseURL = "http://localsocket"
	}
	c.Client.UnmarshalError = httprequest.ErrorUnmarshaler(new(params.Error))
	c.Client.Doer = &clientDoer{
		c: &c,
		httpClient: &http.Client{
			Transport: &http.Transport{
				Dial: func(_, _ string) (net.Conn, error) {
					return net.Dial(netw, addr)
				},
			},
		},
	}
	c.setAccessToken(accessToken)
	return &c
}

// Get implemets bakery.RootKeyStore.Get by getting the key from
// the macaroond server.
func (c *Client) Get(ctx context.Context, id []byte) ([]byte, error) {
	resp, err := c.FindRootKey(ctx, &params.FindRootKeyRequest{
		Id: string(id),
	})
	if err == nil {
		return resp.RootKey, nil
	}
	if errgo.Cause(err) == params.ErrNotFound {
		return nil, bakery.ErrNotFound
	}
	return nil, errgo.Mask(err)
}

// RootKey implements bakery.RootKeyStore.Get by using the
// macaroond server.
func (c *Client) RootKey(ctx context.Context) (rootKey, id []byte, err error) {
	resp, err := c.NewRootKey(ctx, &params.NewRootKeyRequest{})
	if err != nil {
		return nil, nil, errgo.Mask(err)
	}
	return resp.RootKey, resp.Id, nil
}

func (c *Client) setAccessToken(ms macaroon.Slice) {
	var tokenData string
	if len(ms) > 0 {
		tokenData0, err := json.Marshal(ms)
		if err != nil {
			panic(err)
		}
		tokenData = base64.RawURLEncoding.EncodeToString(tokenData0)
	}
	c.accessToken = tokenData
}

type clientDoer struct {
	c          *Client
	httpClient *http.Client
}

func (c *clientDoer) Do(req *http.Request) (*http.Response, error) {
	if c.c.accessToken != "" {
		req.Header.Set(httpbakery.MacaroonsHeader, c.c.accessToken)
	}
	return c.httpClient.Do(req)
}

func (c *Client) Login(ctx context.Context, password string) (*bakery.Macaroon, error) {
	resp, err := c.Access(ctx, &params.AccessRequest{
		Password: password,
	})
	if err != nil {
		return nil, errgo.Mask(err, errgo.Is(params.ErrInitialPasswordNeeded))
	}
	// TODO discharge if necessary
	c.setAccessToken(bakery.Slice{resp.Macaroon}.Bind())
	return resp.Macaroon, nil
}
