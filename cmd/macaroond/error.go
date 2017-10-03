package main

import (
	"context"
	"net/http"

	"github.com/juju/httprequest"
	"gopkg.in/errgo.v1"
	"gopkg.in/macaroon-bakery.v2-unstable/httpbakery"

	"github.com/rogpeppe/macaroon-cmd/params"
)

type errorCoder interface {
	ErrorCode() params.ErrorCode
}

func errorToResponse(ctx context.Context, err error) (int, interface{}) {
	logger.Infof("HTTP error response: %#v", err)
	// Allow bakery errors to be returned as the bakery would
	// like them, so that httpbakery.Client.Do will work.
	if err, ok := errgo.Cause(err).(*httpbakery.Error); ok {
		return httpbakery.ErrorToResponse(ctx, err)
	}
	errorBody := errorResponseBody(err)
	status := http.StatusInternalServerError
	switch errorBody.Code {
	case params.ErrNotFound:
		status = http.StatusNotFound
	case params.ErrBadRequest:
		status = http.StatusBadRequest
	case params.ErrUnauthorized:
		status = http.StatusUnauthorized
	}
	return status, errorBody
}

// errorResponse returns an appropriate error response for the provided error.
func errorResponseBody(err error) *params.Error {
	errResp := &params.Error{
		Message: err.Error(),
	}
	cause := errgo.Cause(err)
	if coder, ok := cause.(errorCoder); ok {
		errResp.Code = coder.ErrorCode()
	} else if errgo.Cause(err) == httprequest.ErrUnmarshal {
		errResp.Code = params.ErrBadRequest
	}
	return errResp
}
