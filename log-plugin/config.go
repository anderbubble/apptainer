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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/apptainer/apptainer/pkg/sylog"
	"go.yaml.in/yaml/v4"
)

var (
	ErrNoLogOutput    = errors.New("no log outputs configured")
	ErrInvalidLogType = errors.New("invalid log type")
)

// LogOutputConfig holds the configuration for a single output target
type LogOutputConfig struct {
	Name     string `yaml:"name"`
	Type     string `yaml:"type"`
	Template string `yaml:"template"`
	// Tag is the syslog tag; defaults to "apptainer"
	Tag string `yaml:"tag"`
	// Facility is the syslog facility name; defaults to "user"
	Facility string `yaml:"facility"`
}

// LogConfig holds the configuration for all output targets used by the plugin
type LogConfig struct {
	Output []LogOutputConfig `yaml:"output"`
}

// supportedLogTypes returns a list of log type strings for supported log types
func supportedLogTypes() []string {
	return []string{TypeSyslog}
}

// isLogType returns true if the string t is known as an implemented log type
func isLogType(t string) bool {
	return slices.Contains(supportedLogTypes(), t)
}

// loadConfig loads the YAML configuration file for the plugin instance
func (p *pluginImplementation) loadConfig() error {
	yamlFile, err := os.ReadFile(filepath.Join(p.configPath, "config.yml"))
	if err != nil {
		sylog.Warningf("Plugin %s: could not read config file: %s", pluginName, err)
		return err
	}

	err = yaml.Unmarshal(yamlFile, &p.config)
	if err != nil {
		sylog.Warningf("Plugin %s: could not parse config file: %s", pluginName, err)
		return err
	}

	if len(p.config.Output) == 0 {
		return ErrNoLogOutput
	}

	for _, o := range p.config.Output {
		if !isLogType(o.Type) {
			return fmt.Errorf("%s: %w", o.Type, ErrInvalidLogType)
		}
	}

	return nil
}
