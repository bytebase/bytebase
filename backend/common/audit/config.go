package audit

import (
	"time"

	"github.com/bytebase/bytebase/backend/common"
)

// Config defines configuration for the audit logging system
type Config struct {
	// Input queue
	InputQueueSize int // Number of events buffered before backpressure (default: 5000)

	// Stdout writer
	StdoutEnabled   bool // Enable JSON streaming to stdout (default: false)
	StdoutQueueSize int  // Stdout writer queue size (default: 5000)

	// Database batching
	DBBatchQueueSize int           // DB batcher queue size (default: 500)
	DBBatchSize      int           // Max events per batch insert (default: 100)
	DBBatchInterval  time.Duration // Max time between batch inserts (default: 100ms)
	DBWriteTimeout   time.Duration // DB write timeout per batch (default: 5s)

	// Circuit breakers (PR #3)
	DBCircuitTimeoutThresh int  // DB timeouts before circuit opens (default: 3)
	StdoutCircuitErrThresh int  // Stdout errors before circuit opens (default: 10)
	StdoutFallbackOnDBFail bool // Enable stdout fallback when DB fails (default: true)

	// Recovery goroutine (PR #3)
	RecoveryBaseInterval time.Duration // Initial recovery attempt interval (default: 30s)
	RecoveryMaxInterval  time.Duration // Max recovery attempt interval (default: 30m)
}

// DefaultConfig returns production-ready configuration
func DefaultConfig() Config {
	return Config{
		// Input
		InputQueueSize: 5000,

		// Stdout
		StdoutEnabled:   false,
		StdoutQueueSize: 5000,

		// Database
		DBBatchQueueSize: 500,
		DBBatchSize:      100,
		DBBatchInterval:  100 * time.Millisecond,
		DBWriteTimeout:   5 * time.Second,

		// Circuit breakers
		DBCircuitTimeoutThresh: 3,
		StdoutCircuitErrThresh: 10,
		StdoutFallbackOnDBFail: true,

		// Recovery
		RecoveryBaseInterval: 30 * time.Second,
		RecoveryMaxInterval:  30 * time.Minute,
	}
}

// Validate checks if configuration is valid
func (c *Config) Validate() error {
	// Queue sizes must be positive
	if c.InputQueueSize <= 0 {
		return common.Errorf(common.Invalid, "InputQueueSize must be > 0, got %d", c.InputQueueSize)
	}
	if c.StdoutQueueSize <= 0 {
		return common.Errorf(common.Invalid, "StdoutQueueSize must be > 0, got %d", c.StdoutQueueSize)
	}
	if c.DBBatchQueueSize <= 0 {
		return common.Errorf(common.Invalid, "DBBatchQueueSize must be > 0, got %d", c.DBBatchQueueSize)
	}

	// Batch size must be reasonable
	if c.DBBatchSize <= 0 {
		return common.Errorf(common.Invalid, "DBBatchSize must be > 0, got %d", c.DBBatchSize)
	}
	if c.DBBatchSize > c.DBBatchQueueSize {
		return common.Errorf(common.Invalid, "DBBatchSize (%d) cannot exceed DBBatchQueueSize (%d)", c.DBBatchSize, c.DBBatchQueueSize)
	}

	// Timeouts must be positive
	if c.DBBatchInterval <= 0 {
		return common.Errorf(common.Invalid, "DBBatchInterval must be > 0, got %v", c.DBBatchInterval)
	}
	if c.DBWriteTimeout <= 0 {
		return common.Errorf(common.Invalid, "DBWriteTimeout must be > 0, got %v", c.DBWriteTimeout)
	}

	// Circuit breaker thresholds must be positive
	if c.DBCircuitTimeoutThresh <= 0 {
		return common.Errorf(common.Invalid, "DBCircuitTimeoutThresh must be > 0, got %d", c.DBCircuitTimeoutThresh)
	}
	if c.StdoutCircuitErrThresh <= 0 {
		return common.Errorf(common.Invalid, "StdoutCircuitErrThresh must be > 0, got %d", c.StdoutCircuitErrThresh)
	}

	// Recovery intervals must be positive
	if c.RecoveryBaseInterval <= 0 {
		return common.Errorf(common.Invalid, "RecoveryBaseInterval must be > 0, got %v", c.RecoveryBaseInterval)
	}
	if c.RecoveryMaxInterval <= 0 {
		return common.Errorf(common.Invalid, "RecoveryMaxInterval must be > 0, got %v", c.RecoveryMaxInterval)
	}
	if c.RecoveryMaxInterval < c.RecoveryBaseInterval {
		return common.Errorf(common.Invalid, "RecoveryMaxInterval (%v) must be >= RecoveryBaseInterval (%v)",
			c.RecoveryMaxInterval, c.RecoveryBaseInterval)
	}

	return nil
}
