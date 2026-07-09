# syslog examples

Runnable pure-Ruby usage of the `syslog` library, verified under the [rbgo](https://github.com/go-embedded-ruby/ruby) interpreter.

```sh
rbgo examples/syslog_usage.rb
```

| File | Shows |
| --- | --- |
| `syslog_usage.rb` | Feature detection of the `Syslog` module, the severity levels (`LOG_EMERG`..`LOG_DEBUG`), common facilities (`LOG_USER`, `LOG_DAEMON`, `LOG_LOCAL0`), and OR-combining open options (`LOG_PID \| LOG_CONS`). |
