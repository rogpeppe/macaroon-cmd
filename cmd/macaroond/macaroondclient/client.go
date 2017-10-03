package client

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net"
	"net/http"

	"github.com/juju/httprequest"
	"github.com/rogpeppe/macaroon-cmd/internal/params"

	errgo "gopkg.in/errgo.v1"
	"gopkg.in/macaroon-bakery.v2-unstable/bakery"
	"gopkg.in/macaroon-bakery.v2-unstable/httpbakery"
	"gopkg.in/macaroon.v2-unstable"
)

//go:generate httprequest-generate-client github.com/rogpeppe/macaroon-cmd/cmd/macaroond handler client

type Client struct {
	client
}

// New returns a new client that uses the given token for
// access. If accessToken is nil, the only methods
// that may be called are Login and ChangePassword.
func New(netw, addr string, accessToken macaroon.Slice) *Client {
	var tokenData string
	if len(accessToken) > 0 {
		tokenData0, err := json.Marshal(accessToken)
		if err != nil {
			panic(err)
		}
		tokenData = base64.RawURLEncoding.EncodeToString(tokenData0)
	}
	var c Client
	if netw == "tcp" {
		c.Client.BaseURL = "http://" + addr
	} else {
		// For decent errors only - address is ignored.
		c.Client.BaseURL = "http://localsocket"
	}
	c.Client.UnmarshalError = httprequest.ErrorUnmarshaler(new(params.Error))
	c.Client.Doer = &clientDoer{
		httpClient: &http.Client{
			Transport: &http.Transport{
				Dial: func(_, _ string) (net.Conn, error) {
					return net.Dial(netw, addr)
				},
			},
		},
		accessToken: tokenData,
	}
	return &c
}

type clientDoer struct {
	c           *Client
	httpClient  *http.Client
	accessToken string
}

func (c *clientDoer) Do(req *http.Request) (*http.Response, error) {
	if c.accessToken != "" {
		req.Header.Set(httpbakery.MacaroonsHeader, c.accessToken)
	}
	return c.httpClient.Do(req)
}

func (c *Client) Login(ctx context.Context, password string) (*bakery.Macaroon, error) {
	resp, err := c.Access(ctx, &params.AccessRequest{
		Password: password,
	})
	if err != nil {
		return nil, errgo.Mask(err)
	}
	return resp.Macaroon, nil
}
