package storage

import (
	"fmt"
	"testing"
	"time"

	expr "google.golang.org/genproto/googleapis/type/expr"
)

func TestEvaluateCondition_StartsWith(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		resource   string
		expected   bool
	}{
		{
			name:       "matches prefix",
			expression: `resource.name.startsWith("projects/prod/")`,
			resource:   "projects/prod/secrets/api-key",
			expected:   true,
		},
		{
			name:       "does not match prefix",
			expression: `resource.name.startsWith("projects/prod/")`,
			resource:   "projects/staging/secrets/api-key",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			condition := &expr.Expr{
				Expression: tt.expression,
			}

			ctx := EvalContext{
				ResourceName: tt.resource,
				ResourceType: "SECRET",
				RequestTime:  time.Now(),
			}

			result, _ := evaluateCondition(condition, ctx)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for expression %s on resource %s", tt.expected, result, tt.expression, tt.resource)
			}
		})
	}
}

func TestEvaluateCondition_ResourceType(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		resource   string
		expected   bool
	}{
		{
			name:       "matches SECRET",
			expression: `resource.type == "SECRET"`,
			resource:   "projects/test/secrets/api-key",
			expected:   true,
		},
		{
			name:       "matches CRYPTO_KEY",
			expression: `resource.type == "CRYPTO_KEY"`,
			resource:   "projects/test/locations/global/keyRings/ring/cryptoKeys/key",
			expected:   true,
		},
		{
			name:       "does not match",
			expression: `resource.type == "SECRET"`,
			resource:   "projects/test/locations/global/keyRings/ring",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			condition := &expr.Expr{
				Expression: tt.expression,
			}

			ctx := EvalContext{
				ResourceName: tt.resource,
				ResourceType: extractResourceType(tt.resource),
				RequestTime:  time.Now(),
			}

			result, _ := evaluateCondition(condition, ctx)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for expression %s on resource %s (type: %s)", tt.expected, result, tt.expression, tt.resource, ctx.ResourceType)
			}
		})
	}
}

func TestEvaluateCondition_RequestTime(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	future := "2026-12-31T00:00:00Z"
	past := "2026-01-01T00:00:00Z"

	tests := []struct {
		name       string
		expression string
		requestTime time.Time
		expected   bool
	}{
		{
			name:       "time before future",
			expression: fmt.Sprintf(`request.time < timestamp("%s")`, future),
			requestTime: now,
			expected:   true,
		},
		{
			name:       "time after past",
			expression: fmt.Sprintf(`request.time > timestamp("%s")`, past),
			requestTime: now,
			expected:   true,
		},
		{
			name:       "time after future (should fail)",
			expression: fmt.Sprintf(`request.time < timestamp("%s")`, past),
			requestTime: now,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			condition := &expr.Expr{
				Expression: tt.expression,
			}

			ctx := EvalContext{
				ResourceName: "projects/test/secrets/api-key",
				ResourceType: "SECRET",
				RequestTime:  tt.requestTime,
			}

			result, _ := evaluateCondition(condition, ctx)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for expression %s at time %s", tt.expected, result, tt.expression, tt.requestTime.Format(time.RFC3339))
			}
		})
	}
}

func TestExtractResourceType(t *testing.T) {
	tests := []struct {
		resource string
		expected string
	}{
		{"projects/test/secrets/api-key", "SECRET"},
		{"projects/test/locations/global/keyRings/ring/cryptoKeys/key", "CRYPTO_KEY"},
		{"projects/test/locations/global/keyRings/ring", "KEY_RING"},
		{"projects/test", "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.resource, func(t *testing.T) {
			result := extractResourceType(tt.resource)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s for resource %s", tt.expected, result, tt.resource)
			}
		})
	}
}
