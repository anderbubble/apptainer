// Copyright (c) Contributors to the Apptainer project, established as
//   Apptainer a Series of LF Projects LLC.
//   For website terms of use, trademark policy, privacy policy and other
//   project policies see https://lfprojects.org/policies
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package apptainer

import (
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

func TestGoModFlag(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, dir string)
		want  string
	}{
		{
			name:  "NoVendor",
			setup: func(*testing.T, string) {},
			want:  "-mod=readonly",
		},
		{
			name: "VendorWithoutModulesTxt",
			setup: func(t *testing.T, dir string) {
				assert.NilError(t, os.Mkdir(filepath.Join(dir, "vendor"), 0o755))
			},
			want: "-mod=readonly",
		},
		{
			name: "Vendor",
			setup: func(t *testing.T, dir string) {
				assert.NilError(t, os.Mkdir(filepath.Join(dir, "vendor"), 0o755))
				assert.NilError(t, os.WriteFile(filepath.Join(dir, "vendor", "modules.txt"), nil, 0o644))
			},
			want: "-mod=vendor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setup(t, dir)
			assert.Equal(t, goModFlag(dir), tt.want)
		})
	}
}
