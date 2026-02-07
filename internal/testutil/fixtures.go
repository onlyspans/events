package testutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// LoadFixture loads a JSON fixture file and unmarshals it into the provided target
func LoadFixture(t *testing.T, filename string, target interface{}) {
	t.Helper()

	path := filepath.Join("../../test/fixtures", filename)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read fixture file %s: %v", filename, err)
	}

	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("failed to unmarshal fixture %s: %v", filename, err)
	}
}

// LoadFixtureBytes loads a JSON fixture file and returns the raw bytes
func LoadFixtureBytes(t *testing.T, filename string) []byte {
	t.Helper()

	path := filepath.Join("../../test/fixtures", filename)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read fixture file %s: %v", filename, err)
	}

	return data
}

// LoadFixtureString loads a JSON fixture file and returns it as a string
func LoadFixtureString(t *testing.T, filename string) string {
	t.Helper()
	return string(LoadFixtureBytes(t, filename))
}
