// Package envutil is from https://github.com/reproducible-containers/repro-get/blob/v0.4.0/pkg/envutil/envutil.go
package envutil

import (
	"fmt"
	"os"
	"strconv"

	"log/slog"
)

func Bool(envName string, defaultValue bool) bool {
	v, ok := os.LookupEnv(envName)
	if !ok {
		return defaultValue
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		slog.Warn(fmt.Sprintf("Failed to parse %q ($%s) as a boolean: %v", v, envName, err))
		return defaultValue
	}
	return b
}
