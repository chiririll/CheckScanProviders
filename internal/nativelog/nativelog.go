package nativelog

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"unicode/utf8"
)

const (
	prioDebug = 3
	prioInfo  = 4
	prioWarn  = 5
	prioError = 6
)

type callKey struct{}

var seq atomic.Uint64

func Next() string {
	return fmt.Sprintf("n%d", seq.Add(1))
}

func WithCall(ctx context.Context, id string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, callKey{}, id)
}

func Call(ctx context.Context) string {
	if ctx == nil {
		return "-"
	}
	if id, ok := ctx.Value(callKey{}).(string); ok && id != "" {
		return id
	}
	return "-"
}

func Debug(format string, args ...any) { logf(prioDebug, format, args...) }
func Info(format string, args ...any)  { logf(prioInfo, format, args...) }
func Warn(format string, args ...any)  { logf(prioWarn, format, args...) }
func Error(format string, args ...any) { logf(prioError, format, args...) }

func Preview(s string, max int) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	if max <= 0 {
		max = 96
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max]) + fmt.Sprintf("…(%d)", len(runes))
}

func logf(prio int, format string, args ...any) {
	write(prio, fmt.Sprintf(format, args...))
}
