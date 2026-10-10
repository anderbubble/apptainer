// Copyright (c) Contributors to the Apptainer project, established as
//   Apptainer a Series of LF Projects LLC.
//   For website terms of use, trademark policy, privacy policy and other
//   project policies see https://lfprojects.org/policies
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"log/syslog"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

type syslogMsg struct {
	Priority syslog.Priority
	Tag      string
	Text     string
}

// Format written by log/syslog: <PRI>TIMESTAMP [HOSTNAME ]TAG[PID]: MSG\n
var syslogMsgRe = regexp.MustCompile(`(?s)^<(\d+)>.*? (\S+)\[\d+\]: (.*?)\n?$`)

// captureSyslog points dialSyslog at a unixgram socket and returns a function
// that collects every message received so far.
func captureSyslog(t *testing.T) func() []syslogMsg {
	t.Helper()
	sock := filepath.Join(t.TempDir(), "log")
	conn, err := net.ListenUnixgram("unixgram", &net.UnixAddr{Name: sock, Net: "unixgram"})
	assert.NilError(t, err)
	t.Cleanup(func() { conn.Close() })

	orig := dialSyslog
	dialSyslog = func(p syslog.Priority, tag string) (*syslog.Writer, error) {
		return syslog.Dial("unixgram", sock, p, tag)
	}
	t.Cleanup(func() { dialSyslog = orig })

	return func() []syslogMsg {
		var msgs []syslogMsg
		buf := make([]byte, 65536)
		for {
			assert.NilError(t, conn.SetReadDeadline(time.Now().Add(100*time.Millisecond)))
			n, err := conn.Read(buf)
			if err != nil {
				return msgs
			}
			m := syslogMsgRe.FindStringSubmatch(string(buf[:n]))
			assert.Assert(t, m != nil, "unparseable syslog message %q", buf[:n])
			pri, err := strconv.Atoi(m[1])
			assert.NilError(t, err)
			msgs = append(msgs, syslogMsg{Priority: syslog.Priority(pri), Tag: m[2], Text: m[3]})
		}
	}
}

// writeTemplate writes tpl into dir as test.tpl.
func writeTemplate(t *testing.T, dir, tpl string) {
	t.Helper()
	assert.NilError(t, os.WriteFile(filepath.Join(dir, "test.tpl"), []byte(tpl), 0o644))
}

func TestSyslogOutputLog(t *testing.T) {
	data := LogData{User: "alice", Command: "exec"}

	tests := []struct {
		name     string
		tag      string
		facility string
		argv0    string
		tpl      string
		want     []syslogMsg
	}{
		{
			name: "Defaults",
			tpl:  `{{.Command}}`,
			want: []syslogMsg{{syslog.LOG_USER | syslog.LOG_NOTICE, "apptainer", "exec"}},
		},
		{
			name:  "DefaultTagIgnoresArgv0",
			argv0: "/usr/bin/singularity",
			tpl:   `{{.Command}}`,
			want:  []syslogMsg{{syslog.LOG_USER | syslog.LOG_NOTICE, "apptainer", "exec"}},
		},
		{
			name: "Tag",
			tag:  "singularity",
			tpl:  `{{.Command}}`,
			want: []syslogMsg{{syslog.LOG_USER | syslog.LOG_NOTICE, "singularity", "exec"}},
		},
		{
			name:     "FacilityLocal3",
			facility: "local3",
			tpl:      `{{.Command}}`,
			want:     []syslogMsg{{syslog.LOG_LOCAL3 | syslog.LOG_NOTICE, "apptainer", "exec"}},
		},
		{
			name:     "FacilityAuthpriv",
			facility: "authpriv",
			tpl:      `{{.Command}}`,
			want:     []syslogMsg{{syslog.LOG_AUTHPRIV | syslog.LOG_NOTICE, "apptainer", "exec"}},
		},
		{
			name: "LiteralTextBetweenExpressions",
			tpl:  `user={{.User}} cmd={{.Command}} end`,
			want: []syslogMsg{{syslog.LOG_USER | syslog.LOG_NOTICE, "apptainer", "user=alice cmd=exec end"}},
		},
		{
			name: "UnsetEnvQuoted",
			tpl:  `LOGNAME={{printf "%q" .Env.LOGNAME}}`,
			want: []syslogMsg{{syslog.LOG_USER | syslog.LOG_NOTICE, "apptainer", `LOGNAME=""`}},
		},
		{
			name: "UnsetEnvPrinted",
			tpl:  `[{{.Env.LOGNAME}}]`,
			want: []syslogMsg{{syslog.LOG_USER | syslog.LOG_NOTICE, "apptainer", "[]"}},
		},
		{
			name: "EmptyOutput",
			tpl:  `{{if eq .Command "run"}}ran{{end}}`,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.argv0 != "" {
				orig := os.Args[0]
				os.Args[0] = tt.argv0
				t.Cleanup(func() { os.Args[0] = orig })
			}
			received := captureSyslog(t)

			dir := t.TempDir()
			writeTemplate(t, dir, tt.tpl)
			s, err := NewSyslogOutput(LogOutputConfig{
				Name:     tt.name,
				Type:     TypeSyslog,
				Template: "test.tpl",
				Tag:      tt.tag,
				Facility: tt.facility,
			}, dir)
			assert.NilError(t, err)
			assert.NilError(t, s.Log(data))

			assert.DeepEqual(t, tt.want, received())
		})
	}
}

func TestNewSyslogOutputUnknownFacility(t *testing.T) {
	dir := t.TempDir()
	writeTemplate(t, dir, `{{.Command}}`)
	_, err := NewSyslogOutput(LogOutputConfig{
		Type:     TypeSyslog,
		Template: "test.tpl",
		Facility: "nosuchfacility",
	}, dir)
	assert.ErrorContains(t, err, `unknown syslog facility "nosuchfacility"`)
}

func TestCallOutputsSkipsInvalidOutput(t *testing.T) {
	received := captureSyslog(t)

	dir := t.TempDir()
	writeTemplate(t, dir, `{{.Command}}`)
	p := pluginImplementation{
		configPath: dir,
		config: LogConfig{Output: []LogOutputConfig{
			{Name: "bad", Type: TypeSyslog, Template: "test.tpl", Facility: "nosuchfacility"},
			{Name: "good", Type: TypeSyslog, Template: "test.tpl"},
		}},
		data: LogData{Command: "exec"},
	}
	p.callOutputs()

	assert.DeepEqual(t, []syslogMsg{{syslog.LOG_USER | syslog.LOG_NOTICE, "apptainer", "exec"}}, received())
}
