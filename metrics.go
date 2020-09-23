// Copyright 2020 Andy Bursavich. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zapr

import (
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

// Metrics are a prometheus.Collector for log metrics.
type Metrics struct {
	lines  *prometheus.CounterVec
	bytes  *prometheus.CounterVec
	errors *prometheus.CounterVec
}

// NewMetrics returns new Metrics.
func NewMetrics() *Metrics {
	return &Metrics{
		lines: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "log_lines_total",
				Help: "Total number of log lines.",
			},
			[]string{"name", "level"},
		),
		bytes: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "log_bytes_total",
				Help: "Total bytes of encoded log lines.",
			},
			[]string{"name", "level"},
		),
		errors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "log_encoder_errors_total",
				Help: "Total number of log entry encoding failures.",
			},
			[]string{"name"},
		),
	}
}

// Describe implements the prometheus.Collector interface.
func (m *Metrics) Describe(ch chan<- *prometheus.Desc) {
	m.lines.Describe(ch)
	m.bytes.Describe(ch)
	m.errors.Describe(ch)
}

// Collect implements the prometheus.Collector interface.
func (m *Metrics) Collect(ch chan<- prometheus.Metric) {
	m.lines.Collect(ch)
	m.bytes.Collect(ch)
	m.errors.Collect(ch)
}

func (m *Metrics) initLogger(log *zap.Logger) {
	name := log.Check(zapcore.FatalLevel, "").LoggerName
	for _, lvl := range []zapcore.Level{zap.InfoLevel, zap.ErrorLevel} {
		m.lines.WithLabelValues(name, lvl.String())
		m.bytes.WithLabelValues(name, lvl.String())
	}
	m.errors.WithLabelValues(name)
}

type metricsEncoder struct {
	zapcore.Encoder
	metrics *Metrics
}

// NewMetricsEncoder returns an Encoder that wraps the given Encoder
// and records metrics.
func NewMetricsEncoder(e zapcore.Encoder, m *Metrics) zapcore.Encoder {
	return &metricsEncoder{Encoder: e, metrics: m}
}

func (enc *metricsEncoder) Clone() zapcore.Encoder {
	return &metricsEncoder{
		Encoder: enc.Encoder.Clone(),
		metrics: enc.metrics,
	}
}

func (enc *metricsEncoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	b, err := enc.Encoder.EncodeEntry(entry, fields)
	if err != nil {
		enc.metrics.errors.WithLabelValues(entry.LoggerName).Inc()
		return nil, err
	}
	lvl := entry.Level.String()
	enc.metrics.lines.WithLabelValues(entry.LoggerName, lvl).Inc()
	enc.metrics.bytes.WithLabelValues(entry.LoggerName, lvl).Add(float64(b.Len()))
	return b, err
}
