// Copyright 2020 Andy Bursavich. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zapr

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	defaultEncoder         = jsonEncoder
	defaultTimeEncoder     = iso8601TimeEncoder
	defaultLevelEncoder    = uppercaseLevelEncoder
	defaultDurationEncoder = secsDurationEncoder
	defaultCallerEncoder   = shortCallerEncoder
)

// newZapLogger returns a new zap.Logger with the given config.
func newZapLogger(c *Config) *zap.Logger {
	var opts []zap.Option
	if c.Development {
		opts = append(opts, zap.Development())
	}
	if c.EnableCaller {
		opts = append(opts, zap.AddCaller())
	}
	if c.EnableStacktrace {
		opts = append(opts, zap.AddStacktrace(zapcore.ErrorLevel))
	}
	if c.SampleInitial != 0 || c.SampleThereafter != 0 {
		opts = append(opts, zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			return zapcore.NewSamplerWithOptions(core, time.Second, c.SampleInitial, c.SampleThereafter)
		}))
	}
	core := zapcore.NewCore(
		c.encoder(),
		zapcore.Lock(os.Stderr),
		zapcore.InfoLevel,
	)
	return zap.New(core, opts...).Named(c.Name)
}

// Config specifies the configuration of a Logger.
type Config struct {
	Name  string
	Level int

	TimeKey       string
	LevelKey      string
	NameKey       string
	CallerKey     string
	FunctionKey   string
	MessageKey    string
	ErrorKey      string
	StacktraceKey string
	LineEnding    string

	Encoder         Encoder
	TimeEncoder     TimeEncoder
	LevelEncoder    LevelEncoder
	DurationEncoder DurationEncoder
	CallerEncoder   CallerEncoder

	EnableStacktrace bool
	EnableCaller     bool
	Development      bool

	SampleInitial    int
	SampleThereafter int

	Metrics *Metrics
}

// DefaultConfig returns the default Config.
func DefaultConfig() *Config {
	return &Config{
		Name:             "",
		Level:            0,
		TimeKey:          "time",
		LevelKey:         "level",
		NameKey:          "logger",
		CallerKey:        "caller",
		FunctionKey:      "",
		MessageKey:       "msg",
		ErrorKey:         "error",
		StacktraceKey:    "stacktrace",
		LineEnding:       zapcore.DefaultLineEnding,
		Encoder:          defaultEncoder,
		TimeEncoder:      defaultTimeEncoder,
		LevelEncoder:     defaultLevelEncoder,
		DurationEncoder:  defaultDurationEncoder,
		CallerEncoder:    defaultCallerEncoder,
		EnableStacktrace: false,
		EnableCaller:     true,
		Development:      false,
		SampleInitial:    100,
		SampleThereafter: 100,
		Metrics:          NewMetrics(),
	}
}

// DevelopmentConfig returns a development-friendly Config.
func DevelopmentConfig() *Config {
	cfg := DefaultConfig()
	cfg.Level = 2
	cfg.FunctionKey = "func"
	cfg.Encoder = consoleEncoder
	cfg.LevelEncoder = colorLevelEncoder
	cfg.DurationEncoder = stringDurationEncoder
	cfg.EnableStacktrace = true
	cfg.Development = true
	return cfg
}

func (c *Config) encoder() zapcore.Encoder {
	enc := c.newEncoder(zapcore.EncoderConfig{
		TimeKey:        c.TimeKey,
		LevelKey:       c.LevelKey,
		NameKey:        c.NameKey,
		CallerKey:      c.CallerKey,
		FunctionKey:    c.FunctionKey,
		MessageKey:     c.MessageKey,
		StacktraceKey:  c.StacktraceKey,
		LineEnding:     c.LineEnding,
		EncodeTime:     c.timeEncoder(),
		EncodeLevel:    c.levelEncoder(),
		EncodeDuration: c.durationEncoder(),
		EncodeCaller:   c.callerEncoder(),
	})
	if c.Metrics != nil {
		return &metricsEncoder{
			Encoder: enc,
			metrics: c.Metrics,
		}
	}
	return enc
}

func (c *Config) newEncoder(cfg zapcore.EncoderConfig) zapcore.Encoder {
	if c == nil || c.Encoder == nil {
		return defaultEncoder.NewEncoder(cfg)
	}
	return c.Encoder.NewEncoder(cfg)
}

func (c *Config) timeEncoder() zapcore.TimeEncoder {
	if c == nil || c.TimeEncoder == nil {
		return defaultTimeEncoder.TimeEncoder()
	}
	return c.TimeEncoder.TimeEncoder()
}

func (c *Config) levelEncoder() zapcore.LevelEncoder {
	if c == nil || c.LevelEncoder == nil {
		return defaultLevelEncoder.LevelEncoder()
	}
	return c.LevelEncoder.LevelEncoder()
}

func (c *Config) durationEncoder() zapcore.DurationEncoder {
	if c == nil || c.DurationEncoder == nil {
		return defaultDurationEncoder.DurationEncoder()
	}
	return c.DurationEncoder.DurationEncoder()
}

func (c *Config) callerEncoder() zapcore.CallerEncoder {
	if c == nil || c.CallerEncoder == nil {
		return defaultCallerEncoder.CallerEncoder()
	}
	return c.CallerEncoder.CallerEncoder()
}

// RegisterCommonFlags registers basic fields of the Config as flags in the
// FlagSet. If fs is nil, flag.CommandLine is used.
func (c *Config) RegisterCommonFlags(fs *flag.FlagSet) *Config {
	if fs == nil {
		fs = flag.CommandLine
	}
	fs.IntVar(&c.Level, "log-level", c.Level, "Log level.")
	fs.Var(&encoderFlag{&c.Encoder}, "log-format", `Log format (e.g. "json" or "console").`)
	fs.Var(&timeEncoderFlag{&c.TimeEncoder}, "log-time-format", `Log time format (e.g. "iso8601", "millis", "nanos", or "secs").`)
	fs.Var(&levelEncoderFlag{&c.LevelEncoder}, "log-level-format", `Log level format (e.g. "upper", "lower", or "color").`)
	fs.Var(&callerEncoderFlag{&c.CallerEncoder}, "log-caller-format", `Log caller format (e.g. "short" or "full").`)
	return c
}

// RegisterFlags registers fields of the Config as flags in the FlagSet.
// If fs is nil, flag.CommandLine is used.
func (c *Config) RegisterFlags(fs *flag.FlagSet) *Config {
	if fs == nil {
		fs = flag.CommandLine
	}
	fs.StringVar(&c.TimeKey, "log-time-key", c.TimeKey, "Log time key.")
	fs.StringVar(&c.LevelKey, "log-level-key", c.LevelKey, "Log level key.")
	fs.StringVar(&c.MessageKey, "log-message-key", c.MessageKey, "Log message key.")
	fs.StringVar(&c.CallerKey, "log-caller-key", c.CallerKey, "Log caller key.")
	fs.StringVar(&c.FunctionKey, "log-function-key", c.FunctionKey, "Log function key.")
	fs.StringVar(&c.StacktraceKey, "log-stacktrace-key", c.StacktraceKey, "Log stacktrace key.")
	fs.BoolVar(&c.EnableStacktrace, "log-stacktrace", c.EnableStacktrace, `Log stacktrace on error or higher levels.`)
	fs.BoolVar(&c.EnableCaller, "log-caller", c.EnableCaller, `Log caller file and line.`)
	fs.IntVar(&c.SampleInitial, "log-sample-initial", c.SampleInitial, "Log every call up to this count per second.")
	fs.IntVar(&c.SampleThereafter, "log-sample-thereafter", c.SampleThereafter, "Log only one of this many calls after reaching the initial sample per second.")
	return c.RegisterCommonFlags(fs)
}

// An Encoder provides a named zapcore.Encoder.
type Encoder interface {
	NewEncoder(zapcore.EncoderConfig) zapcore.Encoder
	Name() string
}

type encoder struct {
	ctor func(zapcore.EncoderConfig) zapcore.Encoder
	name string
}

func (e *encoder) NewEncoder(c zapcore.EncoderConfig) zapcore.Encoder { return e.ctor(c) }
func (e *encoder) Name() string                                       { return e.name }

var (
	consoleEncoder = Encoder(&encoder{name: "console", ctor: zapcore.NewConsoleEncoder})
	jsonEncoder    = Encoder(&encoder{name: "json", ctor: zapcore.NewJSONEncoder})
)

// ConsoleEncoder creates an encoder whose output is designed for human
// consumption, rather than machine consumption.
func ConsoleEncoder() Encoder { return consoleEncoder }

// JSONEncoder creates a fast, low-allocation JSON encoder.
func JSONEncoder() Encoder { return jsonEncoder }

type encoderFlag struct {
	enc *Encoder
}

func (f *encoderFlag) Get() interface{} { return *f.enc }
func (f *encoderFlag) Set(s string) error {
	switch strings.ToLower(s) {
	case "json":
		*f.enc = jsonEncoder
	case "console":
		*f.enc = consoleEncoder
	default:
		return fmt.Errorf("unknown encoder: %q", s)
	}
	return nil
}
func (f *encoderFlag) String() string {
	if f.enc == nil {
		return ""
	}
	return (*f.enc).Name()
}

// A TimeEncoder provides a named zapcore.TimeEncoder.
type TimeEncoder interface {
	TimeEncoder() zapcore.TimeEncoder
	Name() string
}

type timeEncoder struct {
	enc  func(time.Time, zapcore.PrimitiveArrayEncoder)
	name string
}

func (e *timeEncoder) TimeEncoder() zapcore.TimeEncoder { return e.enc }
func (e *timeEncoder) Name() string                     { return e.name }

var (
	iso8601TimeEncoder = TimeEncoder(&timeEncoder{name: "iso8601", enc: zapcore.ISO8601TimeEncoder})
	millisTimeEncoder  = TimeEncoder(&timeEncoder{name: "millis", enc: zapcore.EpochMillisTimeEncoder})
	nanosTimeEncoder   = TimeEncoder(&timeEncoder{name: "nanos", enc: zapcore.EpochNanosTimeEncoder})
	secsTimeEncoder    = TimeEncoder(&timeEncoder{name: "secs", enc: zapcore.EpochTimeEncoder})
	rfc3339TimeEncoder = TimeEncoder(&timeEncoder{
		name: "rfc3339",
		enc: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			encodeTimeLayout(t, "2006-01-02T15:04:05.000Z07:00", enc)
		},
	})
)

func encodeTimeLayout(t time.Time, layout string, enc zapcore.PrimitiveArrayEncoder) {
	switch enc := enc.(type) {
	case interface{ AppendTimeLayout(time.Time, string) }:
		enc.AppendTimeLayout(t, layout)
	default:
		enc.AppendString(t.Format(layout))
	}
}

// ISO8601TimeEncoder serializes a time.Time to an ISO8601-formatted string with
// millisecond precision.
func ISO8601TimeEncoder() TimeEncoder { return iso8601TimeEncoder }

// RFC3339TimeEncoder serializes a time.Time to an RFC3339-formatted string with
// millisecond precision.
func RFC3339TimeEncoder() TimeEncoder { return rfc3339TimeEncoder }

// NanosecondsTimeEncoder serializes a time.Time to an integer number of nanoseconds
// since the Unix epoch.
func NanosecondsTimeEncoder() TimeEncoder { return nanosTimeEncoder }

// MillisecondsTimeEncoder serializes a time.Time to a floating-point number of
// milliseconds since the Unix epoch.
func MillisecondsTimeEncoder() TimeEncoder { return millisTimeEncoder }

// SecondsTimeEncoder serializes a time.Time to a floating-point number of seconds
// since the Unix epoch.
func SecondsTimeEncoder() TimeEncoder { return secsTimeEncoder }

type timeEncoderFlag struct {
	enc *TimeEncoder
}

func (f *timeEncoderFlag) Get() interface{} { return *f.enc }
func (f *timeEncoderFlag) Set(s string) error {
	switch strings.ToLower(s) {
	case "iso8601":
		*f.enc = iso8601TimeEncoder
	case "rfc3339":
		*f.enc = rfc3339TimeEncoder
	case "ns", "nanos", "nanoseconds":
		*f.enc = nanosTimeEncoder
	case "ms", "millis", "milliseconds":
		*f.enc = millisTimeEncoder
	case "s", "secs", "seconds":
		*f.enc = secsTimeEncoder
	default:
		return fmt.Errorf("unknown time encoder: %q", s)
	}
	return nil
}
func (f *timeEncoderFlag) String() string {
	if f.enc == nil {
		return ""
	}
	return (*f.enc).Name()
}

// A LevelEncoder provides a named zapcore.LevelEncoder.
type LevelEncoder interface {
	LevelEncoder() zapcore.LevelEncoder
	Name() string
}

type levelEncoder struct {
	enc  zapcore.LevelEncoder
	name string
}

func (e *levelEncoder) LevelEncoder() zapcore.LevelEncoder { return e.enc }
func (e *levelEncoder) Name() string                       { return e.name }

var (
	colorLevelEncoder     = LevelEncoder(&levelEncoder{name: "color", enc: zapcore.CapitalColorLevelEncoder})
	lowercaseLevelEncoder = LevelEncoder(&levelEncoder{name: "lower", enc: zapcore.LowercaseLevelEncoder})
	uppercaseLevelEncoder = LevelEncoder(&levelEncoder{name: "upper", enc: zapcore.CapitalLevelEncoder})
)

// ColorLevelEncoder serializes a Level to an all-caps string and adds color.
// For example, InfoLevel is serialized to "INFO" and colored blue.
func ColorLevelEncoder() LevelEncoder { return colorLevelEncoder }

// LowercaseLevelEncoder serializes a Level to a lowercase string. For example,
// InfoLevel is serialized to "info".
func LowercaseLevelEncoder() LevelEncoder { return lowercaseLevelEncoder }

// UppercaseLevelEncoder serializes a Level to an all-caps string. For example,
// InfoLevel is serialized to "INFO".
func UppercaseLevelEncoder() LevelEncoder { return uppercaseLevelEncoder }

type levelEncoderFlag struct {
	enc *LevelEncoder
}

func (f *levelEncoderFlag) Get() interface{} { return *f.enc }
func (f *levelEncoderFlag) Set(s string) error {
	switch strings.ToLower(s) {
	case "upper", "uppercase":
		*f.enc = uppercaseLevelEncoder
	case "lower", "lowercase":
		*f.enc = lowercaseLevelEncoder
	case "color":
		*f.enc = colorLevelEncoder
	default:
		return fmt.Errorf("unknown level encoder: %q", s)
	}
	return nil
}
func (f *levelEncoderFlag) String() string {
	if f.enc == nil {
		return ""
	}
	return (*f.enc).Name()
}

// A DurationEncoder provides a named zapcore.DurationEncoder.
type DurationEncoder interface {
	DurationEncoder() zapcore.DurationEncoder
	Name() string
}

type durationEncoder struct {
	enc  zapcore.DurationEncoder
	name string
}

func (e *durationEncoder) DurationEncoder() zapcore.DurationEncoder { return e.enc }
func (e *durationEncoder) Name() string                             { return e.name }

var (
	stringDurationEncoder = DurationEncoder(&durationEncoder{name: "string", enc: zapcore.StringDurationEncoder})
	nanosDurationEncoder  = DurationEncoder(&durationEncoder{name: "nanos", enc: zapcore.NanosDurationEncoder})
	millisDurationEncoder = DurationEncoder(&durationEncoder{name: "millis", enc: zapcore.MillisDurationEncoder})
	secsDurationEncoder   = DurationEncoder(&durationEncoder{name: "secs", enc: zapcore.SecondsDurationEncoder})
)

// StringDurationEncoder serializes a time.Duration using its String method.
func StringDurationEncoder() DurationEncoder { return stringDurationEncoder }

// NanosecondsDurationEncoder serializes a time.Duration to an integer number of nanoseconds.
func NanosecondsDurationEncoder() DurationEncoder { return nanosDurationEncoder }

// MillisecondsDurationEncoder serializes a time.Duration to a floating-point number of milliseconds.
func MillisecondsDurationEncoder() DurationEncoder { return millisDurationEncoder }

// SecondsDurationEncoder serializes a time.Duration to a floating-point number of seconds.
func SecondsDurationEncoder() DurationEncoder { return secsDurationEncoder }

type durationEncoderFlag struct {
	enc *DurationEncoder
}

func (f *durationEncoderFlag) Get() interface{} { return *f.enc }
func (f *durationEncoderFlag) Set(s string) error {
	switch strings.ToLower(s) {
	case "string":
		*f.enc = stringDurationEncoder
	case "ns", "nanos", "nanoseconds":
		*f.enc = nanosDurationEncoder
	case "ms", "millis", "milliseconds":
		*f.enc = millisDurationEncoder
	case "s", "secs", "seconds":
		*f.enc = secsDurationEncoder
	default:
		return fmt.Errorf("unknown time encoder: %q", s)
	}
	return nil
}
func (f *durationEncoderFlag) String() string {
	if f.enc == nil {
		return ""
	}
	return (*f.enc).Name()
}

// A CallerEncoder provides a named zapcore.CallerEncoder.
type CallerEncoder interface {
	CallerEncoder() zapcore.CallerEncoder
	Name() string
}

type callerEncoder struct {
	enc  zapcore.CallerEncoder
	name string
}

func (e *callerEncoder) CallerEncoder() zapcore.CallerEncoder { return e.enc }
func (e *callerEncoder) Name() string                         { return e.name }

var (
	shortCallerEncoder = CallerEncoder(&callerEncoder{name: "short", enc: zapcore.ShortCallerEncoder})
	fullCallerEncoder  = CallerEncoder(&callerEncoder{name: "full", enc: zapcore.FullCallerEncoder})
)

// ShortCallerEncoder serializes a caller in package/file:line format, trimming
// all but the final directory from the full path.
func ShortCallerEncoder() CallerEncoder { return shortCallerEncoder }

// FullCallerEncoder serializes a caller in /full/path/to/package/file:line
// format.
func FullCallerEncoder() CallerEncoder { return fullCallerEncoder }

type callerEncoderFlag struct {
	enc *CallerEncoder
}

func (f *callerEncoderFlag) Get() interface{} { return *f.enc }
func (f *callerEncoderFlag) Set(s string) error {
	switch strings.ToLower(s) {
	case "short":
		*f.enc = shortCallerEncoder
	case "full":
		*f.enc = fullCallerEncoder
	default:
		return fmt.Errorf("unknown level encoder: %q", s)
	}
	return nil
}
func (f *callerEncoderFlag) String() string {
	if f.enc == nil {
		return ""
	}
	return (*f.enc).Name()
}
