package storage

import (
	"fmt"
	"strings"
	"time"

	expr "google.golang.org/genproto/googleapis/type/expr"
)

type EvalContext struct {
	ResourceName string
	ResourceType string
	RequestTime  time.Time
}

func evaluateCondition(condition *expr.Expr, ctx EvalContext) (bool, string) {
	if condition == nil {
		return true, "no condition"
	}

	expr := strings.TrimSpace(condition.Expression)
	if expr == "" {
		return true, "empty condition"
	}

	if strings.Contains(expr, "resource.name.startsWith") {
		return evalStartsWith(expr, ctx.ResourceName)
	}

	if strings.Contains(expr, "resource.type") {
		return evalResourceType(expr, ctx.ResourceType)
	}

	if strings.Contains(expr, "request.time") {
		return evalRequestTime(expr, ctx.RequestTime)
	}

	return false, fmt.Sprintf("unsupported CEL expression: %s", expr)
}

func evalStartsWith(expr, resourceName string) (bool, string) {
	start := strings.Index(expr, `"`)
	end := strings.LastIndex(expr, `"`)
	if start == -1 || end == -1 || start >= end {
		return false, "invalid startsWith syntax"
	}

	prefix := expr[start+1 : end]
	result := strings.HasPrefix(resourceName, prefix)
	
	if result {
		return true, fmt.Sprintf("resource.name '%s' starts with '%s'", resourceName, prefix)
	}
	return false, fmt.Sprintf("resource.name '%s' does not start with '%s'", resourceName, prefix)
}

func evalResourceType(expr, resourceType string) (bool, string) {
	start := strings.Index(expr, `"`)
	end := strings.LastIndex(expr, `"`)
	if start == -1 || end == -1 || start >= end {
		return false, "invalid resource.type syntax"
	}

	expectedType := expr[start+1 : end]
	result := resourceType == expectedType

	if result {
		return true, fmt.Sprintf("resource.type '%s' matches '%s'", resourceType, expectedType)
	}
	return false, fmt.Sprintf("resource.type '%s' does not match '%s'", resourceType, expectedType)
}

func evalRequestTime(exprStr string, requestTime time.Time) (bool, string) {
	start := strings.Index(exprStr, `timestamp("`)
	if start == -1 {
		return false, "invalid request.time syntax"
	}

	start += len(`timestamp("`)
	end := strings.Index(exprStr[start:], `"`)
	if end == -1 {
		return false, "invalid timestamp format"
	}

	timestampStr := exprStr[start : start+end]
	targetTime, err := time.Parse(time.RFC3339, timestampStr)
	if err != nil {
		return false, fmt.Sprintf("invalid timestamp: %s", timestampStr)
	}

	isLessThan := strings.Contains(exprStr, "<")
	isGreaterThan := strings.Contains(exprStr, ">")

	if isLessThan {
		result := requestTime.Before(targetTime)
		if result {
			return true, fmt.Sprintf("request.time %s < %s", requestTime.Format(time.RFC3339), timestampStr)
		}
		return false, fmt.Sprintf("request.time %s >= %s", requestTime.Format(time.RFC3339), timestampStr)
	}

	if isGreaterThan {
		result := requestTime.After(targetTime)
		if result {
			return true, fmt.Sprintf("request.time %s > %s", requestTime.Format(time.RFC3339), timestampStr)
		}
		return false, fmt.Sprintf("request.time %s <= %s", requestTime.Format(time.RFC3339), timestampStr)
	}

	return false, "request.time expression must use < or >"
}

func extractResourceType(resourceName string) string {
	if strings.Contains(resourceName, "/secrets/") {
		return "SECRET"
	}
	if strings.Contains(resourceName, "/cryptoKeys/") {
		return "CRYPTO_KEY"
	}
	if strings.Contains(resourceName, "/keyRings/") {
		return "KEY_RING"
	}
	return "UNKNOWN"
}
