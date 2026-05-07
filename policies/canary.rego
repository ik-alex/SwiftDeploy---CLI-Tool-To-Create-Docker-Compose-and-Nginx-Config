package swiftdeploy.canary

import rego.v1

default allow := false

allow if {
	count(violations) == 0
}

violations contains msg if {
	input.error_rate_pct > data.thresholds.max_error_rate_pct
	msg := sprintf("Error rate %.2f%% exceeds maximum %.2f%%", [input.error_rate_pct, data.thresholds.max_error_rate_pct])
}

violations contains msg if {
	input.p99_latency_ms > data.thresholds.max_p99_latency_ms
	msg := sprintf("P99 latency %.0fms exceeds maximum %.0fms", [input.p99_latency_ms, data.thresholds.max_p99_latency_ms])
}

result := {
	"allow": allow,
	"violations": violations,
	"context": "pre-promote canary safety check",
	"checked_at": input.timestamp,
}
