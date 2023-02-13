// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright 2023 Andy Bursavich. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zapr

import (
	"sync"
	"sync/atomic"

	"github.com/go-logr/logr"
	"go.uber.org/zap"
)

var discard = LogSink(noopLogSink{})

type noopLogSink struct{}

func (noopLogSink) Init(info logr.RuntimeInfo)                        {}
func (noopLogSink) Enabled(level int) bool                            { return false }
func (noopLogSink) Info(level int, msg string, keysAndValues ...any)  {}
func (noopLogSink) Error(err error, msg string, keysAndValues ...any) {}
func (noopLogSink) WithValues(keysAndValues ...any) logr.LogSink      { return discard }
func (noopLogSink) WithName(name string) logr.LogSink                 { return discard }
func (noopLogSink) WithCallDepth(depth int) logr.LogSink              { return discard }
func (noopLogSink) Underlying() *zap.Logger                           { return nil }
func (noopLogSink) Flush() error                                      { return nil }

// LazyLogSink is a LogSink whose underlying implementation
// can be updated after it's been used to create log.Loggers.
type LazyLogSink interface {
	LogSink

	// SetSink sets the underlying LogSink for the LazyLogSink and all of
	// its descendants created by WithDepth, WithName, or WithValues.
	SetSink(LogSink)
}

// NewLazyLogSink returns a new Sink that discards logs until SetSink is called.
func NewLazyLogSink() LazyLogSink {
	return newLazySink()
}

type lazySink struct {
	sink atomic.Pointer[LogSink]
	info logr.RuntimeInfo

	mu       sync.Mutex
	set      bool
	name     string
	depth    int
	values   []any
	children []*lazySink
}

func newLazySink() *lazySink {
	var s lazySink
	s.sink.Store(&discard)
	return &s
}

func (s *lazySink) Init(info logr.RuntimeInfo) {
	info.CallDepth++
	s.info = info
	(*s.sink.Load()).Init(info)
}

func (s *lazySink) Enabled(level int) bool {
	return (*s.sink.Load()).Enabled(level)
}

func (s *lazySink) Info(level int, msg string, keysAndValues ...any) {
	(*s.sink.Load()).Info(level, msg, keysAndValues...)
}

func (s *lazySink) Error(err error, msg string, keysAndValues ...any) {
	(*s.sink.Load()).Error(err, msg, keysAndValues...)
}

func (s *lazySink) WithValues(keysAndValues ...any) logr.LogSink {
	s.mu.Lock()
	defer s.mu.Unlock()

	child := newLazySink()
	child.Init(s.info)
	child.values = append([]any(nil), keysAndValues...)
	if s.set {
		s.SetSink(*s.sink.Load())
	}
	s.children = append(s.children, child)
	return child
}

func (s *lazySink) WithName(name string) logr.LogSink {
	s.mu.Lock()
	defer s.mu.Unlock()

	child := newLazySink()
	child.Init(s.info)
	child.name = name
	if s.set {
		s.SetSink(*s.sink.Load())
	}
	s.children = append(s.children, child)
	return child
}

func (s *lazySink) WithCallDepth(depth int) logr.LogSink {
	s.mu.Lock()
	defer s.mu.Unlock()

	child := newLazySink()
	child.Init(s.info)
	child.depth = depth
	if s.set {
		s.SetSink(*s.sink.Load())
	}
	s.children = append(s.children, child)
	return child
}

func (s *lazySink) Underlying() *zap.Logger {
	return (*s.sink.Load()).Underlying()
}

func (s *lazySink) Flush() error {
	return (*s.sink.Load()).Flush()
}

func (s *lazySink) SetSink(sink LogSink) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sink.Init(s.info)
	if s.name != "" {
		sink = sink.WithName(s.name).(LogSink)
	}
	if len(s.values) > 0 {
		sink = sink.WithValues(s.values...).(LogSink)
	}
	if s.depth > 0 {
		sink = sink.WithCallDepth(s.depth).(LogSink)
	}
	s.sink.Store(&sink)
	s.set = true

	for _, c := range s.children {
		c.SetSink(sink)
	}
}
