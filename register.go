// Package iam provides the composition entry point for the GCP IAM Emulator.
//
// Register wires the IAM Policy gRPC service onto an existing grpc.Server,
// enabling use within the unified gcp-emulator or any custom composition layer.
// For standalone use, see cmd/server.
package iam

import (
	"fmt"

	iampb "google.golang.org/genproto/googleapis/iam/v1" //nolint:staticcheck
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/blackwell-systems/gcp-iam-emulator/internal/config"
	"github.com/blackwell-systems/gcp-iam-emulator/internal/server"
)

// Option configures the IAM server at registration time.
type Option func(*options)

type options struct {
	policyFile        string
	trace             bool
	allowUnknownRoles bool
}

// WithPolicyFile loads IAM policies, groups, and custom roles from a YAML file.
func WithPolicyFile(path string) Option {
	return func(o *options) { o.policyFile = path }
}

// WithTrace enables authorization decision logging.
func WithTrace(enabled bool) Option {
	return func(o *options) { o.trace = enabled }
}

// WithAllowUnknownRoles enables wildcard role matching (compatibility mode).
func WithAllowUnknownRoles(allow bool) Option {
	return func(o *options) { o.allowUnknownRoles = allow }
}

// Register adds the IAM Policy gRPC service to grpcSrv.
// It does not start a listener — the caller owns the grpc.Server lifecycle.
func Register(grpcSrv *grpc.Server, opts ...Option) error {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}

	srv := server.NewServer()
	srv.SetTrace(o.trace)
	srv.SetAllowUnknownRoles(o.allowUnknownRoles)

	if o.policyFile != "" {
		cfg, err := config.LoadFromFile(o.policyFile)
		if err != nil {
			return fmt.Errorf("iam: failed to load policy file %q: %w", o.policyFile, err)
		}
		srv.LoadPolicies(cfg.ToPolicies())
		if len(cfg.Groups) > 0 {
			groups := make(map[string][]string, len(cfg.Groups))
			for name, g := range cfg.Groups {
				groups[name] = g.Members
			}
			srv.LoadGroups(groups)
		}
		if len(cfg.Roles) > 0 {
			roles := make(map[string][]string, len(cfg.Roles))
			for name, r := range cfg.Roles {
				roles[name] = r.Permissions
			}
			srv.LoadCustomRoles(roles)
		}
	}

	iampb.RegisterIAMPolicyServer(grpcSrv, srv) //nolint:staticcheck
	reflection.Register(grpcSrv)
	return nil
}
