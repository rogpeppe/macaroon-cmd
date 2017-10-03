package params

import (
	"github.com/juju/httprequest"
	"gopkg.in/macaroon-bakery.v2-unstable/bakery"
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
	httprequest.Route `httprequest:"GET /macaroon"`
}

type AccessResponse struct {
	Macaroon *bakery.Macaroon `json:"macaroon"`
}

type SetPasswordRequest struct {
	httprequest.Route `httprequest:"PUT /password"`
	OldPassword string	`json:"oldPassword,form"`
	NewPassword string	`json:"newPassword,form"`
}

