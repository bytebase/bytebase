package log

import (
	"fmt"
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

// MustInitialize initializes the global logger for the bytebase server.
func MustInitialize(debug bool) {
	gLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
	if debug {
		gLevel.SetLevel(zap.DebugLevel)
	}
	gl = zap.New(zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
		zapcore.Lock(os.Stdout),
		gLevel,
	))
}

// MustInitializeBB initializes the global logger for the BB CLI.
func MustInitializeBB() {
	logConfig := zap.NewProductionConfig()
	// Always set encoding to "console" for now since we do not redirect to file.
	logConfig.Encoding = "console"
	// "console" encoding needs to use the corresponding development encoder config.
	logConfig.EncoderConfig = zap.NewDevelopmentEncoderConfig()
	logConfig.Level = zap.NewAtomicLevelAt(zap.InfoLevel)

	logger, err := logConfig.Build()
	if err != nil {
		panic(fmt.Errorf("failed to create logger. %w", err))
	}
	gl = logger
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
	gl.Error(msg, fields...)
}

// Sync wraps the zap Logger's Sync method.
func Sync() {
	_ = gl.Sync()
}
