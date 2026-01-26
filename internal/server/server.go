package server

import (
	"context"
	"strings"

	iampb "google.golang.org/genproto/googleapis/iam/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/blackwell-systems/gcp-iam-emulator/internal/storage"
)

type Server struct {
	iampb.UnimplementedIAMPolicyServer
	storage *storage.Storage
}

func NewServer() *Server {
	return &Server{
		storage: storage.NewStorage(),
	}
}

func (s *Server) SetIamPolicy(ctx context.Context, req *iampb.SetIamPolicyRequest) (*iampb.Policy, error) {
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

func (s *Server) GetIamPolicy(ctx context.Context, req *iampb.GetIamPolicyRequest) (*iampb.Policy, error) {
	if req.Resource == "" {
		return nil, status.Error(codes.InvalidArgument, "resource is required")
	}

	policy, err := s.storage.GetIamPolicy(req.Resource)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return policy, nil
}

func (s *Server) TestIamPermissions(ctx context.Context, req *iampb.TestIamPermissionsRequest) (*iampb.TestIamPermissionsResponse, error) {
	if req.Resource == "" {
		return nil, status.Error(codes.InvalidArgument, "resource is required")
	}

	if len(req.Permissions) == 0 {
		return nil, status.Error(codes.InvalidArgument, "permissions is required")
	}

	allowed, err := s.storage.TestIamPermissions(req.Resource, req.Permissions)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &iampb.TestIamPermissionsResponse{
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
