// Copyright 2020 Andy Bursavich. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package zapr provides a logr.Logger interface around a zap implementation,
// including metrics and a standard library log.Logger adapter.
package zapr

import (
	"bytes"
	"log"
	"reflect"

	"github.com/go-logr/logr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LogSink represents the ability to log messages, both errors and not.
type LogSink interface {
	logr.LogSink
	logr.CallDepthLogSink

	// Underlying returns the underlying *zap.Logger with no caller skips.
	// It may return nil if the logger is disabled.
	Underlying() *zap.Logger

	// Flush writes any buffered data to the underlying io.Writer.
	Flush() error
}

type sink struct {
	underlying *zap.Logger
	logger     *zap.Logger
	info       logr.RuntimeInfo
	depth      int
	errKey     string
	logLevel   int
	maxLevel   int
	metrics    Metrics
}

// NewLogger returns a new Logger with the given Config.
func NewLogger(c *Config) logr.Logger {
	return logr.New(NewLogSink(c))
}

// NewLogSink returns a new LogSink with the given Config.
func NewLogSink(c *Config) LogSink {
	underlying := newZapLogger(c)
	if c.Metrics != nil {
		c.Metrics.Init(loggerName(underlying))
	}
	return &sink{
		underlying: underlying,
		logger:     underlying,
		errKey:     c.ErrorKey,
		logLevel:   0,
		maxLevel:   c.Level,
		metrics:    c.Metrics,
	}
}

func (s *sink) sweeten(kvs []interface{}) []zapcore.Field {
	if len(kvs) == 0 {
		return nil
	}
	fields := make([]zapcore.Field, 0, len(kvs)/2)
	for i, n := 0, len(kvs)-1; i <= n; {
		switch key := kvs[i].(type) {
		case string:
			if i == n {
				s.sweetenDPanic("Ignored key without a value.",
					zap.Int("position", i),
					zap.String("key", key),
				)
				return fields
			}
			fields = append(fields, zap.Any(key, kvs[i+1]))
			i += 2
		case zapcore.Field:
			s.sweetenDPanic("Zap Field passed to logr",
				zap.Int("position", i),
				zap.String("key", key.Key),
			)
			fields = append(fields, key)
			i++
		default:
			s.sweetenDPanic("Ignored key-value pair with non-string key",
				zap.Int("position", i),
				zap.Any("type", reflect.TypeOf(key).String()),
			)
			i += 2
		}
	}
	return fields
}

func (s *sink) sweetenDPanic(msg string, fields ...zapcore.Field) {
	s.logger.WithOptions(zap.AddCallerSkip(1)).DPanic(msg, fields...)
}

func (s *sink) Init(info logr.RuntimeInfo) {
	s.info = info
	s.logger = s.underlying.WithOptions(zap.AddCallerSkip(s.info.CallDepth + s.depth))
}

func (s *sink) Enabled(level int) bool { return level <= s.maxLevel }

func (s *sink) Info(level int, msg string, keysAndValues ...interface{}) {
	if level > s.maxLevel {
		return
	}
	if ce := s.logger.Check(zapcore.InfoLevel, msg); ce != nil {
		ce.Write(s.sweeten(keysAndValues)...)
	}
}

func (s *sink) Error(err error, msg string, keysAndValues ...interface{}) {
	if ce := s.logger.Check(zapcore.ErrorLevel, msg); ce != nil {
		kvs := keysAndValues
		if s.errKey != "" && err != nil {
			kvs = make([]interface{}, 0, len(keysAndValues)+2)
			kvs = append(kvs, keysAndValues...)
			kvs = append(kvs, s.errKey, err.Error())
		}
		ce.Write(s.sweeten(kvs)...)
	}
}

func (s *sink) WithValues(keysAndValues ...interface{}) logr.LogSink {
	v := *s
	v.logger = s.logger.With(s.sweeten(keysAndValues)...)
	return &v
}

func (s *sink) WithName(name string) logr.LogSink {
	v := *s
	v.underlying = v.underlying.Named(name)
	v.Init(v.info)
	if v.metrics != nil {
		v.metrics.Init(loggerName(v.logger))
	}
	return &v
}

func (s *sink) WithCallDepth(depth int) logr.LogSink {
	if depth == 0 {
		return s
	}
	v := *s
	v.depth += depth
	v.Init(s.info)
	return &v
}

func (s *sink) Underlying() *zap.Logger { return s.underlying }

func (s *sink) Flush() error { return s.logger.Sync() }

// NewStdInfoLogger returns a *log.Logger which writes to the supplied Logger's Info method.
func NewStdInfoLogger(s logr.CallDepthLogSink) *log.Logger {
	infoFn := s.WithCallDepth(4).Info
	fn := func(msg string, _ ...interface{}) { infoFn(0, msg) }
	return log.New(stdLogWriterFunc(fn), "" /*prefix*/, 0 /*flags*/)
}

// NewStdErrorLogger returns a *log.Logger which writes to the supplied Logger's Error method.
func NewStdErrorLogger(s logr.CallDepthLogSink) *log.Logger {
	errFn := s.WithCallDepth(4).Error
	fn := func(msg string, _ ...interface{}) { errFn(nil, msg) }
	return log.New(stdLogWriterFunc(fn), "" /*prefix*/, 0 /*flags*/)
}

type stdLogWriterFunc func(msg string, _ ...interface{})

func (fn stdLogWriterFunc) Write(b []byte) (int, error) {
	v := bytes.TrimSpace(b)
	fn(string(v))
	return len(b), nil
}
