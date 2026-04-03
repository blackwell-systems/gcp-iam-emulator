package iam

import (
	"net/http"

	"github.com/blackwell-systems/gcp-iam-emulator/internal/gateway"
)

// NewGatewayHandler returns an http.Handler that serves the IAM REST API
// by proxying to the gRPC server at grpcAddr via grpc-gateway v2.
func NewGatewayHandler(grpcAddr string) (http.Handler, error) {
	srv, err := gateway.NewServer(grpcAddr)
	if err != nil {
		return nil, err
	}
	return srv.Handler(), nil
}
