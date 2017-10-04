package params

import (
	"github.com/juju/httprequest"
	"gopkg.in/macaroon-bakery.v2-unstable/bakery"
)

const (
	DefaultNetwork = "tcp"
	DefaultAddress = "localhost:46753"
)

type FindRootKeyRequest struct {
	httprequest.Route `httprequest:"GET /key/:Id"`
	Id                string `httprequest:",path"`
}

type FindRootKeyResponse struct {
	RootKey []byte `json:"rootKey"`
}

type NewRootKeyRequest struct {
	httprequest.Route `httprequest:"POST /key"`
}

type NewRootKeyResponse struct {
	Id      []byte `json:"id"`
	RootKey []byte `json:"rootKey"`
}

type AccessRequest struct {
	httprequest.Route `httprequest:"POST /macaroon"`
	Password          string `httprequest:"password,form"`
}

type AccessResponse struct {
	Macaroon *bakery.Macaroon `json:"macaroon"`
}

type SetPasswordRequest struct {
	httprequest.Route `httprequest:"PUT /password"`
	OldPassword       string `httprequest:"oldPassword,form"`
	NewPassword       string `httprequest:"newPassword,form"`
}
