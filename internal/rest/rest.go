package rest

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	iampb "google.golang.org/genproto/googleapis/iam/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/blackwell-systems/gcp-iam-emulator/internal/storage"
)

type Server struct {
	storage *storage.Storage
	trace   bool
}

func NewServer(store *storage.Storage, trace bool) *Server {
	return &Server{
		storage: store,
		trace:   trace,
	}
}

func (s *Server) RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/v1/", s.handleRequest)
}

func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/v1/"), ":")
	if len(parts) < 2 {
		s.writeError(w, status.Error(codes.InvalidArgument, "invalid path format"))
		return
	}

	resource := parts[0]
	method := parts[1]

	switch method {
	case "setIamPolicy":
		s.handleSetIamPolicy(w, r, resource)
	case "getIamPolicy":
		s.handleGetIamPolicy(w, r, resource)
	case "testIamPermissions":
		s.handleTestIamPermissions(w, r, resource)
	default:
		s.writeError(w, status.Errorf(codes.Unimplemented, "unknown method: %s", method))
	}
}

func (s *Server) handleSetIamPolicy(w http.ResponseWriter, r *http.Request, resource string) {
	if r.Method != http.MethodPost {
		s.writeError(w, status.Error(codes.InvalidArgument, "method must be POST"))
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.writeError(w, status.Error(codes.InvalidArgument, "failed to read request body"))
		return
	}

	var req struct {
		Policy *iampb.Policy `json:"policy"`
	}

	if err := json.Unmarshal(body, &req); err != nil {
		s.writeError(w, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid JSON: %v", err)))
		return
	}

	if req.Policy == nil {
		s.writeError(w, status.Error(codes.InvalidArgument, "policy is required"))
		return
	}

	policy, err := s.storage.SetIamPolicy(resource, req.Policy)
	if err != nil {
		s.writeError(w, status.Error(codes.Internal, err.Error()))
		return
	}

	s.writeJSON(w, policy)
}

func (s *Server) handleGetIamPolicy(w http.ResponseWriter, r *http.Request, resource string) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		s.writeError(w, status.Error(codes.InvalidArgument, "method must be POST or GET"))
		return
	}

	policy, err := s.storage.GetIamPolicy(resource)
	if err != nil {
		s.writeError(w, status.Error(codes.NotFound, err.Error()))
		return
	}

	s.writeJSON(w, policy)
}

func (s *Server) handleTestIamPermissions(w http.ResponseWriter, r *http.Request, resource string) {
	if r.Method != http.MethodPost {
		s.writeError(w, status.Error(codes.InvalidArgument, "method must be POST"))
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.writeError(w, status.Error(codes.InvalidArgument, "failed to read request body"))
		return
	}

	var req struct {
		Permissions []string `json:"permissions"`
	}

	if err := json.Unmarshal(body, &req); err != nil {
		s.writeError(w, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid JSON: %v", err)))
		return
	}

	principal := r.Header.Get("X-Emulator-Principal")
	if principal == "" {
		principal = "user:anonymous"
	}

	allowed, err := s.storage.TestIamPermissions(resource, principal, req.Permissions, s.trace)
	if err != nil {
		s.writeError(w, status.Error(codes.Internal, err.Error()))
		return
	}

	response := map[string][]string{
		"permissions": allowed,
	}

	s.writeJSON(w, response)
}

func (s *Server) writeJSON(w http.ResponseWriter, data interface{}) {
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
	}
}

func (s *Server) writeError(w http.ResponseWriter, err error) {
	st := status.Convert(err)
	
	httpCode := grpcCodeToHTTP(st.Code())
	
	errResponse := map[string]interface{}{
		"error": map[string]interface{}{
			"code":    int(st.Code()),
			"message": st.Message(),
			"status":  st.Code().String(),
		},
	}

	w.WriteHeader(httpCode)
	if err := json.NewEncoder(w).Encode(errResponse); err != nil {
		log.Printf("Failed to encode error response: %v", err)
	}
}

func grpcCodeToHTTP(code codes.Code) int {
	switch code {
	case codes.OK:
		return http.StatusOK
	case codes.Canceled:
		return 499
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.FailedPrecondition:
		return http.StatusBadRequest
	case codes.Aborted:
		return http.StatusConflict
	case codes.OutOfRange:
		return http.StatusBadRequest
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Internal:
		return http.StatusInternalServerError
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DataLoss:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}
