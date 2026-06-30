package syslog

import (
	"fmt"
	"strconv"
	"strings"
)

// Sprintf builds a Syslog message body the way MRI's Syslog.log does: the
// message is Kernel#sprintf(format, args...) (syslog_write does
// str = rb_f_sprintf(argc, argv) in ext/syslog/syslog.c). When format carries
// no conversion directives and args is empty the format is used verbatim.
//
// Note on %m: MRI hands the result to syslog(pri, "%s", str), so the gem itself
// never substitutes %m — that is a libc artifact performed only when the line is
// written through a real syslog(3). A bare %m is therefore rejected here exactly
// as MRI's sprintf rejects it ("malformed format string - %m"); an escaped %%m
// survives as a literal %m. Use ExpandErrno (or FormatMessage) to perform the
// %m → errno-message substitution as a deterministic, host-supplied pure
// formatter.
//
// Sprintf maps Ruby's commonly-used syslog directives (%s, %d/%i, %u, %x, %X,
// %o, %f, %%) onto Go's fmt, which is byte-identical for these. Unknown
// directives are reported as an error, mirroring Ruby's ArgumentError on a
// malformed format string (e.g. a bare %m in a multi-arg call).
func Sprintf(format string, args ...any) (string, error) {
	gf, err := translate(format)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(gf, args...), nil
}

// translate rewrites a Ruby sprintf format into the equivalent Go fmt format for
// the directive subset syslog messages use, validating each directive. It is the
// single place that decides which conversions are accepted.
func translate(format string) (string, error) {
	var b strings.Builder
	b.Grow(len(format))
	for i := 0; i < len(format); i++ {
		c := format[i]
		if c != '%' {
			b.WriteByte(c)
			continue
		}
		// Consume a directive: '%' then optional flags/width/precision, then a
		// verb. We only need to recognise the verb to validate and to remap.
		j := i + 1
		if j >= len(format) {
			return "", fmt.Errorf("malformed format string - %%")
		}
		if format[j] == '%' { // literal percent
			b.WriteString("%%")
			i = j
			continue
		}
		// Copy flags/width/precision verbatim (shared between Ruby and Go).
		start := j
		for j < len(format) && strings.IndexByte("-+ 0#.*0123456789", format[j]) >= 0 {
			j++
		}
		if j >= len(format) {
			return "", fmt.Errorf("malformed format string - %%%s", format[i+1:])
		}
		flags := format[start:j]
		verb := format[j]
		goVerb, ok := mapVerb(verb)
		if !ok {
			return "", fmt.Errorf("malformed format string - %%%c", verb)
		}
		b.WriteByte('%')
		b.WriteString(flags)
		b.WriteString(goVerb)
		i = j
	}
	return b.String(), nil
}

// mapVerb maps a Ruby sprintf verb to the Go fmt verb that produces identical
// output for syslog's purposes. The bool is false for an unsupported verb.
func mapVerb(verb byte) (string, bool) {
	switch verb {
	case 's':
		return "s", true
	case 'd', 'i':
		return "d", true
	case 'u':
		return "d", true // Ruby %u on a non-negative int == %d
	case 'x':
		return "x", true
	case 'X':
		return "X", true
	case 'o':
		return "o", true
	case 'b':
		return "b", true
	case 'f':
		return "f", true
	case 'e':
		return "e", true
	case 'g':
		return "g", true
	case 'c':
		return "c", true
	case 'p':
		return "v", true // Ruby %p == inspect; %v is the closest Go analogue
	default:
		return "", false
	}
}

// ExpandErrno performs the %m → errno-message substitution that a real libc
// syslog(3) applies host-side, but as a pure, deterministic formatter: it
// replaces every %m in s with errnoMsg (the platform's strerror(errno) string,
// supplied by the host) and collapses %%m to a literal %m, exactly as libc does.
// Other %-sequences are left untouched.
//
// This is intentionally separate from Sprintf: the gem itself never expands %m
// (it passes "%s" to the C call), so reproducing the host behaviour requires the
// caller to supply errno's message; that keeps this package free of any
// platform errno table and deterministic across OSes and arches.
func ExpandErrno(s, errnoMsg string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != '%' || i+1 >= len(s) {
			b.WriteByte(s[i])
			continue
		}
		switch s[i+1] {
		case 'm':
			b.WriteString(errnoMsg)
		case '%':
			b.WriteByte('%')
		default:
			b.WriteByte('%')
			b.WriteByte(s[i+1])
		}
		i++
	}
	return b.String()
}

// FormatMessage produces the full message body the syslog facility emits for one
// MRI Syslog.log call, including the ident[pid]: prefix it prepends. It is the
// one-call composite of the whole pipeline: the host-supplied %m expansion
// (errnoMsg comes from the caller's strerror(errno), the substitution libc does
// when it writes the line), then the Sprintf(format, args...) the gem performs,
// then the line shaping MRI/libc applies:
//
//	ident[pid]: message   when opts has LOG_PID set (and pid != 0)
//	ident: message        otherwise, when ident is non-empty
//	message               when ident is empty
//
// %m is expanded before sprintf (as libc resolves it independently of the gem's
// sprintf pass), so a bare %m is accepted here even though Sprintf alone rejects
// it; the substituted errno text is percent-escaped so a "%" inside it survives
// the sprintf pass literally. errnoMsg may be "" when the message has no %m. opts
// is the openlog option bitmask (only LOG_PID is consulted here; the rest are
// host-side).
func FormatMessage(ident string, pid int, opts int, format string, args []any, errnoMsg string) (string, error) {
	format = ExpandErrno(format, strings.ReplaceAll(errnoMsg, "%", "%%"))
	msg, err := Sprintf(format, args...)
	if err != nil {
		return "", err
	}
	return shapeLine(ident, pid, opts, msg), nil
}

// shapeLine prepends the ident[pid]: / ident: prefix to a message body.
func shapeLine(ident string, pid, opts int, msg string) string {
	if ident == "" {
		return msg
	}
	if opts&LOG_PID != 0 && pid != 0 {
		return ident + "[" + strconv.Itoa(pid) + "]: " + msg
	}
	return ident + ": " + msg
}

// FormatRFC3164 frames a message as a BSD syslog (RFC 3164) line:
//
//	<PRI>TIMESTAMP HOST TAG: MSG
//
// PRI is Priority(facility, severity); timestamp ("Mmm dd hh:mm:ss") and host
// are supplied by the caller (this package performs no clock or hostname lookup
// — those are host-side). tag is the ident[pid] tag (build it with TagPID); msg
// is the already-formatted body (e.g. from Sprintf/ExpandErrno). The tag and its
// colon are omitted when tag is empty.
func FormatRFC3164(priority int, timestamp, host, tag, msg string) string {
	pri := "<" + strconv.Itoa(priority) + ">"
	if tag == "" {
		return pri + timestamp + " " + host + " " + msg
	}
	return pri + timestamp + " " + host + " " + tag + ": " + msg
}

// FormatRFC5424 frames a message as an IETF syslog (RFC 5424) line:
//
//	<PRI>VERSION TIMESTAMP HOST APPNAME PROCID MSGID STRUCTURED-DATA MSG
//
// VERSION is fixed at 1. timestamp (RFC 3339), host, appName, procID, msgID and
// structuredData are caller-supplied (host-side); any empty field is emitted as
// the NILVALUE "-", per the RFC. msg is the formatted body.
func FormatRFC5424(priority int, timestamp, host, appName, procID, msgID, structuredData, msg string) string {
	nv := func(s string) string {
		if s == "" {
			return "-"
		}
		return s
	}
	return "<" + strconv.Itoa(priority) + ">1 " +
		nv(timestamp) + " " + nv(host) + " " + nv(appName) + " " +
		nv(procID) + " " + nv(msgID) + " " + nv(structuredData) + " " + msg
}

// TagPID builds the "ident[pid]" tag (or "ident" when opts lacks LOG_PID or pid
// is 0) used as the RFC 3164 TAG field, matching how the syslog facility labels
// the line.
func TagPID(ident string, pid, opts int) string {
	if ident == "" {
		return ""
	}
	if opts&LOG_PID != 0 && pid != 0 {
		return ident + "[" + strconv.Itoa(pid) + "]"
	}
	return ident
}
