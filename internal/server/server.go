package server

import (
	"context"
	"strings"

	iampb "google.golang.org/genproto/googleapis/iam/v1" //nolint:staticcheck // Using standard genproto package
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/blackwell-systems/gcp-iam-emulator/internal/storage"
)

type Server struct {
	iampb.UnimplementedIAMPolicyServer
	storage *storage.Storage
	trace   bool
}

func NewServer() *Server {
	return &Server{
		storage: storage.NewStorage(),
		trace:   false,
	}
}

func (s *Server) SetTrace(trace bool) {
	s.trace = trace
}

func (s *Server) LoadPolicies(policies map[string]*iampb.Policy) { //nolint:staticcheck // Using standard genproto package
	s.storage.LoadPolicies(policies)
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

	allowed, err := s.storage.TestIamPermissions(req.Resource, principal, req.Permissions, s.trace)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

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
