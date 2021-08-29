// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright 2020 Andy Bursavich. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zapr

import (
	"os"
	"sort"
	"time"

	"bursavich.dev/zapr/encoding"
	"go.uber.org/zap/zapcore"
)

type config struct {
	ws    zapcore.WriteSyncer
	name  string
	level int

	timeKey       string
	levelKey      string
	nameKey       string
	callerKey     string
	functionKey   string
	messageKey    string
	errorKey      string
	stacktraceKey string
	lineEnding    string

	encoder         encoding.Encoder
	timeEncoder     encoding.TimeEncoder
	levelEncoder    encoding.LevelEncoder
	durationEncoder encoding.DurationEncoder
	callerEncoder   encoding.CallerEncoder

	enableStacktrace bool
	enableCaller     bool
	development      bool

	sampleTick       time.Duration
	sampleFirst      int
	sampleThereafter int
	sampleOpts       []zapcore.SamplerOption

	observer Observer
}

func configWithOptions(options []Option) *config {
	c := &config{
		ws:               zapcore.Lock(os.Stderr),
		name:             "",
		level:            0,
		timeKey:          "time",
		levelKey:         "level",
		nameKey:          "logger",
		callerKey:        "caller",
		functionKey:      "",
		messageKey:       "msg",
		errorKey:         "error",
		stacktraceKey:    "stacktrace",
		lineEnding:       zapcore.DefaultLineEnding,
		encoder:          encoding.JSONEncoder(),
		timeEncoder:      encoding.ISO8601TimeEncoder(),
		levelEncoder:     encoding.UppercaseLevelEncoder(),
		durationEncoder:  encoding.SecondsDurationEncoder(),
		callerEncoder:    encoding.ShortCallerEncoder(),
		enableStacktrace: false,
		enableCaller:     true,
		development:      false,
		sampleTick:       time.Second,
		sampleFirst:      100,
		sampleThereafter: 100,
		observer:         nil,
	}
	for _, o := range sortedOptions(options) {
		o.apply(c)
	}
	return c
}

// An Option applies optional configuration.
type Option interface {
	apply(*config)
	weight() int
}

type opt struct {
	fn func(*config)
	w  int
}

func optionFunc(fn func(*config)) Option {
	return opt{fn: fn}
}

func weightedOptionFunc(weight int, fn func(*config)) Option {
	return opt{fn: fn, w: weight}
}

func (o opt) apply(c *config) { o.fn(c) }
func (o opt) weight() int     { return o.w }

type byWeightDesc []Option

func (s byWeightDesc) Len() int           { return len(s) }
func (s byWeightDesc) Less(i, k int) bool { return s[i].weight() > s[k].weight() } // reversed
func (s byWeightDesc) Swap(i, k int)      { s[i], s[k] = s[k], s[i] }

func sortedOptions(options []Option) []Option {
	if sort.IsSorted(byWeightDesc(options)) {
		return options
	}
	// sort a copy
	options = append(make([]Option, 0, len(options)), options...)
	sort.Stable(byWeightDesc(options))
	return options
}

// WithWriteSyncer returns an Option that sets the underlying writer.
// The default value is stderr.
func WithWriteSyncer(ws zapcore.WriteSyncer) Option {
	return optionFunc(func(c *config) { c.ws = ws })
}

// WithName returns an Option that sets the name.
// The default value is empty.
func WithName(name string) Option {
	return optionFunc(func(c *config) { c.name = name })
}

// WithLevel returns an Option that sets the level.
// The default value is 0.
func WithLevel(level int) Option {
	return optionFunc(func(c *config) { c.level = level })
}

// WithTimeKey returns an Option that sets the time key.
// The default value is "time".
func WithTimeKey(key string) Option {
	return optionFunc(func(c *config) { c.timeKey = key })
}

// WithLevelKey returns an Option that sets the level key.
// The default value is "level".
func WithLevelKey(key string) Option {
	return optionFunc(func(c *config) { c.levelKey = key })
}

// WithNameKey returns an Option that sets the name key.
// The default value is "logger".
func WithNameKey(key string) Option {
	return optionFunc(func(c *config) { c.nameKey = key })
}

// WithCallerKey returns an Option that sets the caller key.
// The default value is "caller".
func WithCallerKey(key string) Option {
	return optionFunc(func(c *config) { c.callerKey = key })
}

// WithFunctionKey returns an Option that sets the function key.
// The default value is empty.
func WithFunctionKey(key string) Option {
	return optionFunc(func(c *config) { c.functionKey = key })
}

// WithMessageKey returns an Option that sets the message key.
// The default value is "msg".
func WithMessageKey(key string) Option {
	return optionFunc(func(c *config) { c.messageKey = key })
}

// WithErrorKey returns an Option that sets the error key.
// The default value is "error".
func WithErrorKey(key string) Option {
	return optionFunc(func(c *config) { c.errorKey = key })
}

// WithStacktraceKey returns an Option that sets the stacktrace key.
// The default value is "stacktrace".
func WithStacktraceKey(key string) Option {
	return optionFunc(func(c *config) { c.stacktraceKey = key })
}

// WithLineEnding returns an Option that sets the line-ending.
// The default value is "\n".
func WithLineEnding(ending string) Option {
	return optionFunc(func(c *config) { c.lineEnding = ending })
}

// WithEncoder returns an Option that sets the encoder.
// The default value is a JSONEncoder.
func WithEncoder(encoder encoding.Encoder) Option {
	return optionFunc(func(c *config) { c.encoder = encoder })
}

// WithTimeEncoder returns an Option that sets the encoder.
// The default encoding is ISO 8601.
func WithTimeEncoder(encoder encoding.TimeEncoder) Option {
	return optionFunc(func(c *config) { c.timeEncoder = encoder })
}

// WithLevelEncoder returns an Option that sets the level encoder.
// The default encoding is uppercase.
func WithLevelEncoder(encoder encoding.LevelEncoder) Option {
	return optionFunc(func(c *config) { c.levelEncoder = encoder })
}

// WithDurationEncoder returns an Option that sets the duration encoder.
// The default encoding is seconds.
func WithDurationEncoder(encoder encoding.DurationEncoder) Option {
	return optionFunc(func(c *config) { c.durationEncoder = encoder })
}

// WithCallerEncoder returns an Option that sets the caller encoder.
// The default encoding is short.
func WithCallerEncoder(encoder encoding.CallerEncoder) Option {
	return optionFunc(func(c *config) { c.callerEncoder = encoder })
}

// WithCallerEnabled returns an Option that sets whether the caller field
// is enabled. It's enabled by default.
func WithCallerEnabled(enabled bool) Option {
	return optionFunc(func(c *config) { c.enableCaller = enabled })
}

// WithStacktraceEnabled returns an Option that sets whether the stacktrace
// field is enabled. It's disabled by default.
func WithStacktraceEnabled(enabled bool) Option {
	return optionFunc(func(c *config) { c.enableStacktrace = enabled })
}

// WithSampler returns an Option that sets sampler options.
// The default is 1s tick, 100 first, and 100 thereafter.
func WithSampler(tick time.Duration, first, thereafter int, opts ...zapcore.SamplerOption) Option {
	return weightedOptionFunc(1, func(c *config) {
		c.sampleTick = tick
		c.sampleFirst = first
		c.sampleThereafter = thereafter
		c.sampleOpts = opts
	})
}

// WithDevelopmentOptions returns an Option that enables a set of
// development-friendly options.
func WithDevelopmentOptions() Option {
	return weightedOptionFunc(1, func(c *config) {
		c.level = 3
		c.functionKey = "func"
		c.encoder = encoding.ConsoleEncoder()
		c.levelEncoder = encoding.ColorLevelEncoder()
		c.durationEncoder = encoding.StringDurationEncoder()
		c.enableStacktrace = true
		c.development = true
	})
}
