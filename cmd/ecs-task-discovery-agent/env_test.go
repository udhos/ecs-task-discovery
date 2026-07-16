package main

import (
	"testing"
	"time"

	"github.com/udhos/ecs-task-discovery/discovery"
)

func TestEnvStringUsesValue(t *testing.T) {
	t.Setenv("TEST_STRING_VALUE", "abc")

	got := envString("TEST_STRING_VALUE", "fallback")

	if got != "abc" {
		t.Fatalf("envString()=%q want=%q", got, "abc")
	}
}

func TestEnvStringUsesDefaultWhenEmpty(t *testing.T) {
	t.Setenv("TEST_STRING_DEFAULT", "")

	got := envString("TEST_STRING_DEFAULT", "fallback")

	if got != "fallback" {
		t.Fatalf("envString()=%q want=%q", got, "fallback")
	}
}

func TestEnvDurationParsesValue(t *testing.T) {
	t.Setenv("TEST_DURATION_VALUE", "2m30s")

	got := envDuration("TEST_DURATION_VALUE", 5*time.Second)

	const want = 150 * time.Second
	if got != want {
		t.Fatalf("envDuration()=%v want=%v", got, want)
	}
}

func TestEnvDurationUsesDefaultOnInvalid(t *testing.T) {
	t.Setenv("TEST_DURATION_INVALID", "not-a-duration")

	const want = 7 * time.Second
	got := envDuration("TEST_DURATION_INVALID", want)

	if got != want {
		t.Fatalf("envDuration()=%v want default=%v", got, want)
	}
}

func TestEnvBoolParsesValue(t *testing.T) {
	t.Setenv("TEST_BOOL_VALUE", "true")

	got := envBool("TEST_BOOL_VALUE", false)

	if !got {
		t.Fatal("envBool()=false want=true")
	}
}

func TestEnvBoolUsesDefaultOnInvalid(t *testing.T) {
	t.Setenv("TEST_BOOL_INVALID", "not-a-bool")

	got := envBool("TEST_BOOL_INVALID", true)

	if !got {
		t.Fatal("envBool()=false want default=true")
	}
}

func TestEnvInt64ParsesValue(t *testing.T) {
	t.Setenv("TEST_INT64_VALUE", "123456")

	got := envInt64("TEST_INT64_VALUE", 7)

	const want int64 = 123456
	if got != want {
		t.Fatalf("envInt64()=%d want=%d", got, want)
	}
}

func TestEnvInt64UsesDefaultOnInvalidValue(t *testing.T) {
	t.Setenv("TEST_INT64_INVALID", "bad-number")

	const want int64 = 42
	got := envInt64("TEST_INT64_INVALID", want)

	if got != want {
		t.Fatalf("envInt64()=%d want default=%d", got, want)
	}
}

func TestEnvHealthCheckModeValidValue(t *testing.T) {
	t.Setenv("TEST_HEALTH_MODE_VALID", "TRUE")

	got := envHealthCheckMode("TEST_HEALTH_MODE_VALID", discovery.HealthCheckModeDetect)

	if got != discovery.HealthCheckModeTrue {
		t.Fatalf("envHealthCheckMode()=%q want=%q", got, discovery.HealthCheckModeTrue)
	}
}

func TestEnvHealthCheckModeUsesDefaultOnInvalidValue(t *testing.T) {
	t.Setenv("TEST_HEALTH_MODE_INVALID", "invalid")

	got := envHealthCheckMode("TEST_HEALTH_MODE_INVALID", discovery.HealthCheckModeDetectAndHandleErrorAsFalse)

	if got != discovery.HealthCheckModeDetectAndHandleErrorAsFalse {
		t.Fatalf("envHealthCheckMode()=%q want default=%q", got, discovery.HealthCheckModeDetectAndHandleErrorAsFalse)
	}
}
