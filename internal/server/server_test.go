package server

import (
	"context"
	"testing"

	iampb "google.golang.org/genproto/googleapis/iam/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestSetIamPolicy(t *testing.T) {
	s := NewServer()
	ctx := context.Background()

	policy := &iampb.Policy{
		Version: 1,
		Bindings: []*iampb.Binding{
			{
				Role:    "roles/owner",
				Members: []string{"user:alice@example.com"},
			},
		},
	}

	req := &iampb.SetIamPolicyRequest{
		Resource: "projects/test/secrets/secret1",
		Policy:   policy,
	}

	resp, err := s.SetIamPolicy(ctx, req)
	if err != nil {
		t.Fatalf("SetIamPolicy failed: %v", err)
	}

	if len(resp.Bindings) != 1 {
		t.Errorf("Expected 1 binding, got %d", len(resp.Bindings))
	}
}

func TestSetIamPolicy_MissingResource(t *testing.T) {
	s := NewServer()
	ctx := context.Background()

	req := &iampb.SetIamPolicyRequest{
		Resource: "",
		Policy: &iampb.Policy{
			Bindings: []*iampb.Binding{
				{Role: "roles/owner", Members: []string{"user:test@example.com"}},
			},
		},
	}

	_, err := s.SetIamPolicy(ctx, req)
	if err == nil {
		t.Fatal("Expected error for missing resource")
	}

	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Errorf("Expected InvalidArgument, got %v", err)
	}
}

func TestSetIamPolicy_MissingPolicy(t *testing.T) {
	s := NewServer()
	ctx := context.Background()

	req := &iampb.SetIamPolicyRequest{
		Resource: "projects/test/secrets/secret1",
		Policy:   nil,
	}

	_, err := s.SetIamPolicy(ctx, req)
	if err == nil {
		t.Fatal("Expected error for missing policy")
	}

	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Errorf("Expected InvalidArgument, got %v", err)
	}
}

func TestGetIamPolicy(t *testing.T) {
	s := NewServer()
	ctx := context.Background()

	policy := &iampb.Policy{
		Version: 1,
		Bindings: []*iampb.Binding{
			{
				Role:    "roles/viewer",
				Members: []string{"user:bob@example.com"},
			},
		},
	}

	setReq := &iampb.SetIamPolicyRequest{
		Resource: "projects/test/secrets/secret1",
		Policy:   policy,
	}
	s.SetIamPolicy(ctx, setReq)

	getReq := &iampb.GetIamPolicyRequest{
		Resource: "projects/test/secrets/secret1",
	}

	resp, err := s.GetIamPolicy(ctx, getReq)
	if err != nil {
		t.Fatalf("GetIamPolicy failed: %v", err)
	}

	if len(resp.Bindings) != 1 {
		t.Errorf("Expected 1 binding, got %d", len(resp.Bindings))
	}

	if resp.Bindings[0].Role != "roles/viewer" {
		t.Errorf("Expected role 'roles/viewer', got '%s'", resp.Bindings[0].Role)
	}
}

func TestGetIamPolicy_MissingResource(t *testing.T) {
	s := NewServer()
	ctx := context.Background()

	req := &iampb.GetIamPolicyRequest{
		Resource: "",
	}

	_, err := s.GetIamPolicy(ctx, req)
	if err == nil {
		t.Fatal("Expected error for missing resource")
	}

	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Errorf("Expected InvalidArgument, got %v", err)
	}
}

func TestTestIamPermissions(t *testing.T) {
	s := NewServer()
	ctx := context.Background()

	policy := &iampb.Policy{
		Version: 1,
		Bindings: []*iampb.Binding{
			{
				Role:    "roles/secretmanager.secretAccessor",
				Members: []string{"serviceAccount:ci@test.iam.gserviceaccount.com"},
			},
		},
	}

	setReq := &iampb.SetIamPolicyRequest{
		Resource: "projects/test/secrets/secret1",
		Policy:   policy,
	}
	s.SetIamPolicy(ctx, setReq)

	testReq := &iampb.TestIamPermissionsRequest{
		Resource: "projects/test/secrets/secret1",
		Permissions: []string{
			"secretmanager.versions.access",
			"secretmanager.secrets.delete",
		},
	}

	resp, err := s.TestIamPermissions(ctx, testReq)
	if err != nil {
		t.Fatalf("TestIamPermissions failed: %v", err)
	}

	if len(resp.Permissions) != 1 {
		t.Errorf("Expected 1 allowed permission, got %d", len(resp.Permissions))
	}

	if resp.Permissions[0] != "secretmanager.versions.access" {
		t.Errorf("Expected 'secretmanager.versions.access', got '%s'", resp.Permissions[0])
	}
}

func TestTestIamPermissions_MissingResource(t *testing.T) {
	s := NewServer()
	ctx := context.Background()

	req := &iampb.TestIamPermissionsRequest{
		Resource:    "",
		Permissions: []string{"secretmanager.versions.access"},
	}

	_, err := s.TestIamPermissions(ctx, req)
	if err == nil {
		t.Fatal("Expected error for missing resource")
	}

	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Errorf("Expected InvalidArgument, got %v", err)
	}
}

func TestTestIamPermissions_MissingPermissions(t *testing.T) {
	s := NewServer()
	ctx := context.Background()

	req := &iampb.TestIamPermissionsRequest{
		Resource:    "projects/test/secrets/secret1",
		Permissions: []string{},
	}

	_, err := s.TestIamPermissions(ctx, req)
	if err == nil {
		t.Fatal("Expected error for missing permissions")
	}

	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Errorf("Expected InvalidArgument, got %v", err)
	}
}
