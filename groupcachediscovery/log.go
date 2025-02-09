package groupcachediscovery

import (
	"fmt"
	"log/slog"
)

func infof(format string, a ...any) {
	slog.Info(fmt.Sprintf(format, a...))
}

func errorf(format string, a ...any) {
	slog.Error(fmt.Sprintf(format, a...))
}
