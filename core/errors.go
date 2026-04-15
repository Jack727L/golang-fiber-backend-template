package core

import (
	"fmt"
	"path/filepath"
	"runtime"
	"time"

	"github.com/yourusername/go-api-starter/env"
	"github.com/gofiber/fiber/v2"
)

// LogError logs an error to stdout with file/function/line context.
// Pass c=nil for background jobs.
func LogError(c *fiber.Ctx, err error, customMessage ...string) {
	if env.IsTestMode() && c != nil {
		return
	}
	if err == nil {
		return
	}

	pc, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "unknown"
		line = 0
	}
	file = filepath.Base(file)

	funcName := "unknown"
	if fn := runtime.FuncForPC(pc); fn != nil {
		funcName = filepath.Base(fn.Name())
	}

	message := fmt.Sprintf("Error in %s:%d (%s): %v", file, line, funcName, err)
	if len(customMessage) > 0 && customMessage[0] != "" {
		message = fmt.Sprintf("%s - %s", customMessage[0], message)
	}

	fmt.Printf("ERROR %s %s\n", time.Now().Format(time.RFC3339), message)
}

// LogInfo logs an info message to stdout (skipped in test mode for HTTP handlers).
func LogInfo(c *fiber.Ctx, message string, args ...interface{}) {
	if env.IsTestMode() && c != nil {
		return
	}
	formatted := message
	if len(args) > 0 {
		formatted = fmt.Sprintf(message, args...)
	}
	fmt.Printf("INFO %s %s\n", time.Now().Format(time.RFC3339), formatted)
}

// LogDebug logs a debug message (only in non-test, non-prod environments).
func LogDebug(c *fiber.Ctx, message string, args ...interface{}) {
	if env.IsTestMode() || env.IsProdMode() {
		return
	}
	formatted := message
	if len(args) > 0 {
		formatted = fmt.Sprintf(message, args...)
	}
	fmt.Printf("DEBUG %s %s\n", time.Now().Format(time.RFC3339), formatted)
}

// SetError stores error context on the Fiber request locals (used by LoggingMiddleware).
func SetError(c *fiber.Ctx, err error) {
	if c == nil || err == nil {
		return
	}
	pc, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "unknown"
		line = 0
	}
	file = filepath.Base(file)
	funcName := "unknown"
	if fn := runtime.FuncForPC(pc); fn != nil {
		funcName = filepath.Base(fn.Name())
	}
	c.Locals("_err", err.Error())
	c.Locals("_err_file", file)
	c.Locals("_err_line", line)
	c.Locals("_err_func", funcName)
}

// ErrorResponse returns a JSON error response and records caller context for the logging middleware.
func ErrorResponse(c *fiber.Ctx, status int, message string, err ...error) error {
	if len(err) > 0 && err[0] != nil {
		SetError(c, err[0])
	}
	return c.Status(status).JSON(fiber.Map{"error": message})
}
