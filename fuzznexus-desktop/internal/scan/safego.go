package scan

import (
	"fmt"
	"log/slog"
	"regexp"
	"runtime"
)

var (
	bearerRegex    = regexp.MustCompile(`(?i)(bearer\s+)[A-Za-z0-9_\-\.]+`)
	cookieRegex    = regexp.MustCompile(`(?i)(cookie:\s*)[^\r\n]+`)
	setCookieRegex = regexp.MustCompile(`(?i)(set-cookie:\s*)[^\r\n]+`)
	authRegex      = regexp.MustCompile(`(?i)(authorization:\s*)[^\r\n]+`)
)

// MaskSensitiveData redacts authentication headers, cookies, and tokens from log messages or stack traces.
func MaskSensitiveData(s string) string {
	s = bearerRegex.ReplaceAllString(s, "${1}***")
	s = cookieRegex.ReplaceAllString(s, "${1}***")
	s = setCookieRegex.ReplaceAllString(s, "${1}***")
	s = authRegex.ReplaceAllString(s, "${1}***")
	return s
}

// SafeGo runs a function in a new goroutine with panic recovery, sanitized stack logging, and an optional panic callback.
func SafeGo(logger *slog.Logger, name string, onPanic func(err error), fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				buf := make([]byte, 8192)
				n := runtime.Stack(buf, false)
				rawStack := string(buf[:n])
				maskedStack := MaskSensitiveData(rawStack)

				err := fmt.Errorf("panic in goroutine [%s]: %v", name, r)
				if logger != nil {
					logger.Error("Recovered from panic in background task",
						"task", name,
						"panic", fmt.Sprintf("%v", r),
					)
					logger.Debug("Panic stack trace",
						"task", name,
						"stack", maskedStack,
					)
				}

				if onPanic != nil {
					onPanic(err)
				}
			}
		}()
		fn()
	}()
}
