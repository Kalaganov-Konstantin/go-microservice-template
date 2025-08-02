package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGet(t *testing.T) {
	version := Get()
	assert.Equal(t, "dev", version, "Default version should be 'dev'")
}

func TestGet_WithCustomVersion(t *testing.T) {
	originalVersion := Version
	defer func() {
		Version = originalVersion
	}()

	Version = "1.2.3"

	version := Get()
	assert.Equal(t, "1.2.3", version, "Should return custom version")
}

func TestInfo(t *testing.T) {
	info := Info()

	expected := BuildInfo{
		Version:   "dev",
		BuildTime: "unknown",
		GitCommit: "unknown",
	}

	assert.Equal(t, expected, info, "Should return BuildInfo with default values")
}

func TestInfo_WithCustomValues(t *testing.T) {
	originalVersion := Version
	originalBuildTime := BuildTime
	originalGitCommit := GitCommit
	defer func() {
		Version = originalVersion
		BuildTime = originalBuildTime
		GitCommit = originalGitCommit
	}()

	Version = "2.1.0"
	BuildTime = "2025-08-01T10:00:00Z"
	GitCommit = "abc123def456"

	info := Info()

	expected := BuildInfo{
		Version:   "2.1.0",
		BuildTime: "2025-08-01T10:00:00Z",
		GitCommit: "abc123def456",
	}

	assert.Equal(t, expected, info, "Should return BuildInfo with custom values")
}

func TestConcurrentAccess(t *testing.T) {
	const numGoroutines = 10
	results := make(chan string, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			results <- Get()
		}()
	}

	for i := 0; i < numGoroutines; i++ {
		version := <-results
		assert.Equal(t, "dev", version, "All goroutines should get same version")
	}
}

func BenchmarkGet(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Get()
	}
}

func BenchmarkInfo(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Info()
	}
}
