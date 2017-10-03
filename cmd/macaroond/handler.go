package main

import (
	"bytes"
	"context"
	"sync"
	"time"

	"github.com/juju/httprequest"
	errgo "gopkg.in/errgo.v1"
	"gopkg.in/macaroon-bakery.v2-unstable/bakery"
	"gopkg.in/macaroon-bakery.v2-unstable/httpbakery"

	"github.com/rogpeppe/macaroon-cmd/params"
)

var rootKeyId = []byte("0")

const expiryDuration = 24 * time.Hour

type handler struct {
	srv     *server
	mu      sync.Mutex
	rootKey []byte
}

var accessOp = bakery.Op{
	Entity: "global",
	Action: "access",
}

func (srv *server) newHandler(p httprequest.Params, req interface{}) (*handler, context.Context, error) {
	switch req.(type) {
	case *params.AccessRequest,
		*params.SetPasswordRequest:
		// Both these requests check the password held in the request.
	default:
		// All other requests require the access token.
		_, err := srv.bakery.Checker.Auth(httpbakery.RequestMacaroons(p.Request)...).Allow(p.Context, accessOp)
		if err != nil {
			return nil, nil, errgo.Mask(err)
		}
	}
	return &handler{
		srv: srv,
	}, p.Context, nil
}

func (h *handler) SetPassword(req *params.SetPasswordRequest) error {
	if err := h.srv.setPassword(req.OldPassword, req.NewPassword); err != nil {
		return errgo.Mask(err)
	}
	return nil
}

func (h *handler) NewRootKey(p httprequest.Params, req *params.NewRootKeyRequest) (*params.NewRootKeyResponse, error) {
	masterKey, err := h.srv.getMasterKey()
	if err != nil {
		return nil, errgo.Mask(err)
	}
	// TODO use master key to decrypt root key stored elsewhere.
	return &params.NewRootKeyResponse{
		Id:      rootKeyId,
		RootKey: masterKey,
	}, nil
}

func (h *handler) FindRootKey(p httprequest.Params, req *params.FindRootKeyRequest) (*params.FindRootKeyResponse, error) {
	if !bytes.Equal([]byte(req.Id), rootKeyId) {
		return nil, params.ErrNotFound
	}
	masterKey, err := h.srv.getMasterKey()
	if err != nil {
		return nil, errgo.Mask(err)
	}
	return &params.FindRootKeyResponse{
		RootKey: masterKey,
	}, nil
}

func (h *handler) Access(p httprequest.Params, req *params.AccessRequest) (*params.AccessResponse, error) {
	if h.srv.needsPassword() {
		return nil, errgo.WithCausef(nil, params.ErrInitialPasswordNeeded, "")
	}
	if err := h.srv.checkPassword(req.Password); err != nil {
		return nil, errgo.WithCausef(err, params.ErrUnauthorized, "")
	}
	m, err := h.srv.bakery.Oven.NewMacaroon(p.Context, httpbakery.RequestVersion(p.Request), time.Now().Add(expiryDuration), nil, accessOp)
	if err != nil {
		return nil, errgo.Notef(err, "cannot make macaroon")
	}
	return &params.AccessResponse{
		Macaroon: m,
	}, nil
}
