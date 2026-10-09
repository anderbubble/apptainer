// Copyright (c) Contributors to the Apptainer project, established as
//   Apptainer a Series of LF Projects LLC.
//   For website terms of use, trademark policy, privacy policy and other
//   project policies see https://lfprojects.org/policies
// Copyright (c) 2020-2026, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"path/filepath"

	"github.com/apptainer/apptainer/internal/pkg/buildcfg"
	"github.com/apptainer/apptainer/pkg/cmdline"
	pluginapi "github.com/apptainer/apptainer/pkg/plugin"
	clicallback "github.com/apptainer/apptainer/pkg/plugin/callback/cli"
	"github.com/apptainer/apptainer/pkg/sylog"
	"github.com/spf13/cobra"
)

// Const as we cannot read out of Plugin.Manifest due to initialization loop
const pluginName = "apptainer.org/log-plugin"

// Plugin is the only variable which a plugin MUST export.
// This symbol is accessed by the plugin framework to initialize the plugin.
// Appears unused to the linter, but is consumed by Apptainer when plugin loaded.
//
//nolint:deadcode
var Plugin = pluginapi.Plugin{
	Manifest: pluginapi.Manifest{
		Name:        pluginName,
		Author:      "Apptainer",
		Version:     "1.0.0",
		Description: `A configurable logging plugin for Apptainer`,
	},
	Callbacks: []pluginapi.Callback{
		clicallback.Command(callbackLog),
	},
}

type pluginImplementation struct {
	configPath     string
	config         LogConfig
	data           LogData
	commandManager *cmdline.CommandManager
	command        *cobra.Command
	args           []string
}

func callbackLog(manager *cmdline.CommandManager) {
	rootCmd := manager.GetRootCmd()

	// Keep track of an existing PreRun so we can call it
	f := rootCmd.PersistentPreRunE

	rootCmd.PersistentPreRunE = func(c *cobra.Command, args []string) error {
		p := pluginImplementation{}
		p.configPath = configPath()
		p.commandManager = manager
		p.command = c
		p.args = args
		_ = p.loadConfig()
		p.populateData()
		p.callOutputs()

		// Call any existing PreRun
		if f != nil {
			return f(c, args)
		}
		return nil
	}
}

func (p *pluginImplementation) callOutputs() {
	for _, output := range p.config.Output {
		switch output.Type {
		case TypeSyslog:
			s, err := NewSyslogOutput(output, p.configPath)
			if err != nil {
				sylog.Warningf("Plugin %s: cannot log to %s (%s): %s", pluginName, output.Type, output.Name, err)
				break
			}
			if err := s.Log(p.data); err != nil {
				sylog.Warningf("Plugin %s: cannot log to %s (%s): %s", pluginName, output.Type, output.Name, err)
			}
		default:
			sylog.Warningf("Plugin %s: unknown output type: %s", pluginName, output.Type)
		}
	}
}

// configPath returns the path to the YAML file that configures log outputs
func configPath() string {
	// TODO - come back to this
	// This is a hack as the pkg/plugin API doesn't expose the plugin location
	return filepath.Join(
		buildcfg.APPTAINER_CONFDIR,
		"plugins",
		filepath.FromSlash(pluginName))
}
