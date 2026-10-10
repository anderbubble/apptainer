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
	"bytes"
	"fmt"
	"log/syslog"
	"path"
	"text/template"
)

// TypeSyslog is the identification string for this type of logger.
const TypeSyslog = "syslog"

const (
	defaultTag      = "apptainer"
	defaultFacility = "user"
)

// facilities maps standard syslog facility names to their values.
var facilities = map[string]syslog.Priority{
	"kern":     syslog.LOG_KERN,
	"user":     syslog.LOG_USER,
	"mail":     syslog.LOG_MAIL,
	"daemon":   syslog.LOG_DAEMON,
	"auth":     syslog.LOG_AUTH,
	"syslog":   syslog.LOG_SYSLOG,
	"lpr":      syslog.LOG_LPR,
	"news":     syslog.LOG_NEWS,
	"uucp":     syslog.LOG_UUCP,
	"cron":     syslog.LOG_CRON,
	"authpriv": syslog.LOG_AUTHPRIV,
	"ftp":      syslog.LOG_FTP,
	"local0":   syslog.LOG_LOCAL0,
	"local1":   syslog.LOG_LOCAL1,
	"local2":   syslog.LOG_LOCAL2,
	"local3":   syslog.LOG_LOCAL3,
	"local4":   syslog.LOG_LOCAL4,
	"local5":   syslog.LOG_LOCAL5,
	"local6":   syslog.LOG_LOCAL6,
	"local7":   syslog.LOG_LOCAL7,
}

// dialSyslog opens a connection to the system logger. Tests replace it to
// capture messages.
var dialSyslog = func(priority syslog.Priority, tag string) (*syslog.Writer, error) {
	return syslog.Dial("", "", priority, tag)
}

// SyslogOutput holds configuration and state for a single output of the logger.
// The plugin may be configured to use multiple outputs, each with their own configuration.
type SyslogOutput struct {
	config     LogOutputConfig
	configPath string
	template   *template.Template
	priority   syslog.Priority
	tag        string
}

// NewSyslogOutput creates a SyslogOutput configured for a particular destination.
func NewSyslogOutput(c LogOutputConfig, cPath string) (*SyslogOutput, error) {
	s := SyslogOutput{
		config:     c,
		configPath: cPath,
		tag:        c.Tag,
	}

	if c.Type != TypeSyslog {
		return nil, fmt.Errorf("invalid type %q != %q", c.Type, TypeSyslog)
	}

	if s.tag == "" {
		s.tag = defaultTag
	}

	facilityName := c.Facility
	if facilityName == "" {
		facilityName = defaultFacility
	}
	facility, ok := facilities[facilityName]
	if !ok {
		return nil, fmt.Errorf("unknown syslog facility %q", facilityName)
	}
	s.priority = facility | syslog.LOG_NOTICE

	if s.config.Template == "" {
		return nil, fmt.Errorf("a template file is required")
	}

	if path.IsAbs(s.config.Template) {
		return nil, fmt.Errorf("template file must be a relative path inside plugin directory")
	}

	tplFile := path.Join(s.configPath, s.config.Template)
	// Unset environment variables render as "" instead of "<no value>"
	tpl, err := template.New(path.Base(tplFile)).Option("missingkey=zero").ParseFiles(tplFile)
	if err != nil {
		return nil, fmt.Errorf("error parsing template file: %v", err)
	}

	s.template = tpl
	return &s, nil
}

// Log writes a syslog entry through this configured output and its template.
// The whole template output is sent as a single message; nothing is sent if
// the template produces no output.
func (s SyslogOutput) Log(data LogData) error {
	var msg bytes.Buffer
	if err := s.template.Execute(&msg, data); err != nil {
		return fmt.Errorf("could not process log message: %v", err)
	}
	if msg.Len() == 0 {
		return nil
	}

	w, err := dialSyslog(s.priority, s.tag)
	if err != nil {
		return fmt.Errorf("could not connect to syslog: %v", err)
	}
	defer w.Close()

	if _, err := w.Write(msg.Bytes()); err != nil {
		return fmt.Errorf("could not write log message: %v", err)
	}

	return nil
}
