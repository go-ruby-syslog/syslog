// Copyright (c) the go-ruby-syslog/syslog authors
//
// SPDX-License-Identifier: BSD-3-Clause

package syslog

import "testing"

// TestFacilityConstants pins every facility constant to MRI 4.0.5's value
// (facility code << 3), so a drift is caught without a Ruby runtime.
func TestFacilityConstants(t *testing.T) {
	cases := []struct {
		name string
		got  int
		want int
	}{
		{"LOG_KERN", LOG_KERN, 0},
		{"LOG_USER", LOG_USER, 8},
		{"LOG_MAIL", LOG_MAIL, 16},
		{"LOG_DAEMON", LOG_DAEMON, 24},
		{"LOG_AUTH", LOG_AUTH, 32},
		{"LOG_SYSLOG", LOG_SYSLOG, 40},
		{"LOG_LPR", LOG_LPR, 48},
		{"LOG_NEWS", LOG_NEWS, 56},
		{"LOG_UUCP", LOG_UUCP, 64},
		{"LOG_CRON", LOG_CRON, 72},
		{"LOG_AUTHPRIV", LOG_AUTHPRIV, 80},
		{"LOG_FTP", LOG_FTP, 88},
		{"LOG_LOCAL0", LOG_LOCAL0, 128},
		{"LOG_LOCAL1", LOG_LOCAL1, 136},
		{"LOG_LOCAL2", LOG_LOCAL2, 144},
		{"LOG_LOCAL3", LOG_LOCAL3, 152},
		{"LOG_LOCAL4", LOG_LOCAL4, 160},
		{"LOG_LOCAL5", LOG_LOCAL5, 168},
		{"LOG_LOCAL6", LOG_LOCAL6, 176},
		{"LOG_LOCAL7", LOG_LOCAL7, 184},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %d, want %d", c.name, c.got, c.want)
		}
	}
}

// TestLevelAndOptionConstants pins the severity levels and option flags.
func TestLevelAndOptionConstants(t *testing.T) {
	cases := []struct {
		name string
		got  int
		want int
	}{
		{"LOG_EMERG", LOG_EMERG, 0},
		{"LOG_ALERT", LOG_ALERT, 1},
		{"LOG_CRIT", LOG_CRIT, 2},
		{"LOG_ERR", LOG_ERR, 3},
		{"LOG_WARNING", LOG_WARNING, 4},
		{"LOG_NOTICE", LOG_NOTICE, 5},
		{"LOG_INFO", LOG_INFO, 6},
		{"LOG_DEBUG", LOG_DEBUG, 7},
		{"LOG_PID", LOG_PID, 1},
		{"LOG_CONS", LOG_CONS, 2},
		{"LOG_ODELAY", LOG_ODELAY, 4},
		{"LOG_NDELAY", LOG_NDELAY, 8},
		{"LOG_NOWAIT", LOG_NOWAIT, 16},
		{"LOG_PERROR", LOG_PERROR, 32},
		{"LOG_PRIMASK", LOG_PRIMASK, 7},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %d, want %d", c.name, c.got, c.want)
		}
	}
}

// TestPriority checks the pri = facility | severity encoding and its inverse
// decomposition (Facility/Severity).
func TestPriority(t *testing.T) {
	cases := []struct {
		fac, sev, pri int
	}{
		{LOG_KERN, LOG_EMERG, 0},
		{LOG_USER, LOG_INFO, 14},
		{LOG_MAIL, LOG_NOTICE, 21},
		{LOG_LOCAL0, LOG_ERR, 131},
		{LOG_LOCAL7, LOG_DEBUG, 191},
	}
	for _, c := range cases {
		if got := Priority(c.fac, c.sev); got != c.pri {
			t.Errorf("Priority(%d,%d) = %d, want %d", c.fac, c.sev, got, c.pri)
		}
		if got := Facility(c.pri); got != c.fac {
			t.Errorf("Facility(%d) = %d, want %d", c.pri, got, c.fac)
		}
		if got := Severity(c.pri); got != c.sev {
			t.Errorf("Severity(%d) = %d, want %d", c.pri, got, c.sev)
		}
	}
}

// TestLogMask checks Syslog::LOG_MASK(pri) == 1 << pri.
func TestLogMask(t *testing.T) {
	cases := []struct{ pri, want int }{
		{LOG_EMERG, 1},
		{LOG_ERR, 8},
		{LOG_DEBUG, 128},
	}
	for _, c := range cases {
		if got := LogMask(c.pri); got != c.want {
			t.Errorf("LogMask(%d) = %d, want %d", c.pri, got, c.want)
		}
	}
}

// TestLogUpto checks Syslog::LOG_UPTO(pri) == (1 << (pri+1)) - 1.
func TestLogUpto(t *testing.T) {
	cases := []struct{ pri, want int }{
		{LOG_EMERG, 1},
		{LOG_ERR, 15},
		{LOG_DEBUG, 255},
	}
	for _, c := range cases {
		if got := LogUpto(c.pri); got != c.want {
			t.Errorf("LogUpto(%d) = %d, want %d", c.pri, got, c.want)
		}
	}
}
