package testutil

import (
	"errors"
	"testing"
)

// AssertNoError fails the test if err is not nil
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// AssertError fails the test if err is nil
func AssertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error but got nil")
	}
}

// AssertErrorIs fails the test if err is not the target error
func AssertErrorIs(t *testing.T, err, target error) {
	t.Helper()
	if !errors.Is(err, target) {
		t.Fatalf("expected error %v, got %v", target, err)
	}
}

// AssertErrorContains fails the test if err doesn't contain the substring
func AssertErrorContains(t *testing.T, err error, substring string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if !containsString(err.Error(), substring) {
		t.Fatalf("error %q does not contain %q", err.Error(), substring)
	}
}

// AssertEqual fails the test if got != want
func AssertEqual[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// AssertNotEqual fails the test if got == want
func AssertNotEqual[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got == want {
		t.Fatalf("expected values to be different, but both are %v", got)
	}
}

// AssertNil fails the test if value is not nil
func AssertNil(t *testing.T, value interface{}) {
	t.Helper()
	if value != nil {
		t.Fatalf("expected nil, got %v", value)
	}
}

// AssertNotNil fails the test if value is nil
func AssertNotNil(t *testing.T, value interface{}) {
	t.Helper()
	if value == nil {
		t.Fatal("expected non-nil value")
	}
}

// AssertTrue fails the test if condition is false
func AssertTrue(t *testing.T, condition bool, message string) {
	t.Helper()
	if !condition {
		t.Fatalf("assertion failed: %s", message)
	}
}

// AssertFalse fails the test if condition is true
func AssertFalse(t *testing.T, condition bool, message string) {
	t.Helper()
	if condition {
		t.Fatalf("assertion failed: %s", message)
	}
}

// AssertLen fails the test if the length doesn't match
func AssertLen[T any](t *testing.T, slice []T, expectedLen int) {
	t.Helper()
	if len(slice) != expectedLen {
		t.Fatalf("expected length %d, got %d", expectedLen, len(slice))
	}
}

// AssertContains fails the test if slice doesn't contain item
func AssertContains[T comparable](t *testing.T, slice []T, item T) {
	t.Helper()
	for _, v := range slice {
		if v == item {
			return
		}
	}
	t.Fatalf("slice does not contain %v", item)
}

// Helper function to check substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
