package log

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// `gl` is the global logger.
	// Other packages should use public methods like Info/Error to do the logging.
	// If special logging is required (like log to a separate file for some special operations), we need to add other loggers.
	gl     *zap.Logger
	gLevel zap.AtomicLevel
)

// Initializes the global console logger.
func init() {
	gLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
	gl = zap.New(zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
		zapcore.Lock(os.Stdout),
		gLevel,
	))
}

// SetLevel wraps the zap Level's SetLevel method.
func SetLevel(level zapcore.Level) {
	gLevel.SetLevel(level)
}

// EnabledLevel wraps the zap Level's Enabled method.
func EnabledLevel(level zapcore.Level) bool {
	return gLevel.Enabled(level)
}

// Debug wraps the zap Logger's Debug method.
func Debug(msg string, fields ...zap.Field) {
	gl.Debug(msg, fields...)
}

// Info wraps the zap Logger's Info method.
func Info(msg string, fields ...zap.Field) {
	gl.Info(msg, fields...)
}

// Warn wraps the zap Logger's Warn method.
func Warn(msg string, fields ...zap.Field) {
	gl.Warn(msg, fields...)
}

// Error wraps the zap Logger's Error method.
func Error(msg string, fields ...zap.Field) {
	// Append the stack info in Error logging for better debugging experience.
	// Note that we should skip one stack frames so that the top frame start at the caller of log.Error.
	fields = append(fields, zap.StackSkip("stack", 1))
	gl.Error(msg, fields...)
}

// Sync wraps the zap Logger's Sync method.
func Sync() {
	_ = gl.Sync()
}
