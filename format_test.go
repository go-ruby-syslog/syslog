// Copyright (c) the go-ruby-syslog/syslog authors
//
// SPDX-License-Identifier: BSD-3-Clause

package syslog

import "testing"

// TestSprintf covers the Ruby-sprintf directive subset and verbatim formats.
func TestSprintf(t *testing.T) {
	cases := []struct {
		format string
		args   []any
		want   string
	}{
		{"plain message", nil, "plain message"},
		{"100%% done", nil, "100% done"},
		{"%s[%d]: %s", []any{"id", 42, "msg"}, "id[42]: msg"},
		{"%i items", []any{7}, "7 items"},
		{"%u count", []any{9}, "9 count"},
		{"hex=%x HEX=%X oct=%o bin=%b", []any{255, 255, 8, 5}, "hex=ff HEX=FF oct=10 bin=101"},
		{"f=%.2f e=%e g=%g", []any{3.14159, 1000.0, 0.5}, "f=3.14 e=1.000000e+03 g=0.5"},
		{"char=%c", []any{65}, "char=A"},
		{"%-5d|", []any{3}, "3    |"},
		{"%05d", []any{42}, "00042"},
		{"ptr=%p", []any{[]int{1, 2}}, "ptr=[1 2]"},
		{"escaped %%m survives", nil, "escaped %m survives"},
	}
	for _, c := range cases {
		got, err := Sprintf(c.format, c.args...)
		if err != nil {
			t.Errorf("Sprintf(%q) error: %v", c.format, err)
			continue
		}
		if got != c.want {
			t.Errorf("Sprintf(%q, %v) = %q, want %q", c.format, c.args, got, c.want)
		}
	}
}

// TestSprintfErrors covers every malformed-format path: a bad verb, a trailing
// bare %, and a % followed only by flags then end-of-string.
func TestSprintfErrors(t *testing.T) {
	cases := []struct {
		format string
		want   string
	}{
		{"bad %z verb", "malformed format string - %z"},
		{"trailing %", "malformed format string - %"},
		{"flags then end %0", "malformed format string - %0"},
		{"%05", "malformed format string - %05"},
	}
	for _, c := range cases {
		_, err := Sprintf(c.format)
		if err == nil {
			t.Errorf("Sprintf(%q): expected error", c.format)
			continue
		}
		if err.Error() != c.want {
			t.Errorf("Sprintf(%q) error = %q, want %q", c.format, err.Error(), c.want)
		}
	}
}

// TestExpandErrno covers %m substitution, %%m escaping, a trailing lone %, and a
// non-m/non-% directive left untouched.
func TestExpandErrno(t *testing.T) {
	cases := []struct {
		in, errno, want string
	}{
		{"open: %m", "No such file or directory", "open: No such file or directory"},
		{"literal %%m here", "X", "literal %m here"},
		{"keep %s intact", "X", "keep %s intact"},
		{"trailing %", "X", "trailing %"},
		{"no directives", "X", "no directives"},
		{"%m and %m", "E", "E and E"},
	}
	for _, c := range cases {
		if got := ExpandErrno(c.in, c.errno); got != c.want {
			t.Errorf("ExpandErrno(%q,%q) = %q, want %q", c.in, c.errno, got, c.want)
		}
	}
}

// TestFormatMessage covers the three prefix shapes plus %m expansion and the
// error path from a malformed format.
func TestFormatMessage(t *testing.T) {
	cases := []struct {
		name   string
		ident  string
		pid    int
		opts   int
		format string
		args   []any
		errno  string
		want   string
	}{
		{"pid", "myapp", 4242, LOG_PID, "open failed: %m", nil, "No such file or directory",
			"myapp[4242]: open failed: No such file or directory"},
		{"ident no pid opt", "myapp", 4242, 0, "hello %s", []any{"world"}, "",
			"myapp: hello world"},
		{"pid opt but zero pid", "myapp", 0, LOG_PID, "x", nil, "", "myapp: x"},
		{"no ident", "", 4242, LOG_PID, "naked %d", []any{1}, "", "naked 1"},
	}
	for _, c := range cases {
		got, err := FormatMessage(c.ident, c.pid, c.opts, c.format, c.args, c.errno)
		if err != nil {
			t.Errorf("%s: FormatMessage error: %v", c.name, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s: FormatMessage = %q, want %q", c.name, got, c.want)
		}
	}

	if _, err := FormatMessage("id", 1, LOG_PID, "bad %z", nil, ""); err == nil {
		t.Error("FormatMessage: expected error on malformed format")
	}
}

// TestTagPID covers all three tag shapes.
func TestTagPID(t *testing.T) {
	cases := []struct {
		ident     string
		pid, opts int
		want      string
	}{
		{"app", 7, LOG_PID, "app[7]"},
		{"app", 7, 0, "app"},
		{"app", 0, LOG_PID, "app"},
		{"", 7, LOG_PID, ""},
	}
	for _, c := range cases {
		if got := TagPID(c.ident, c.pid, c.opts); got != c.want {
			t.Errorf("TagPID(%q,%d,%d) = %q, want %q", c.ident, c.pid, c.opts, got, c.want)
		}
	}
}

// TestFormatRFC3164 covers the framed line with and without a tag.
func TestFormatRFC3164(t *testing.T) {
	pri := Priority(LOG_LOCAL0, LOG_ERR) // 131
	got := FormatRFC3164(pri, "Jun 30 14:10:11", "host1", TagPID("app", 7, LOG_PID), "boom")
	want := "<131>Jun 30 14:10:11 host1 app[7]: boom"
	if got != want {
		t.Errorf("FormatRFC3164 = %q, want %q", got, want)
	}

	got = FormatRFC3164(pri, "Jun 30 14:10:11", "host1", "", "boom")
	want = "<131>Jun 30 14:10:11 host1 boom"
	if got != want {
		t.Errorf("FormatRFC3164 (no tag) = %q, want %q", got, want)
	}
}

// TestFormatRFC5424 covers full fields and NILVALUE substitution.
func TestFormatRFC5424(t *testing.T) {
	pri := Priority(LOG_LOCAL0, LOG_ERR) // 131
	got := FormatRFC5424(pri, "2026-06-30T14:10:11Z", "host1", "app", "4242", "ID7", `[ex@1 k="v"]`, "boom")
	want := `<131>1 2026-06-30T14:10:11Z host1 app 4242 ID7 [ex@1 k="v"] boom`
	if got != want {
		t.Errorf("FormatRFC5424 = %q, want %q", got, want)
	}

	got = FormatRFC5424(pri, "", "", "", "", "", "", "boom")
	want = "<131>1 - - - - - - boom"
	if got != want {
		t.Errorf("FormatRFC5424 (nilvalues) = %q, want %q", got, want)
	}
}
