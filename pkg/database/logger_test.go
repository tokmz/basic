package database

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"
	"gorm.io/gorm/logger"
)

func TestNewZapLogger(t *testing.T) {
	zapLogger := zaptest.NewLogger(t)
	config := LoggerConfig{
		ZapLogger:     zapLogger,
		Level:         Info,
		SlowThreshold: time.Millisecond * 100,
		Colorful:      true,
	}

	gormLogger := NewZapLogger(zapLogger, config)
	zapLog, ok := gormLogger.(*ZapLogger)
	if !ok {
		t.Fatal("Expected *ZapLogger type")
	}

	if zapLog.ZapLogger != zapLogger {
		t.Error("ZapLogger field not set correctly")
	}
	if zapLog.LogLevel != logger.Info {
		t.Errorf("Expected log level Info, got %v", zapLog.LogLevel)
	}
	if zapLog.SlowThreshold != time.Millisecond*100 {
		t.Errorf("Expected slow threshold 100ms, got %v", zapLog.SlowThreshold)
	}
	if !zapLog.Colorful {
		t.Error("Expected colorful to be true")
	}
	if !zapLog.IgnoreRecordNotFoundError {
		t.Error("Expected IgnoreRecordNotFoundError to be true")
	}
}

func TestZapLogger_LogMode(t *testing.T) {
	zapLogger := zaptest.NewLogger(t)
	config := LoggerConfig{Level: Info}
	
	gormLogger := NewZapLogger(zapLogger, config)
	newLogger := gormLogger.LogMode(logger.Warn)

	zapLog, ok := newLogger.(*ZapLogger)
	if !ok {
		t.Fatal("Expected *ZapLogger type")
	}

	if zapLog.LogLevel != logger.Warn {
		t.Errorf("Expected log level Warn, got %v", zapLog.LogLevel)
	}

	// Original logger should be unchanged
	originalZapLog := gormLogger.(*ZapLogger)
	if originalZapLog.LogLevel != logger.Info {
		t.Errorf("Original logger level should remain Info, got %v", originalZapLog.LogLevel)
	}
}

func TestZapLogger_Info(t *testing.T) {
	zapLogger := zaptest.NewLogger(t)
	config := LoggerConfig{Level: Info}
	
	gormLogger := NewZapLogger(zapLogger, config)
	ctx := context.Background()

	// Should log when level is Info or higher
	gormLogger.Info(ctx, "test info message: %s", "param")

	// Should not log when level is too low
	quietLogger := gormLogger.LogMode(logger.Error)
	quietLogger.Info(ctx, "this should not be logged: %s", "param")
}

func TestZapLogger_Warn(t *testing.T) {
	zapLogger := zaptest.NewLogger(t)
	config := LoggerConfig{Level: Warn}
	
	gormLogger := NewZapLogger(zapLogger, config)
	ctx := context.Background()

	// Should log when level is Warn or higher
	gormLogger.Warn(ctx, "test warning message: %s", "param")

	// Should not log when level is too low
	quietLogger := gormLogger.LogMode(logger.Error)
	quietLogger.Warn(ctx, "this should not be logged: %s", "param")
}

func TestZapLogger_Error(t *testing.T) {
	zapLogger := zaptest.NewLogger(t)
	config := LoggerConfig{Level: Error}
	
	gormLogger := NewZapLogger(zapLogger, config)
	ctx := context.Background()

	// Should log when level is Error or higher
	gormLogger.Error(ctx, "test error message: %s", "param")

	// Should not log when level is Silent
	silentLogger := gormLogger.LogMode(logger.Silent)
	silentLogger.Error(ctx, "this should not be logged: %s", "param")
}

func TestZapLogger_Trace(t *testing.T) {
	zapLogger := zaptest.NewLogger(t)
	config := LoggerConfig{
		Level:         Info,
		SlowThreshold: time.Millisecond * 100,
	}
	
	gormLogger := NewZapLogger(zapLogger, config)
	ctx := context.Background()
	begin := time.Now()

	tests := []struct {
		name     string
		duration time.Duration
		err      error
		sql      string
		rows     int64
	}{
		{
			name:     "normal query",
			duration: time.Millisecond * 50,
			err:      nil,
			sql:      "SELECT * FROM users",
			rows:     5,
		},
		{
			name:     "slow query",
			duration: time.Millisecond * 200,
			err:      nil,
			sql:      "SELECT * FROM users WHERE complex_condition = true",
			rows:     10,
		},
		{
			name:     "query with error",
			duration: time.Millisecond * 30,
			err:      errors.New("database error"),
			sql:      "SELECT * FROM invalid_table",
			rows:     0,
		},
		{
			name:     "record not found error",
			duration: time.Millisecond * 20,
			err:      logger.ErrRecordNotFound,
			sql:      "SELECT * FROM users WHERE id = 999",
			rows:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc := func() (string, int64) {
				return tt.sql, tt.rows
			}

			// Simulate the duration by setting begin time
			testBegin := begin.Add(-tt.duration)
			gormLogger.Trace(ctx, testBegin, fc, tt.err)
		})
	}
}

func TestConvertLogLevel(t *testing.T) {
	tests := []struct {
		input    LogLevel
		expected logger.LogLevel
	}{
		{Silent, logger.Silent},
		{Error, logger.Error},
		{Warn, logger.Warn},
		{Info, logger.Info},
		{LogLevel(999), logger.Info}, // Default case
	}

	for _, tt := range tests {
		t.Run(tt.input.String(), func(t *testing.T) {
			result := convertLogLevel(tt.input)
			if result != tt.expected {
				t.Errorf("convertLogLevel(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGormLogWriter_Write(t *testing.T) {
	writer := &GormLogWriter{}
	
	testData := []byte("test log message\n")
	n, err := writer.Write(testData)
	
	if err != nil {
		t.Errorf("Write() returned error: %v", err)
	}
	if n != len(testData) {
		t.Errorf("Write() returned %d bytes written, expected %d", n, len(testData))
	}
}

func TestZapLogger_SilentLevel(t *testing.T) {
	zapLogger := zaptest.NewLogger(t)
	config := LoggerConfig{Level: Silent}
	
	gormLogger := NewZapLogger(zapLogger, config)
	ctx := context.Background()
	begin := time.Now()

	fc := func() (string, int64) {
		return "SELECT 1", 1
	}

	// No logs should be produced when level is Silent
	gormLogger.Trace(ctx, begin, fc, nil)
	gormLogger.Info(ctx, "test")
	gormLogger.Warn(ctx, "test")
	gormLogger.Error(ctx, "test")
}