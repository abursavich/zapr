package zapr

import (
	"bytes"
	"encoding/json"
	"flag"
	"log"
	"strconv"
	"strings"
	"testing"

	"bursavich.dev/zapr/encoding"
	"github.com/go-logr/logr"
	"go.uber.org/zap/zapcore"
)

func TestLogger(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	log, _ := NewLogger(
		WithEncoder(encoding.JSONEncoder()),
		WithLineEnding("\n"),
		WithCallerEncoder(encoding.ShortCallerEncoder()),
		WithCallerKey("caller"),
		WithLevelEncoder(encoding.UppercaseLevelEncoder()),
		WithLevelKey("level"),
		WithMessageKey("message"),
		WithWriteSyncer(zapcore.AddSync(buf)),
	)
	log.Info("hello", "foo", "world", "bar", 42)
	t.Log("\n" + strings.TrimSpace(buf.String())) // help debugging

	var entry struct {
		Level   string `json:"level"`
		Caller  string `json:"caller"`
		Message string `json:"message"`
		Foo     string `json:"foo"`
		Bar     int    `json:"bar"`
	}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatal(err)
	}
	if want, got := "INFO", entry.Level; got != want {
		t.Errorf("unexpected level: want: %q; got: %q", want, got)
	}
	if want, got := "zapr/sink_test.go:", entry.Caller; !strings.HasPrefix(got, want) {
		t.Errorf("unexpected caller: want prefix: %q; got: %q", want, got)
	}
	if want, got := "hello", entry.Message; got != want {
		t.Errorf("unexpected message: want: %q; got: %q", want, got)
	}
	if want, got := "world", entry.Foo; got != want {
		t.Errorf("unexpected foo: want: %q; got: %q", want, got)
	}
	if want, got := 42, entry.Bar; got != want {
		t.Errorf("unexpected bar: want: %v; got: %v", want, got)
	}
}

func TestFlag(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	opts := RegisterFlags(fs, AllOptions()...)
	if err := fs.Parse([]string{"--log-development"}); err != nil {
		t.Fatal(err)
	}
	c := configWithOptions(opts)
	if want, got := true, c.development; got != want {
		t.Errorf("unexpected development: want: %v; got: %v", want, got)
	}
}

func TestStdLog(t *testing.T) {
	tests := []struct {
		name string
		ctor func(logr.CallDepthLogSink) *log.Logger
		lvl  string
	}{
		{
			name: "NewStdInfoLogger",
			ctor: NewStdInfoLogger,
			lvl:  "INFO",
		},
		{
			name: "NewStdErrorLogger",
			ctor: NewStdErrorLogger,
			lvl:  "ERROR",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewBuffer(nil)
			_, sink := NewLogger(
				WithEncoder(encoding.JSONEncoder()),
				WithLineEnding("\n"),
				WithCallerEncoder(encoding.ShortCallerEncoder()),
				WithCallerKey("caller"),
				WithLevelEncoder(encoding.UppercaseLevelEncoder()),
				WithLevelKey("level"),
				WithMessageKey("message"),
				WithWriteSyncer(zapcore.AddSync(buf)),
			)

			logger := tt.ctor(sink)
			logger.Printf("can be on any line")
			logger.Printf("must be on the next line")
			t.Log("\n" + strings.TrimSpace(buf.String())) // help debugging

			var lines [2]int
			dec := json.NewDecoder(buf)
			for i := range lines {
				var entry struct {
					Level   string `json:"level"`
					Caller  string `json:"caller"`
					Message string `json:"message"`
				}
				if err := dec.Decode(&entry); err != nil {
					t.Fatalf("failed to decode entry: %v", err)
				}
				if want, got := tt.lvl, entry.Level; want != got {
					t.Errorf("unexpected level; want: %q; got: %q", want, got)
				}
				const prefix = "zapr/sink_test.go:"
				if !strings.HasPrefix(entry.Caller, prefix) {
					t.Fatalf("unexpected caller; want prefix: %q; got: %q", prefix, entry.Caller)
				}
				line, err := strconv.Atoi(strings.TrimPrefix(entry.Caller, prefix))
				if err != nil {
					t.Fatalf("failed to parse line from caller: %q; err: %v", entry.Caller, err)
				}
				lines[i] = line
			}
			if lines[1]-lines[0] != 1 {
				t.Fatalf("unexpected caller lines: got: %d, %d", lines[0], lines[1])
			}
		})
	}
}
