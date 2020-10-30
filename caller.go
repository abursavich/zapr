// Copyright 2020 Andy Bursavich. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zapr

import (
	"flag"
	"fmt"

	"go.uber.org/zap/zapcore"
)

// A CallerEncoder provides a named zapcore.CallerEncoder.
type CallerEncoder interface {
	CallerEncoder() zapcore.CallerEncoder
	Name() string
}

var callerEncoders = make(map[string]CallerEncoder)

// RegisterCallerEncoder registers the CallerEncoder for use as a flag argument.
func RegisterCallerEncoder(e CallerEncoder) error {
	name := e.Name()
	if _, ok := callerEncoders[name]; ok {
		return fmt.Errorf("zapr: CallerEncoder %q already ", name)
	}
	callerEncoders[name] = e
	return nil
}

type callerEncoder struct {
	e    zapcore.CallerEncoder
	name string
}

func (e *callerEncoder) CallerEncoder() zapcore.CallerEncoder { return e.e }
func (e *callerEncoder) Name() string                         { return e.name }

var (
	shortCallerEncoder = CallerEncoder(&callerEncoder{name: "short", e: zapcore.ShortCallerEncoder})
	fullCallerEncoder  = CallerEncoder(&callerEncoder{name: "full", e: zapcore.FullCallerEncoder})
)

func init() {
	RegisterCallerEncoder(shortCallerEncoder)
	RegisterCallerEncoder(fullCallerEncoder)
}

// ShortCallerEncoder serializes a caller in package/file:line format, trimming
// all but the final directory from the full path.
func ShortCallerEncoder() CallerEncoder { return shortCallerEncoder }

// FullCallerEncoder serializes a caller in /full/path/to/package/file:line
// format.
func FullCallerEncoder() CallerEncoder { return fullCallerEncoder }

type callerEncoderFlag struct {
	e *CallerEncoder
}

// CallerEncoderFlag returns a flag value for the encoder.
func CallerEncoderFlag(encoder *CallerEncoder) flag.Value {
	return &callerEncoderFlag{encoder}
}

func (f *callerEncoderFlag) Get() interface{} { return *f.e }
func (f *callerEncoderFlag) Set(s string) error {
	if e, ok := callerEncoders[s]; ok {
		*f.e = e
		return nil
	}
	return fmt.Errorf("zapr: unknown CallerEncoder: %q", s)
}
func (f *callerEncoderFlag) String() string {
	if f.e == nil {
		return ""
	}
	return (*f.e).Name()
}
