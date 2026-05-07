package swiftdeploy.infra

import rego.v1

default allow := false

allow if {
	count(violations) == 0
}

violations contains msg if {
	input.disk_free_gb < data.thresholds.min_disk_free_gb
	msg := sprintf("Disk free space %.1fGB is below minimum %.1fGB", [input.disk_free_gb, data.thresholds.min_disk_free_gb])
}

violations contains msg if {
	input.cpu_load > data.thresholds.max_cpu_load
	msg := sprintf("CPU load %.2f exceeds maximum %.2f", [input.cpu_load, data.thresholds.max_cpu_load])
}

result := {
	"allow": allow,
	"violations": violations,
	"context": "pre-deploy infrastructure check",
	"checked_at": input.timestamp,
}
