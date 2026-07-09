# frozen_string_literal: true

require "syslog"

# Feature detection: code that logs conditionally first checks that the
# Syslog module is present before wiring itself to the system logger.
puts "Syslog available: #{defined?(Syslog) ? 'yes' : 'no'}"

# Severity levels, ordered from most to least urgent. Applications compare
# and threshold against these when deciding what to emit.
%i[LOG_EMERG LOG_ALERT LOG_CRIT LOG_ERR LOG_WARNING LOG_NOTICE LOG_INFO LOG_DEBUG].each do |level|
  puts format("%-12s = %d", level, Syslog.const_get(level))
end

# Facilities identify which subsystem a message came from.
puts "LOG_USER    = #{Syslog::LOG_USER}"
puts "LOG_DAEMON  = #{Syslog::LOG_DAEMON}"
puts "LOG_LOCAL0  = #{Syslog::LOG_LOCAL0}"

# Open options are bit flags that are OR'd together, e.g. log the PID and
# also write to the console on failure.
options = Syslog::LOG_PID | Syslog::LOG_CONS
puts "options (LOG_PID | LOG_CONS) = #{options}"
