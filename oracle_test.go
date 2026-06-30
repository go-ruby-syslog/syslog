// Copyright (c) the go-ruby-syslog/syslog authors
//
// SPDX-License-Identifier: BSD-3-Clause

package syslog

import (
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// rubyBin locates a usable `ruby` once. The oracle tests skip themselves when it
// is absent (the qemu cross-arch lanes, the Windows lane, and any host without
// Ruby), so the deterministic, ruby-free suite alone drives the 100% gate there.
// On Windows the syslog gem is not available, so we skip unconditionally.
func rubyBin(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("ruby -rsyslog unavailable on Windows; skipping MRI oracle")
	}
	path, err := exec.LookPath("ruby")
	if err != nil {
		t.Skip("ruby not on PATH; skipping MRI Syslog oracle")
	}
	return path
}

// rubyEval runs a Ruby script with the syslog gem loaded and returns its stdout.
// The script $stdout.binmode's itself so a stray CRLF can never perturb the
// comparison. If `ruby` lacks the syslog gem (it is a default gem but can be
// absent on a stripped build) the test skips rather than fails.
func rubyEval(t *testing.T, bin, script string) string {
	t.Helper()
	cmd := exec.Command(bin, "-rsyslog", "-e", "$stdout.binmode\n"+script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		s := string(out)
		if strings.Contains(s, "cannot load such file") && strings.Contains(s, "syslog") {
			t.Skip("ruby has no syslog gem; skipping MRI oracle")
		}
		t.Fatalf("ruby error: %v\nscript:\n%s\noutput:\n%s", err, script, s)
	}
	return string(out)
}

// TestOracleConstants asserts every Go constant equals MRI's Syslog::LOG_* value.
func TestOracleConstants(t *testing.T) {
	bin := rubyBin(t)
	names := []struct {
		name string
		got  int
	}{
		{"LOG_KERN", LOG_KERN}, {"LOG_USER", LOG_USER}, {"LOG_MAIL", LOG_MAIL},
		{"LOG_DAEMON", LOG_DAEMON}, {"LOG_AUTH", LOG_AUTH}, {"LOG_SYSLOG", LOG_SYSLOG},
		{"LOG_LPR", LOG_LPR}, {"LOG_NEWS", LOG_NEWS}, {"LOG_UUCP", LOG_UUCP},
		{"LOG_CRON", LOG_CRON}, {"LOG_AUTHPRIV", LOG_AUTHPRIV}, {"LOG_FTP", LOG_FTP},
		{"LOG_LOCAL0", LOG_LOCAL0}, {"LOG_LOCAL1", LOG_LOCAL1}, {"LOG_LOCAL2", LOG_LOCAL2},
		{"LOG_LOCAL3", LOG_LOCAL3}, {"LOG_LOCAL4", LOG_LOCAL4}, {"LOG_LOCAL5", LOG_LOCAL5},
		{"LOG_LOCAL6", LOG_LOCAL6}, {"LOG_LOCAL7", LOG_LOCAL7},
		{"LOG_EMERG", LOG_EMERG}, {"LOG_ALERT", LOG_ALERT}, {"LOG_CRIT", LOG_CRIT},
		{"LOG_ERR", LOG_ERR}, {"LOG_WARNING", LOG_WARNING}, {"LOG_NOTICE", LOG_NOTICE},
		{"LOG_INFO", LOG_INFO}, {"LOG_DEBUG", LOG_DEBUG},
		{"LOG_PID", LOG_PID}, {"LOG_CONS", LOG_CONS}, {"LOG_ODELAY", LOG_ODELAY},
		{"LOG_NDELAY", LOG_NDELAY}, {"LOG_NOWAIT", LOG_NOWAIT}, {"LOG_PERROR", LOG_PERROR},
	}
	var b strings.Builder
	for _, n := range names {
		b.WriteString("print Syslog::" + n.name + "; print \"\\n\"\n")
	}
	out := rubyEval(t, bin, b.String())
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != len(names) {
		t.Fatalf("got %d values, want %d:\n%s", len(lines), len(names), out)
	}
	for i, n := range names {
		want, err := strconv.Atoi(strings.TrimSpace(lines[i]))
		if err != nil {
			t.Fatalf("%s: parse %q: %v", n.name, lines[i], err)
		}
		if n.got != want {
			t.Errorf("Syslog::%s = %d (MRI), Go = %d", n.name, want, n.got)
		}
	}
}

// TestOraclePriorityAndMasks checks Priority/LogMask/LogUpto against MRI's own
// facility|severity composition and its LOG_MASK / LOG_UPTO macros.
func TestOraclePriorityAndMasks(t *testing.T) {
	bin := rubyBin(t)
	out := rubyEval(t, bin, `
include Syslog::Constants
print(Syslog::LOG_LOCAL0 | Syslog::LOG_ERR); print "\n"
print(Syslog::LOG_MAIL | Syslog::LOG_NOTICE); print "\n"
print Syslog::LOG_MASK(Syslog::LOG_ERR); print "\n"
print Syslog::LOG_MASK(Syslog::LOG_DEBUG); print "\n"
print Syslog::LOG_UPTO(Syslog::LOG_ERR); print "\n"
print Syslog::LOG_UPTO(Syslog::LOG_DEBUG); print "\n"
`)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	want := []int{
		Priority(LOG_LOCAL0, LOG_ERR),
		Priority(LOG_MAIL, LOG_NOTICE),
		LogMask(LOG_ERR),
		LogMask(LOG_DEBUG),
		LogUpto(LOG_ERR),
		LogUpto(LOG_DEBUG),
	}
	if len(lines) != len(want) {
		t.Fatalf("got %d lines, want %d:\n%s", len(lines), len(want), out)
	}
	for i, w := range want {
		got, err := strconv.Atoi(strings.TrimSpace(lines[i]))
		if err != nil {
			t.Fatalf("parse %q: %v", lines[i], err)
		}
		if got != w {
			t.Errorf("line %d: MRI = %d, Go = %d", i, got, w)
		}
	}
}

// TestOracleFormat checks that Sprintf reproduces MRI's Kernel#sprintf — the
// exact message-building step Syslog.log performs (rb_f_sprintf) — for the
// directive subset syslog messages use.
func TestOracleFormat(t *testing.T) {
	bin := rubyBin(t)
	cases := []struct {
		format string
		args   []any
		ruby   string // the Ruby args literal
	}{
		{"%s[%d]: %s", []any{"id", 42, "msg"}, `"id", 42, "msg"`},
		{"hex=%x oct=%o", []any{255, 8}, `255, 8`},
		{"%05d|%-5d|", []any{42, 3}, `42, 3`},
		{"100%% done", nil, ``},
		{"f=%.2f", []any{3.14159}, `3.14159`},
	}
	for _, c := range cases {
		goOut, err := Sprintf(c.format, c.args...)
		if err != nil {
			t.Fatalf("Sprintf(%q) error: %v", c.format, err)
		}
		script := `print(sprintf(` + strconv.Quote(c.format)
		if c.ruby != "" {
			script += ", " + c.ruby
		}
		script += "))"
		rubyOut := rubyEval(t, bin, script)
		if goOut != rubyOut {
			t.Errorf("format %q: Go = %q, MRI = %q", c.format, goOut, rubyOut)
		}
	}
}
