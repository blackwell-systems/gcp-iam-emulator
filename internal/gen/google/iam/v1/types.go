// Package iamv1 re-exports IAM proto types for use by generated grpc-gateway code.
package iamv1

import (
	iampb "cloud.google.com/go/iam/apiv1/iampb"
	"google.golang.org/grpc"
)

// Request/response types referenced by the generated gateway code.
type SetIamPolicyRequest = iampb.SetIamPolicyRequest
type GetIamPolicyRequest = iampb.GetIamPolicyRequest
type TestIamPermissionsRequest = iampb.TestIamPermissionsRequest
type TestIamPermissionsResponse = iampb.TestIamPermissionsResponse
type Policy = iampb.Policy

// Service interfaces and client constructor referenced by the generated gateway code.
type IAMPolicyClient = iampb.IAMPolicyClient
type IAMPolicyServer = iampb.IAMPolicyServer

func NewIAMPolicyClient(cc grpc.ClientConnInterface) IAMPolicyClient {
	return iampb.NewIAMPolicyClient(cc)
}
