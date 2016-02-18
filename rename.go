package main

import (
	"regexp"
	"strings"
)

func renameRate(originalRate string) (name string) {
	switch originalRate {
	case "m1_rate":
		name = "1m"
	case "m5_rate":
		name = "5m"
	case "m15_rate":
		name = "15m"
	default:
		name = strings.TrimSuffix(originalRate, "_rate")
	}
	return
}

var (
	metric_symbol_repl_expr   = regexp.MustCompile(`[\.\$\-\(\)]`)
	chronos_jobs_capture_expr = regexp.MustCompile(`jobs\.run\.(\w+)\.([\w-]+)`)
)

func renameMetric(name string) (string, map[string]string) {
	labels := map[string]string{}

	captures := chronos_jobs_capture_expr.FindStringSubmatch(name)
	if len(captures) == 3 {
		name = "jobs_run_" + captures[1]
		labels["job"] = captures[2]
	}

	name = metric_symbol_repl_expr.ReplaceAllLiteralString(name, "_")
	name = strings.TrimRight(name, "_")
	name = strings.ToLower(name)
	return name, labels
}
