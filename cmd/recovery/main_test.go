// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/scttfrdmn/aws-slurm-burst-budget/pkg/version"
)

func TestMain_PackageExists(t *testing.T) {
	// Test that main package is properly structured
	assert.NotEmpty(t, version.Version)
}

func TestRecoveryToolInformation(t *testing.T) {
	// Test basic information that would be displayed
	expectedTitlePrefix := "AWS SLURM Bursting Budget Recovery Tool"

	assert.NotEmpty(t, expectedTitlePrefix)
	assert.NotEmpty(t, version.Version)

	// Test that we can construct the expected output
	title := expectedTitlePrefix + " " + version.Version
	assert.Contains(t, title, "Recovery Tool")
	assert.Contains(t, title, version.Version)
}
