// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// Package push tests only test the oras transport against a local registry
package push

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
)

type ctx struct {
	env e2e.TestEnv
}

func (c *ctx) testPushCmd(t *testing.T) {
	e2e.EnsureImage(t, c.env)
	e2e.PrepRegistry(t, c.env)

	// setup file and dir to use as invalid sources
	orasInvalidDir, err := ioutil.TempDir(c.env.TestDir, "oras_push_dir-")
	if err != nil {
		t.Fatalf("unable to create src dir for push tests: %v", err)
	}

	orasInvalidFile, err := e2e.WriteTempFile(orasInvalidDir, "oras_invalid_image-", "Invalid Image Contents")
	if err != nil {
		t.Fatalf("unable to create src file for push tests: %v", err)
	}

	tests := []struct {
		desc          string // case description
		dstURI        string // destination URI for image
		imagePath     string // src image path
		expectSuccess bool   // expection regarding the test's success
	}{
		{
			desc:          "non existent image",
			imagePath:     filepath.Join(orasInvalidDir, "not_an_existing_file.sif"),
			dstURI:        fmt.Sprintf("oras://%s/non_existent:test", c.env.TestRegistry),
			expectSuccess: false,
		},
		{
			desc:          "non SIF file",
			imagePath:     orasInvalidFile,
			dstURI:        fmt.Sprintf("oras://%s/non_sif:test", c.env.TestRegistry),
			expectSuccess: false,
		},
		{
			desc:          "directory",
			imagePath:     orasInvalidDir,
			dstURI:        fmt.Sprintf("oras://%s/directory:test", c.env.TestRegistry),
			expectSuccess: false,
		},
		{
			desc:          "standard SIF push",
			imagePath:     c.env.ImagePath,
			dstURI:        fmt.Sprintf("oras://%s/standard_sif:test", c.env.TestRegistry),
			expectSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			tmpdir, err := ioutil.TempDir(c.env.TestDir, "pull_test.")
			if err != nil {
				t.Fatalf("Failed to create temporary directory for pull test: %+v", err)
			}
			defer os.RemoveAll(tmpdir)

			args := tt.dstURI
			if tt.imagePath != "" {
				args = tt.imagePath + " " + args
			}

			expectedExitCode := 0
			if !tt.expectSuccess {
				expectedExitCode = 255
			}

			e2e.RunSingularity(
				t,
				tt.desc,
				e2e.WithPrivileges(false),
				e2e.WithCommand("push"),
				e2e.WithArgs(strings.Split(args, " ")...),
				e2e.ExpectExit(expectedExitCode),
			)
		})
	}
}

// RunE2ETests is the main func to trigger the test suite
func RunE2ETests(env e2e.TestEnv) func(*testing.T) {
	c := &ctx{
		env: env,
	}

	return func(t *testing.T) {
		e2e.EnsureImage(t, c.env)

		t.Run("push", c.testPushCmd)
	}
}
