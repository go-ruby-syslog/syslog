// Package syslog is a pure-Go (CGO=0) reimplementation of the message
// FORMATTING and priority-encoding half of Ruby's [Syslog] module — MRI
// 4.0.5's syslog-0.3.0 gem — without any Ruby runtime or libc.
//
// What it is — and isn't. The actual syslog facility (openlog(3) / syslog(3),
// the connection to /dev/log or a UDP collector, and the libc %m expansion the
// platform performs when it writes the line) is host-side and is NOT in scope.
// What lives here is the pure-compute core that needs no syscall: the facility,
// severity and option constants with MRI's exact integer values; the priority
// encoding pri = facility | severity (the RFC <PRI> value = facility/8*8 +
// severity, i.e. facility code * 8 + severity); the LOG_MASK / LOG_UPTO mask
// macros; and the message shaping MRI's Syslog.log(level, fmt, *args) performs
// before it hands the bytes to the C syslog() call.
//
// MRI's Syslog.log builds its message as Kernel#sprintf(fmt, *args) (see
// syslog_write in ext/syslog/syslog.c: str = rb_f_sprintf(argc, argv)) and then
// calls syslog(pri, "%s", str). Because the C format is the constant "%s", the
// gem itself does NOT expand %m — the %m → strerror(errno) substitution is a
// libc artifact performed host-side when (and only when) the line is written
// through a real syslog(3). This package therefore offers Sprintf as the
// faithful gem-level formatter and a separate, explicit ExpandErrno that
// performs the %m → errno-string substitution as a pure formatter taking errno
// as an input — so a host can reproduce the libc behaviour deterministically.
//
// [Syslog]: https://docs.ruby-lang.org/en/master/Syslog.html
package syslog

// Facility codes (Syslog::LOG_*). The value is the facility number shifted left
// by three bits (facility code * 8), so it can be OR'd with a severity to form a
// priority. These are MRI 4.0.5's values, verified against `ruby -rsyslog`.
const (
	LOG_KERN     = 0 << 3  // 0   — kernel messages
	LOG_USER     = 1 << 3  // 8   — random user-level messages
	LOG_MAIL     = 2 << 3  // 16  — mail system
	LOG_DAEMON   = 3 << 3  // 24  — system daemons
	LOG_AUTH     = 4 << 3  // 32  — security/authorization messages
	LOG_SYSLOG   = 5 << 3  // 40  — messages generated internally by syslogd
	LOG_LPR      = 6 << 3  // 48  — line printer subsystem
	LOG_NEWS     = 7 << 3  // 56  — network news subsystem
	LOG_UUCP     = 8 << 3  // 64  — UUCP subsystem
	LOG_CRON     = 9 << 3  // 72  — clock daemon
	LOG_AUTHPRIV = 10 << 3 // 80  — security/authorization (private)
	LOG_FTP      = 11 << 3 // 88  — ftp daemon
	LOG_LOCAL0   = 16 << 3 // 128 — reserved for local use
	LOG_LOCAL1   = 17 << 3 // 136
	LOG_LOCAL2   = 18 << 3 // 144
	LOG_LOCAL3   = 19 << 3 // 152
	LOG_LOCAL4   = 20 << 3 // 160
	LOG_LOCAL5   = 21 << 3 // 168
	LOG_LOCAL6   = 22 << 3 // 176
	LOG_LOCAL7   = 23 << 3 // 184
)

// Severity / priority levels (Syslog::LOG_*), most-severe first. MRI 4.0.5
// values, verified against `ruby -rsyslog`.
const (
	LOG_EMERG   = 0 // system is unusable
	LOG_ALERT   = 1 // action must be taken immediately
	LOG_CRIT    = 2 // critical conditions
	LOG_ERR     = 3 // error conditions
	LOG_WARNING = 4 // warning conditions
	LOG_NOTICE  = 5 // normal but significant condition
	LOG_INFO    = 6 // informational
	LOG_DEBUG   = 7 // debug-level messages
)

// Option flags for openlog (Syslog::LOG_*). MRI 4.0.5 values, verified against
// `ruby -rsyslog`. They govern host-side behaviour (whether to log the pid,
// write to the console on failure, open the connection eagerly, etc.); this
// package exposes their values and formats LOG_PID into the line, but the
// connection itself is host-side.
const (
	LOG_PID    = 0x01 // 1  — log the process id with each message
	LOG_CONS   = 0x02 // 2  — log on the console if errors sending
	LOG_ODELAY = 0x04 // 4  — delay open until first syslog() (default)
	LOG_NDELAY = 0x08 // 8  — open the connection immediately
	LOG_NOWAIT = 0x10 // 16 — do not wait for console forks (deprecated)
	LOG_PERROR = 0x20 // 32 — log to stderr as well
)

// LOG_PRIMASK is the mask for the severity bits of a priority value: the low
// three bits. priority & LOG_PRIMASK extracts the severity; priority &^
// LOG_PRIMASK extracts the facility.
const LOG_PRIMASK = 0x07

// Priority encodes (facility | severity) into a single priority value, exactly
// as MRI's Syslog.log composes its first syslog(3) argument and as RFC 3164 /
// RFC 5424 define the <PRI> field: PRI = facility-code*8 + severity. Because the
// facility constants here are already pre-shifted (facility code * 8), this is a
// bitwise OR of facility and severity.
func Priority(facility, severity int) int {
	return facility | severity
}

// Facility returns the facility component of a priority value (the bits above
// the low three): priority &^ LOG_PRIMASK.
func Facility(priority int) int {
	return priority &^ LOG_PRIMASK
}

// Severity returns the severity component of a priority value: the low three
// bits, priority & LOG_PRIMASK.
func Severity(priority int) int {
	return priority & LOG_PRIMASK
}

// LogMask returns the log mask for a single priority level — the
// Syslog::LOG_MASK(pri) macro: 1 << pri. It selects exactly that one level in a
// priority mask.
func LogMask(pri int) int {
	return 1 << uint(pri)
}

// LogUpto returns the log mask of all priorities up to and including pri — the
// Syslog::LOG_UPTO(pri) macro: (1 << (pri+1)) - 1. For example LogUpto(LOG_ERR)
// selects EMERG..ERR.
func LogUpto(pri int) int {
	return (1 << uint(pri+1)) - 1
}
