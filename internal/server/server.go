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

	"github.com/blackwell-systems/gcp-emulator-auth/pkg/trace"
	"github.com/blackwell-systems/gcp-iam-emulator/internal/storage"
)

type Server struct {
	iampb.UnimplementedIAMPolicyServer
	storage     *storage.Storage
	trace       bool
	explain     bool
	traceFile   *os.File
	traceLogger *slog.Logger
	traceWriter *trace.Writer
}

func NewServer() *Server {
	// Initialize trace writer from environment
	traceWriter, _ := trace.NewWriterFromEnv()
	
	return &Server{
		storage:     storage.NewStorage(),
		trace:       false,
		explain:     false,
		traceWriter: traceWriter,
	}
}

func (s *Server) SetTrace(trace bool) {
	s.trace = trace
}

func (s *Server) SetExplain(explain bool) {
	s.explain = explain
}

func (s *Server) SetAllowUnknownRoles(allow bool) {
	s.storage.SetAllowUnknownRoles(allow)
}

func (s *Server) SetTraceOutput(path string) error {
	// Create legacy slog trace file
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create trace output file: %w", err)
	}
	
	s.traceFile = f
	s.traceLogger = slog.New(slog.NewJSONHandler(f, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	
	// Also create structured trace writer if not already set from env
	if s.traceWriter == nil {
		w, err := trace.NewWriter(path)
		if err != nil {
			return fmt.Errorf("failed to create trace writer: %w", err)
		}
		s.traceWriter = w
	}
	
	return nil
}

func (s *Server) LoadPolicies(policies map[string]*iampb.Policy) { //nolint:staticcheck // Using standard genproto package
	s.storage.LoadPolicies(policies)
}

func (s *Server) LoadGroups(groups map[string][]string) {
	s.storage.LoadGroups(groups)
}

func (s *Server) LoadCustomRoles(roles map[string][]string) {
	s.storage.LoadCustomRoles(roles)
}

func (s *Server) GetStorage() *storage.Storage {
	return s.storage
}

func (s *Server) logTrace(resource, principal string, allowed []string, duration time.Duration) {
	// Legacy slog trace
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

func (s *Server) emitTraceEvents(resource, principal string, permissions []string, allowed []string, duration time.Duration) {
	if s.traceWriter == nil {
		return
	}
	
	// Create a map of allowed permissions for quick lookup
	allowedMap := make(map[string]bool, len(allowed))
	for _, perm := range allowed {
		allowedMap[perm] = true
	}
	
	// Emit one event per permission check
	for _, perm := range permissions {
		outcome := trace.OutcomeDeny
		reason := "no_matching_binding"
		
		if allowedMap[perm] {
			outcome = trace.OutcomeAllow
			reason = "binding_match"
		}
		
		event := trace.AuthzEvent{
			SchemaVersion: trace.SchemaV1_0,
			EventType:     trace.EventTypeAuthzCheck,
			Timestamp:     trace.NowRFC3339Nano(),
			Actor: &trace.Actor{
				Principal: principal,
			},
			Target: &trace.Target{
				Resource: resource,
			},
			Action: &trace.Action{
				Permission: perm,
				Method:     "TestIamPermissions",
			},
			Decision: &trace.Decision{
				Outcome:     outcome,
				Reason:      reason,
				EvaluatedBy: "gcp-iam-emulator",
				LatencyMS:   duration.Milliseconds(),
			},
		}
		
		// Emit event (gracefully ignores if writer is nil)
		_ = s.traceWriter.Emit(event)
	}
	
	// Flush after emitting all events
	_ = s.traceWriter.Flush()
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

	// Legacy slog trace
	s.logTrace(req.Resource, principal, allowed, duration)
	
	// Structured trace events (JSONL)
	s.emitTraceEvents(req.Resource, principal, req.Permissions, allowed, duration)

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
