package main

import (
	"os"
	"strconv"
	"time"
)

// envString extracts string from env var.
// It returns the provided defaultValue if the env var is empty.
// The value returned is also recorded in logs.
func envString(name string, defaultValue string) string {
	str := os.Getenv(name)
	if str != "" {
		infof("%s=[%s] using %s=%s default=%s", name, str, name, str, defaultValue)
		return str
	}
	infof("%s=[%s] using %s=%s default=%s", name, str, name, defaultValue, defaultValue)
	return defaultValue
}

// envBool extracts duration from env var.
// It returns the provided defaultValue if the env var is empty.
// The value returned is also recorded in logs.
func envDuration(name string, defaultValue time.Duration) time.Duration {
	str := os.Getenv(name)
	if str != "" {
		value, errConv := time.ParseDuration(str)
		if errConv == nil {
			infof("%s=[%s] using %s=%v default=%v", name, str, name, value, defaultValue)
			return value
		}
		errorf("bad %s=[%s]: error: %v", name, str, errConv)
	}
	infof("%s=[%s] using %s=%v default=%v", name, str, name, defaultValue, defaultValue)
	return defaultValue
}

// envBool extracts bool from env var.
// It returns the provided defaultValue if the env var is empty.
// The value returned is also recorded in logs.
func envBool(name string, defaultValue bool) bool {
	str := os.Getenv(name)
	if str != "" {
		value, errConv := strconv.ParseBool(str)
		if errConv == nil {
			infof("%s=[%s] using %s=%t default=%t", name, str, name, value, defaultValue)
			return value
		}
		errorf("bad %s=[%s]: error: %v", name, str, errConv)
	}
	infof("%s=[%s] using %s=%t default=%t", name, str, name, defaultValue, defaultValue)
	return defaultValue
}

// envInt64 extracts int64 value from env var.
// It returns the provided defaultValue if the env var is empty.
// The value returned is also recorded in logs.
func envInt64(name string, defaultValue int64) int64 {
	str := os.Getenv(name)
	if str != "" {
		value, errConv := strconv.ParseInt(name, 10, 64)
		if errConv == nil {
			infof("%s=[%s] using %s=%v default=%v", name, str, name, value, defaultValue)
			return value
		}
		errorf("bad %s=[%s]: error: %v", name, str, errConv)
	}
	infof("%s=[%s] using %s=%v default=%v", name, str, name, defaultValue, defaultValue)
	return defaultValue
}
