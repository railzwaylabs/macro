package server

import "context"

// Server is a lifecycle component managed by a Macro service.
// Concrete transports, such as gRPC or the debug HTTP server, implement it.
type Server interface {
	Start(context.Context) error
	Stop(context.Context) error
}
