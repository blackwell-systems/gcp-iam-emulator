package server

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	iampb "google.golang.org/genproto/googleapis/iam/v1" //nolint:staticcheck // Using standard genproto package
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/blackwell-systems/gcp-iam-emulator/internal/storage"
)

type Server struct {
	iampb.UnimplementedIAMPolicyServer
	storage     *storage.Storage
	trace       bool
	explain     bool
	traceFile   *os.File
	traceLogger *slog.Logger
}

func NewServer() *Server {
	return &Server{
		storage: storage.NewStorage(),
		trace:   false,
		explain: false,
	}
}

func (s *Server) SetTrace(trace bool) {
	s.trace = trace
}

func (s *Server) SetExplain(explain bool) {
	s.explain = explain
}

func (s *Server) SetTraceOutput(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create trace output file: %w", err)
	}
	
	s.traceFile = f
	s.traceLogger = slog.New(slog.NewJSONHandler(f, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	
	return nil
}

func (s *Server) LoadPolicies(policies map[string]*iampb.Policy) { //nolint:staticcheck // Using standard genproto package
	s.storage.LoadPolicies(policies)
}

func (s *Server) LoadGroups(groups map[string][]string) {
	s.storage.LoadGroups(groups)
}

func (s *Server) GetStorage() *storage.Storage {
	return s.storage
}

func (s *Server) logTrace(resource, principal string, allowed []string, duration time.Duration) {
	if s.traceLogger != nil {
		s.traceLogger.Info("permission_check",
			"resource", resource,
			"principal", principal,
			"allowed_permissions", allowed,
			"duration_ms", duration.Milliseconds(),
			"timestamp", time.Now().Format(time.RFC3339),
		)
	}
}

func (s *Server) extractPrincipal(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}

	principals := md.Get("x-emulator-principal")
	if len(principals) == 0 {
		return ""
	}

	return principals[0]
}

func (s *Server) SetIamPolicy(ctx context.Context, req *iampb.SetIamPolicyRequest) (*iampb.Policy, error) { //nolint:staticcheck // Using standard genproto package
	if req.Resource == "" {
		return nil, status.Error(codes.InvalidArgument, "resource is required")
	}

	if req.Policy == nil {
		return nil, status.Error(codes.InvalidArgument, "policy is required")
	}

	policy, err := s.storage.SetIamPolicy(req.Resource, req.Policy)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return policy, nil
}

func (s *Server) GetIamPolicy(ctx context.Context, req *iampb.GetIamPolicyRequest) (*iampb.Policy, error) { //nolint:staticcheck // Using standard genproto package
	if req.Resource == "" {
		return nil, status.Error(codes.InvalidArgument, "resource is required")
	}

	policy, err := s.storage.GetIamPolicy(req.Resource)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return policy, nil
}

func (s *Server) TestIamPermissions(ctx context.Context, req *iampb.TestIamPermissionsRequest) (*iampb.TestIamPermissionsResponse, error) { //nolint:staticcheck // Using standard genproto package
	if req.Resource == "" {
		return nil, status.Error(codes.InvalidArgument, "resource is required")
	}

	if len(req.Permissions) == 0 {
		return nil, status.Error(codes.InvalidArgument, "permissions is required")
	}

	principal := s.extractPrincipal(ctx)

	start := time.Now()
	allowed, err := s.storage.TestIamPermissions(req.Resource, principal, req.Permissions, s.trace || s.explain)
	duration := time.Since(start)
	
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	s.logTrace(req.Resource, principal, allowed, duration)

	return &iampb.TestIamPermissionsResponse{ //nolint:staticcheck // Using standard genproto package
		Permissions: allowed,
	}, nil
}

type ProjectsServer struct {
	storage *storage.Storage
}

func NewProjectsServer(storage *storage.Storage) *ProjectsServer {
	return &ProjectsServer{storage: storage}
}

func (s *ProjectsServer) CreateProject(projectID string) error {
	_, err := s.storage.CreateProject(projectID)
	return err
}
