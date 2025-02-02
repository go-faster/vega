// Code generated by ogen, DO NOT EDIT.

package oas

import (
	"context"
)

// Handler handles operations described by OpenAPI v3 specification.
type Handler interface {
	// GetApplication implements getApplication operation.
	//
	// Get application.
	//
	// GET /applications/{name}
	GetApplication(ctx context.Context, params GetApplicationParams) (*ApplicationSummary, error)
	// GetApplications implements getApplications operation.
	//
	// Get application list.
	//
	// GET /applications
	GetApplications(ctx context.Context) (ApplicationList, error)
	// GetHealth implements getHealth operation.
	//
	// Get health.
	//
	// GET /health
	GetHealth(ctx context.Context) (*Health, error)
	// NewError creates *ErrorStatusCode from error returned by handler.
	//
	// Used for common default response.
	NewError(ctx context.Context, err error) *ErrorStatusCode
}

// Server implements http server based on OpenAPI v3 specification and
// calls Handler to handle requests.
type Server struct {
	h Handler
	baseServer
}

// NewServer creates new Server.
func NewServer(h Handler, opts ...ServerOption) (*Server, error) {
	s, err := newServerConfig(opts...).baseServer()
	if err != nil {
		return nil, err
	}
	return &Server{
		h:          h,
		baseServer: s,
	}, nil
}
