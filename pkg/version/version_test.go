// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package version

import (
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetBuildInfo(t *testing.T) {
	buildInfo := GetBuildInfo()

	assert.NotNil(t, buildInfo)
	assert.Equal(t, Version, buildInfo.Version)
	assert.Equal(t, GitCommit, buildInfo.GitCommit)
	assert.Equal(t, BuildTime, buildInfo.BuildTime)
	assert.Equal(t, GoVersion, buildInfo.GoVersion)
	assert.Equal(t, runtime.GOOS, buildInfo.OS)
	assert.Equal(t, runtime.GOARCH, buildInfo.Arch)
}

func TestString(t *testing.T) {
	versionString := String()

	assert.NotEmpty(t, versionString)
	assert.Contains(t, versionString, "aws-slurm-burst-budget")
	assert.Contains(t, versionString, Version)
	assert.Contains(t, versionString, GitCommit)
}

func TestUserAgent(t *testing.T) {
	userAgent := UserAgent()

	assert.NotEmpty(t, userAgent)
	assert.Contains(t, userAgent, "aws-slurm-burst-budget")
	assert.Contains(t, userAgent, Version)
	assert.Contains(t, userAgent, runtime.GOOS)
	assert.Contains(t, userAgent, runtime.GOARCH)

	// Verify format: aws-slurm-burst-budget/version (os; arch)
	parts := strings.Split(userAgent, "/")
	assert.Len(t, parts, 2)
	assert.Equal(t, "aws-slurm-burst-budget", parts[0])

	versionPart := parts[1]
	assert.Contains(t, versionPart, Version)
	assert.Contains(t, versionPart, runtime.GOOS)
	assert.Contains(t, versionPart, runtime.GOARCH)
}

func TestVersionConstants(t *testing.T) {
	// Test that version constants are properly set
	assert.NotEmpty(t, Version)
	assert.NotEmpty(t, GitCommit)
	assert.NotEmpty(t, BuildTime)
	assert.NotEmpty(t, GoVersion)

	// Test version format (should be semantic version)
	assert.True(t, strings.HasPrefix(Version, "0.1.") || strings.HasPrefix(Version, "0.2."))

	// Test Go version format
	assert.True(t, strings.HasPrefix(GoVersion, "go"))
}

func TestBuildInfo_JSONSerialization(t *testing.T) {
	buildInfo := GetBuildInfo()

	// Test that all fields are accessible for JSON serialization
	assert.NotEmpty(t, buildInfo.Version)
	assert.NotEmpty(t, buildInfo.GitCommit)
	assert.NotEmpty(t, buildInfo.BuildTime)
	assert.NotEmpty(t, buildInfo.GoVersion)
	assert.NotEmpty(t, buildInfo.OS)
	assert.NotEmpty(t, buildInfo.Arch)

	// Verify OS and Arch are valid values
	validOS := []string{"linux", "darwin", "windows", "freebsd", "openbsd", "netbsd"}
	assert.Contains(t, validOS, buildInfo.OS)

	validArch := []string{"amd64", "arm64", "386", "arm"}
	assert.Contains(t, validArch, buildInfo.Arch)
}

// Benchmark tests
func BenchmarkGetBuildInfo(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetBuildInfo()
	}
}

func BenchmarkString(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = String()
	}
}

func BenchmarkUserAgent(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = UserAgent()
	}
}
