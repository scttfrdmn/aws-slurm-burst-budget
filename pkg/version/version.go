// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package version

import (
	"fmt"
	"runtime"
)

var (
	// Version is the current version of the application
	Version = "0.1.0"

	// GitCommit is the git commit hash
	GitCommit = "unknown"

	// BuildTime is the build timestamp
	BuildTime = "unknown"

	// GoVersion is the Go version used to build the binary
	GoVersion = runtime.Version()
)

// BuildInfo contains version and build information
type BuildInfo struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildTime string `json:"build_time"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

// GetBuildInfo returns the current build information
func GetBuildInfo() *BuildInfo {
	return &BuildInfo{
		Version:   Version,
		GitCommit: GitCommit,
		BuildTime: BuildTime,
		GoVersion: GoVersion,
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
}

// String returns a formatted version string
func String() string {
	return fmt.Sprintf("aws-slurm-burst-budget %s (%s)", Version, GitCommit)
}

// UserAgent returns a user agent string for HTTP requests
func UserAgent() string {
	return fmt.Sprintf("aws-slurm-burst-budget/%s (%s; %s)", Version, runtime.GOOS, runtime.GOARCH)
}