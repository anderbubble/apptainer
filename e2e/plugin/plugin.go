// Copyright (c) Contributors to the Apptainer project, established as
//   Apptainer a Series of LF Projects LLC.
//   For website terms of use, trademark policy, privacy policy and other
//   project policies see https://lfprojects.org/policies
// Copyright (c) 2020-2022, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package plugin

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/apptainer/apptainer/e2e/internal/e2e"
	"github.com/apptainer/apptainer/e2e/internal/testhelper"
	"github.com/apptainer/apptainer/internal/pkg/buildcfg"
	"github.com/google/uuid"
	"golang.org/x/sys/unix"
)

type ctx struct {
	env e2e.TestEnv
}

func (c ctx) testPluginBasic(t *testing.T) {
	pluginName := "github.com/apptainer/apptainer/e2e-plugin"

	// plugin code directory
	pluginDir, cleanup := e2e.MakeTempDir(t, "testdata", "e2e-plugin-dir-", "")
	defer cleanup(t)

	// plugin sif file
	sifFile := filepath.Join(pluginDir, "plugin.sif")

	tests := []struct {
		name       string
		profile    e2e.Profile
		command    string
		args       []string
		expectExit int
		expectOp   e2e.ApptainerCmdResultOp
	}{
		{
			name:       "Create",
			profile:    e2e.UserProfile,
			command:    "plugin create",
			args:       []string{pluginDir, pluginName},
			expectExit: 0,
		},
		{
			name:       "ListNoPlugins",
			profile:    e2e.UserProfile,
			command:    "plugin list",
			args:       []string{},
			expectExit: 0,
			expectOp:   e2e.ExpectOutput(e2e.ExactMatch, "There are no plugins installed."),
		},
		{
			name:       "Compile",
			profile:    e2e.UserProfile,
			command:    "plugin compile",
			args:       []string{"--out", sifFile, pluginDir},
			expectExit: 0,
		},
		{
			name:       "Install",
			profile:    e2e.RootProfile,
			command:    "plugin install",
			args:       []string{sifFile},
			expectExit: 0,
		},
		{
			name:       "InstallAsUser",
			profile:    e2e.UserProfile,
			command:    "plugin install",
			args:       []string{sifFile},
			expectExit: 255,
		},
		{
			name:       "ListAfterInstall",
			profile:    e2e.UserProfile,
			command:    "plugin list",
			args:       []string{},
			expectExit: 0,
			expectOp:   e2e.ExpectOutput(e2e.ContainMatch, "yes  "+pluginName),
		},
		{
			name:       "Disable",
			profile:    e2e.RootProfile,
			command:    "plugin disable",
			args:       []string{pluginName},
			expectExit: 0,
		},
		{
			name:       "ListAfterDisable",
			profile:    e2e.UserProfile,
			command:    "plugin list",
			args:       []string{},
			expectExit: 0,
			expectOp:   e2e.ExpectOutput(e2e.ContainMatch, "no  "+pluginName),
		},
		{
			name:       "DisableAsUser",
			profile:    e2e.UserProfile,
			command:    "plugin disable",
			args:       []string{pluginName},
			expectExit: 255,
		},
		{
			name:       "Enable",
			profile:    e2e.RootProfile,
			command:    "plugin enable",
			args:       []string{pluginName},
			expectExit: 0,
		},
		{
			name:       "ListAfterEnable",
			profile:    e2e.UserProfile,
			command:    "plugin list",
			args:       []string{},
			expectExit: 0,
			expectOp:   e2e.ExpectOutput(e2e.ContainMatch, "yes  "+pluginName),
		},
		{
			name:       "EnableAsUser",
			profile:    e2e.UserProfile,
			command:    "plugin enable",
			args:       []string{pluginName},
			expectExit: 255,
		},
		{
			name:       "InspectFromName",
			profile:    e2e.UserProfile,
			command:    "plugin inspect",
			args:       []string{pluginName},
			expectExit: 0,
		},
		{
			name:       "InspectFromSIF",
			profile:    e2e.UserProfile,
			command:    "plugin inspect",
			args:       []string{sifFile},
			expectExit: 0,
			expectOp:   e2e.ExpectOutput(e2e.ContainMatch, "Name: "+pluginName),
		},
		{
			name:       "UninstallAsUser",
			profile:    e2e.UserProfile,
			command:    "plugin uninstall",
			args:       []string{pluginName},
			expectExit: 255,
		},
		{
			name:       "Uninstall",
			profile:    e2e.RootProfile,
			command:    "plugin uninstall",
			args:       []string{pluginName},
			expectExit: 0,
		},
		{
			name:       "ListAfterUninstall",
			profile:    e2e.UserProfile,
			command:    "plugin list",
			args:       []string{},
			expectExit: 0,
			expectOp:   e2e.ExpectOutput(e2e.ExactMatch, "There are no plugins installed."),
		},
	}

	for _, tt := range tests {
		var stderr, stdout string
		c.env.RunApptainer(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(tt.profile),
			e2e.WithCommand(tt.command),
			e2e.WithArgs(tt.args...),
			e2e.ExpectExit(tt.expectExit, tt.expectOp, e2e.GetStreams(&stdout, &stderr)),
		)
		t.Logf("stdout:\n%s\n\nstderr:\n%s\n\n", stdout, stderr)
	}
}

func (c ctx) testCLICallbacks(t *testing.T) {
	pluginDir := "./plugin/testdata/cli"
	pluginName := "github.com/apptainer/apptainer/e2e-cli-plugin"

	// plugin sif file
	sifFile := filepath.Join(c.env.TestDir, "plugin.sif")
	defer os.Remove(sifFile)

	tests := []struct {
		name       string
		profile    e2e.Profile
		command    string
		args       []string
		expectExit int
	}{
		{
			name:       "Compile",
			profile:    e2e.UserProfile,
			command:    "plugin compile",
			args:       []string{"--out", sifFile, pluginDir},
			expectExit: 0,
		},
		{
			name:       "Install",
			profile:    e2e.RootProfile,
			command:    "plugin install",
			args:       []string{sifFile},
			expectExit: 0,
		},
		{
			name:       "CLICallback",
			profile:    e2e.UserProfile,
			command:    "exit",
			args:       []string{"42"},
			expectExit: 42,
		},
		{
			name:       "ApptainerConfigCallback",
			profile:    e2e.UserProfile,
			command:    "shell",
			args:       []string{c.env.TestDir},
			expectExit: 43,
		},
		{
			name:       "Uninstall",
			profile:    e2e.RootProfile,
			command:    "plugin uninstall",
			args:       []string{pluginName},
			expectExit: 0,
		},
	}

	for _, tt := range tests {
		var stderr, stdout string
		c.env.RunApptainer(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithGlobalOptions("--debug"),
			e2e.WithProfile(tt.profile),
			e2e.WithCommand(tt.command),
			e2e.WithArgs(tt.args...),
			e2e.ExpectExit(tt.expectExit, e2e.GetStreams(&stdout, &stderr)),
		)
		t.Logf("stdout:\n%s\n\nstderr:\n%s\n\n", stdout, stderr)
	}
}

func (c ctx) testApptainerCallbacks(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	pluginDir := "./plugin/testdata/runtime_apptainer"
	pluginName := "github.com/apptainer/apptainer/e2e-runtime-plugin"

	// plugin sif file
	sifFile := filepath.Join(c.env.TestDir, "plugin.sif")
	defer os.Remove(sifFile)

	tests := []struct {
		name       string
		profile    e2e.Profile
		command    string
		args       []string
		expectExit int
	}{
		{
			name:       "Compile",
			profile:    e2e.UserProfile,
			command:    "plugin compile",
			args:       []string{"--out", sifFile, pluginDir},
			expectExit: 0,
		},
		{
			name:       "Install",
			profile:    e2e.RootProfile,
			command:    "plugin install",
			args:       []string{sifFile},
			expectExit: 0,
		},
		{
			name:       "MonitorCallback",
			profile:    e2e.UserProfile,
			command:    "exec",
			args:       []string{c.env.ImagePath, "true"},
			expectExit: 42,
		},
		{
			name:       "PostStartProcessCallback",
			profile:    e2e.UserProfile,
			command:    "exec",
			args:       []string{"--contain", c.env.ImagePath, "true"},
			expectExit: 43,
		},
		{
			name:       "Uninstall",
			profile:    e2e.RootProfile,
			command:    "plugin uninstall",
			args:       []string{pluginName},
			expectExit: 0,
		},
	}

	for _, tt := range tests {
		var stderr, stdout string
		c.env.RunApptainer(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithGlobalOptions("--debug"),
			e2e.WithProfile(tt.profile),
			e2e.WithCommand(tt.command),
			e2e.WithArgs(tt.args...),
			e2e.ExpectExit(tt.expectExit, e2e.GetStreams(&stdout, &stderr)),
		)
		t.Logf("stdout:\n%s\n\nstderr:\n%s\n\n", stdout, stderr)
	}
}

// setupLogPluginConfig bind mounts a temporary directory over the plugins
// configuration directory, holding a log-plugin config.yml that logs with tag
// through a template with literal text between expressions.
func (c ctx) setupLogPluginConfig(t *testing.T, tag string) (cleanup func(t *testing.T)) {
	pluginsDir := filepath.Join(buildcfg.APPTAINER_CONFDIR, "plugins")
	tmpDir := filepath.Join(c.env.TestDir, "log-plugin-conf")
	created := false

	e2e.Privileged(func(t *testing.T) {
		confDir := filepath.Join(tmpDir, "apptainer.org", "log-plugin")
		if err := os.MkdirAll(confDir, 0o755); err != nil {
			t.Fatalf("while creating %s: %s", confDir, err)
		}
		config := "output:\n" +
			"- name: e2e\n" +
			"  type: syslog\n" +
			"  tag: " + tag + "\n" +
			"  template: e2e.tpl\n"
		if err := os.WriteFile(filepath.Join(confDir, "config.yml"), []byte(config), 0o644); err != nil {
			t.Fatalf("while writing config.yml: %s", err)
		}
		tpl := "e2e cmd={{.Command}} image={{.Image}}"
		if err := os.WriteFile(filepath.Join(confDir, "e2e.tpl"), []byte(tpl), 0o644); err != nil {
			t.Fatalf("while writing e2e.tpl: %s", err)
		}

		if err := os.Mkdir(pluginsDir, 0o755); err == nil {
			created = true
		} else if !os.IsExist(err) {
			t.Fatalf("while creating %s: %s", pluginsDir, err)
		}
		if err := unix.Mount(tmpDir, pluginsDir, "", unix.MS_BIND, ""); err != nil {
			t.Fatalf("while mounting %s to %s: %s", tmpDir, pluginsDir, err)
		}
	})(t)

	return e2e.Privileged(func(t *testing.T) {
		if err := unix.Unmount(pluginsDir, 0); err != nil {
			t.Errorf("while unmounting %s: %s", pluginsDir, err)
		}
		if created {
			os.Remove(pluginsDir)
		}
		os.RemoveAll(tmpDir)
	})
}

// journalLines returns the messages logged to the journal under tag, waiting
// for up to 10 seconds for the first to arrive.
func journalLines(t *testing.T, tag string) (lines []string) {
	e2e.Privileged(func(t *testing.T) {
		for range 20 {
			out, err := exec.Command("journalctl", "-t", tag, "-o", "cat", "--no-pager").Output()
			if err != nil {
				t.Fatalf("while running journalctl: %s", err)
			}
			if s := strings.TrimSpace(string(out)); s != "" {
				lines = strings.Split(s, "\n")
				return
			}
			time.Sleep(500 * time.Millisecond)
		}
	})(t)
	return lines
}

func (c ctx) testLogPlugin(t *testing.T) {
	if fi, err := os.Stat("/dev/log"); err != nil || fi.Mode()&os.ModeSocket == 0 {
		t.Skip("no syslog socket at /dev/log")
	}
	if _, err := exec.LookPath("journalctl"); err != nil {
		t.Skip("journalctl not found")
	}
	e2e.EnsureImage(t, c.env)

	pluginDir := "../log-plugin"
	pluginName := "apptainer.org/log-plugin"
	tag := "apptainer-e2e-" + uuid.NewString()

	sifFile := filepath.Join(c.env.TestDir, "log-plugin.sif")
	defer os.Remove(sifFile)

	cleanup := c.setupLogPluginConfig(t, tag)
	defer cleanup(t)

	steps := []struct {
		name    string
		profile e2e.Profile
		command string
		args    []string
	}{
		{"Compile", e2e.UserProfile, "plugin compile", []string{"--out", sifFile, pluginDir}},
		{"Install", e2e.RootProfile, "plugin install", []string{sifFile}},
		{"Exec", e2e.UserProfile, "exec", []string{c.env.ImagePath, "true"}},
		{"Uninstall", e2e.RootProfile, "plugin uninstall", []string{pluginName}},
	}
	for _, s := range steps {
		c.env.RunApptainer(
			t,
			e2e.AsSubtest(s.name),
			e2e.WithProfile(s.profile),
			e2e.WithCommand(s.command),
			e2e.WithArgs(s.args...),
			e2e.ExpectExit(0),
		)
	}

	// The whole template output must arrive as a single message.
	want := "e2e cmd=exec image=" + c.env.ImagePath
	lines := journalLines(t, tag)
	count := 0
	for _, l := range lines {
		if l == want {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly one %q journal line for tag %s, got:\n%s", want, tag, strings.Join(lines, "\n"))
	}
}

// E2ETests is the main func to trigger the test suite
func E2ETests(env e2e.TestEnv) testhelper.Tests {
	c := ctx{
		env: env,
	}

	np := testhelper.NoParallel

	return testhelper.Tests{
		"basic":               np(c.testPluginBasic),
		"CLI_callbacks":       np(c.testCLICallbacks),
		"Apptainer_callbacks": np(c.testApptainerCallbacks),
		"log_plugin":          np(c.testLogPlugin),
	}
}
