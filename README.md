<p align="center"><img src="https://raw.githubusercontent.com/go-ruby-syslog/brand/main/social/go-ruby-syslog-syslog.png" alt="go-ruby-syslog/syslog" width="720"></p>

# syslog — go-ruby-syslog

[![Docs](https://img.shields.io/badge/docs-mkdocs--material-DC2626)](https://go-ruby-syslog.github.io/docs/)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.26.4%2B-00ADD8)](https://go.dev/dl/)
[![Coverage](https://img.shields.io/badge/coverage-100%25-1a7f37)](#tests--coverage)

**A pure-Go (no cgo) reimplementation of the message-FORMATTING and
priority-encoding half of Ruby's [Syslog](https://docs.ruby-lang.org/en/master/Syslog.html)
module** — MRI 4.0.5's `syslog-0.3.0` gem — **without any Ruby runtime, libc, or
syscall**. It ships the facility/severity/option constants with MRI's exact
integer values, the `pri = facility | severity` priority encoding (the RFC
`<PRI>` value), the `LOG_MASK` / `LOG_UPTO` mask macros, and the `ident[pid]:
message` line shaping `Syslog.log` produces — plus optional RFC 3164 / RFC 5424
framers as pure formatters.

It is the Syslog backend for
[go-embedded-ruby](https://github.com/go-embedded-ruby/ruby), but is a
**standalone, reusable** module — a sibling of
[go-ruby-marshal](https://github.com/go-ruby-marshal/marshal),
[go-ruby-pstore](https://github.com/go-ruby-pstore/pstore),
[go-ruby-yaml](https://github.com/go-ruby-yaml/yaml), and
[go-ruby-regexp](https://github.com/go-ruby-regexp/regexp).

> **What it is — and isn't.** The constant tables, the priority math, the mask
> macros, and the message shaping are pure compute and need **no interpreter and
> no syscall**, so they live here as pure Go. The *facility* half — `openlog(3)`,
> the connection to `/dev/log` or a UDP collector, the actual `syslog(3)` write,
> and the platform's libc `%m` → `strerror(errno)` expansion it performs when it
> writes the line — is the **host's job** and is **not in scope**. This package
> reproduces the libc `%m` substitution deterministically via `ExpandErrno`,
> taking the errno message as an input, so a host can format the exact bytes
> off-line.

## The `%m` truth (verified against MRI 4.0.5)

MRI's `Syslog.log(level, fmt, *args)` builds its message as
`Kernel#sprintf(fmt, *args)` (`syslog_write` does `str = rb_f_sprintf(argc, argv)`
in `ext/syslog/syslog.c`) and then calls `syslog(pri, "%s", str)`. Because the C
format string is the constant `"%s"`, **the gem itself never expands `%m`** — that
substitution is a libc artifact performed only when the line is written through a
real `syslog(3)`. Consequently a bare `%m` in a `sprintf` call raises
`ArgumentError: malformed format string - %m`, and an escaped `%%m` reaches the
output as a literal `%m`. This library mirrors that exactly:

- **`Sprintf`** is the gem-faithful formatter — it rejects a bare `%m` and passes
  `%%m` through as `%m`.
- **`ExpandErrno`** performs the host-side `%m` → errno-message substitution as a
  pure, deterministic formatter (you supply `strerror(errno)`), so it carries no
  platform errno table and is byte-identical across OSes and arches.
- **`FormatMessage`** is the one-call composite: it expands `%m` (as libc does,
  independently of the gem's sprintf pass), runs `sprintf`, and shapes the
  `ident[pid]: message` line.

## Install

```sh
go get github.com/go-ruby-syslog/syslog
```

## Usage

```go
package main

import (
	"fmt"

	"github.com/go-ruby-syslog/syslog"
)

func main() {
	// Priority encoding: pri = facility | severity (the RFC <PRI> value).
	pri := syslog.Priority(syslog.LOG_LOCAL0, syslog.LOG_ERR) // 131
	fmt.Println(pri, syslog.Facility(pri), syslog.Severity(pri))

	// Mask macros.
	fmt.Println(syslog.LogMask(syslog.LOG_ERR)) // 8
	fmt.Println(syslog.LogUpto(syslog.LOG_ERR)) // 15 (EMERG..ERR)

	// One-call message build: %m → errno (host-supplied), sprintf, ident[pid]:.
	line, _ := syslog.FormatMessage(
		"myapp", 4242, syslog.LOG_PID,
		"open failed: %m for %s", []any{"/etc/x"},
		"No such file or directory", // strerror(errno), supplied by the host
	)
	fmt.Println(line) // myapp[4242]: open failed: No such file or directory for /etc/x

	// RFC 3164 / RFC 5424 framing (timestamp + host are caller-supplied).
	tag := syslog.TagPID("myapp", 4242, syslog.LOG_PID)
	fmt.Println(syslog.FormatRFC3164(pri, "Jun 30 14:10:11", "host1", tag, "boom"))
	fmt.Println(syslog.FormatRFC5424(pri, "2026-06-30T14:10:11Z", "host1", "myapp", "4242", "-", "-", "boom"))
}
```

## API

```go
// Constants (MRI Syslog::LOG_* values):
//   facilities: LOG_KERN LOG_USER LOG_MAIL LOG_DAEMON LOG_AUTH LOG_SYSLOG
//               LOG_LPR LOG_NEWS LOG_UUCP LOG_CRON LOG_AUTHPRIV LOG_FTP
//               LOG_LOCAL0..LOG_LOCAL7
//   severities: LOG_EMERG LOG_ALERT LOG_CRIT LOG_ERR LOG_WARNING LOG_NOTICE
//               LOG_INFO LOG_DEBUG
//   options:    LOG_PID LOG_CONS LOG_ODELAY LOG_NDELAY LOG_NOWAIT LOG_PERROR
//   LOG_PRIMASK (severity-bit mask)

// Priority math.
func Priority(facility, severity int) int // pri = facility | severity
func Facility(priority int) int           // priority &^ LOG_PRIMASK
func Severity(priority int) int           // priority & LOG_PRIMASK
func LogMask(pri int) int                 // Syslog::LOG_MASK: 1 << pri
func LogUpto(pri int) int                 // Syslog::LOG_UPTO: (1 << (pri+1)) - 1

// Message formatting.
func Sprintf(format string, args ...any) (string, error)       // gem-faithful (rejects bare %m)
func ExpandErrno(s, errnoMsg string) string                    // host-side %m → errno (pure)
func FormatMessage(ident string, pid, opts int, format string, // %m + sprintf + ident[pid]:
	args []any, errnoMsg string) (string, error)
func TagPID(ident string, pid, opts int) string                // ident[pid] (or ident)

// RFC framing (pure formatters; timestamp/host are inputs).
func FormatRFC3164(priority int, timestamp, host, tag, msg string) string
func FormatRFC5424(priority int, timestamp, host, appName, procID, msgID,
	structuredData, msg string) string
```

The socket connection (`openlog`, the `syslog(3)` send to `/dev/log` or a UDP
collector, the `mask=`/`open`/`close` lifecycle, and `LOG_CONS`/`LOG_PERROR`
side-effects) is **host-side** and intentionally absent. rbgo wires those to the
real OS; this module is the deterministic, Ruby-free, syscall-free formatter the
rest of the go-embedded-ruby stack shares.

## Tests & coverage

The suite pairs deterministic, ruby-free tests over the pure
constant/priority/format core (which alone hold coverage at 100%, so the qemu
cross-arch and Windows lanes pass the gate) with a **differential MRI oracle**: it
runs the real `ruby -rsyslog` and asserts every `Syslog::LOG_*` constant, the
`facility | severity` composition, the `LOG_MASK` / `LOG_UPTO` macros, and a set
of `Kernel#sprintf` formats all match this package byte-for-byte. The oracle
`$stdout.binmode`s itself so Windows text-mode never perturbs the comparison, and
skips itself on Windows (no syslog gem) and wherever `ruby` is absent.

```sh
COVERPKG=$(go list ./... | paste -sd, -)
go test -race -coverpkg="$COVERPKG" -coverprofile=cover.out ./...
go tool cover -func=cover.out | tail -1   # 100.0%
```

CGO-free, **100% test coverage**, `gofmt` + `go vet` clean, and green across the
six 64-bit Go targets (amd64, arm64, riscv64, loong64, ppc64le, s390x) and three
OSes (Linux, macOS, Windows).

## License

BSD-3-Clause — see [LICENSE](LICENSE). Copyright the go-ruby-syslog/syslog authors.
