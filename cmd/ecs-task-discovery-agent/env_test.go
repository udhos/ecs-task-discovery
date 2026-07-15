package main

import "testing"

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
