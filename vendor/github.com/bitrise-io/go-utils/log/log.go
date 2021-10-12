package log

import (
	"fmt"
	"io"
	"os"
	"time"
)

// Logger ...
type Logger interface {
	Infof(format string, v ...interface{})
	Warnf(format string, v ...interface{})
	Printf(format string, v ...interface{})
	Donef(format string, v ...interface{})
	Debugf(format string, v ...interface{})
	Errorf(format string, v ...interface{})
	TInfof(format string, v ...interface{})
	TWarnf(format string, v ...interface{})
	TPrintf(format string, v ...interface{})
	TDonef(format string, v ...interface{})
	TDebugf(format string, v ...interface{})
	TErrorf(format string, v ...interface{})
	Println()
	EnableDebugLog(enable bool)
}

const defaultTimeStampLayout = "15:04:05"

type defaultLogger struct {
	enableDebugLog  bool
	timestampLayout string
	stdout          io.Writer
}

// NewLogger ...
func NewLogger() Logger {
	return &defaultLogger{enableDebugLog: false, timestampLayout: defaultTimeStampLayout, stdout: os.Stdout}
}

// EnableDebugLog ...
func (l *defaultLogger) EnableDebugLog(enable bool) {
	l.enableDebugLog = enable
}

// Infof ...
func (l *defaultLogger) Infof(format string, v ...interface{}) {
	l.printf(infoSeverity, false, format, v...)
}

// Warnf ...
func (l *defaultLogger) Warnf(format string, v ...interface{}) {
	l.printf(warnSeverity, false, format, v...)
}

// Printf ...
func (l *defaultLogger) Printf(format string, v ...interface{}) {
	l.printf(normalSeverity, false, format, v...)
}

// Donef ...
func (l *defaultLogger) Donef(format string, v ...interface{}) {
	l.printf(doneSeverity, false, format, v...)
}

// Debugf ...
func (l *defaultLogger) Debugf(format string, v ...interface{}) {
	if l.enableDebugLog {
		l.printf(debugSeverity, false, format, v...)
	}
}

// Errorf ...
func (l *defaultLogger) Errorf(format string, v ...interface{}) {
	l.printf(errorSeverity, false, format, v...)
}

// TInfof ...
func (l *defaultLogger) TInfof(format string, v ...interface{}) {
	l.printf(infoSeverity, true, format, v...)
}

// TWarnf ...
func (l *defaultLogger) TWarnf(format string, v ...interface{}) {
	l.printf(warnSeverity, true, format, v...)
}

// TPrintf ...
func (l *defaultLogger) TPrintf(format string, v ...interface{}) {
	l.printf(normalSeverity, true, format, v...)
}

// TDonef ...
func (l *defaultLogger) TDonef(format string, v ...interface{}) {
	l.printf(doneSeverity, true, format, v...)
}

// TDebugf ...
func (l *defaultLogger) TDebugf(format string, v ...interface{}) {
	if l.enableDebugLog {
		l.printf(debugSeverity, true, format, v...)
	}
}

// TErrorf ...
func (l *defaultLogger) TErrorf(format string, v ...interface{}) {
	l.printf(errorSeverity, true, format, v...)
}

// Println ...
func (l *defaultLogger) Println() {
	fmt.Println()
}

func (l *defaultLogger) timestampField() string {
	currentTime := time.Now()
	return fmt.Sprintf("[%s]", currentTime.Format(l.timestampLayout))
}

func (l *defaultLogger) prefixCurrentTime(message string) string {
	return fmt.Sprintf("%s %s", l.timestampField(), message)
}

func (l *defaultLogger) createLogMsg(severity Severity, withTime bool, format string, v ...interface{}) string {
	colorFunc := severityColorFuncMap[severity]
	message := colorFunc(format, v...)
	if withTime {
		message = l.prefixCurrentTime(message)
	}

	return message
}

func (l *defaultLogger) printf(severity Severity, withTime bool, format string, v ...interface{}) {
	message := l.createLogMsg(severity, withTime, format, v...)
	if _, err := fmt.Fprintln(l.stdout, message); err != nil {
		fmt.Printf("failed to print message: %s, error: %s\n", message, err)
	}
}

// RInfof ...
func RInfof(stepID string, tag string, data map[string]interface{}, format string, v ...interface{}) {
	rprintf("info", stepID, tag, data, format, v...)
}

// RWarnf ...
func RWarnf(stepID string, tag string, data map[string]interface{}, format string, v ...interface{}) {
	rprintf("warn", stepID, tag, data, format, v...)
}

// RErrorf ...
func RErrorf(stepID string, tag string, data map[string]interface{}, format string, v ...interface{}) {
	rprintf("error", stepID, tag, data, format, v...)
}

var deprecatedLogger = defaultLogger{stdout: os.Stdout, enableDebugLog: false, timestampLayout: defaultTimeStampLayout}

// SetEnableDebugLog ...
// Deprecated: use Logger instead.
func SetEnableDebugLog(enable bool) {
	deprecatedLogger.enableDebugLog = enable
}

// SetTimestampLayout ...
// Deprecated: use Logger instead.
func SetTimestampLayout(layout string) {
	deprecatedLogger.timestampLayout = layout
}

// SetOutWriter ...
// Deprecated: use Logger for verification instead.
func SetOutWriter(writer io.Writer) {
	deprecatedLogger.stdout = writer
}

// Donef ...
// Deprecated: use Logger instead.
func Donef(format string, v ...interface{}) {
	deprecatedLogger.Donef(format, v...)
}

// Infof ...
// Deprecated: use Logger instead.
func Infof(format string, v ...interface{}) {
	deprecatedLogger.Infof(format, v...)
}

// Printf ...
// Deprecated: use Logger instead.
func Printf(format string, v ...interface{}) {
	deprecatedLogger.Printf(format, v...)
}

// Debugf ...
// Deprecated: use Logger instead.
func Debugf(format string, v ...interface{}) {
	deprecatedLogger.Debugf(format, v...)
}

// Warnf ...
// Deprecated: use Logger instead.
func Warnf(format string, v ...interface{}) {
	deprecatedLogger.Warnf(format, v...)
}

// Errorf ...
// Deprecated: use Logger instead.
func Errorf(format string, v ...interface{}) {
	deprecatedLogger.Errorf(format, v...)
}

// TDonef ...
// Deprecated: use Logger instead.
func TDonef(format string, v ...interface{}) {
	deprecatedLogger.TDonef(format, v...)
}

// TInfof ...
// Deprecated: use Logger instead.
func TInfof(format string, v ...interface{}) {
	deprecatedLogger.TInfof(format, v...)
}

// TPrintf ...
// Deprecated: use Logger instead.
func TPrintf(format string, v ...interface{}) {
	deprecatedLogger.TPrintf(format, v...)
}

// TDebugf ...
// Deprecated: use Logger instead.
func TDebugf(format string, v ...interface{}) {
	deprecatedLogger.TDebugf(format, v...)
}

// TWarnf ...
// Deprecated: use Logger instead.
func TWarnf(format string, v ...interface{}) {
	deprecatedLogger.TWarnf(format, v...)
}

// TErrorf ...
// Deprecated: use Logger instead.
func TErrorf(format string, v ...interface{}) {
	deprecatedLogger.TErrorf(format, v...)
}
