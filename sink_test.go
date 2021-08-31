package zapr

import (
	"bytes"
	"encoding/json"
	"flag"
	"strings"
	"testing"

	"bursavich.dev/zapr/encoding"
	"go.uber.org/zap/zapcore"
)

func TestLogger(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	log, _ := NewLogger(
		// defaults
		WithEncoder(encoding.JSONEncoder()),
		WithLineEnding("\n"),
		WithCallerEncoder(encoding.ShortCallerEncoder()),
		WithCallerKey("caller"),
		WithLevelEncoder(encoding.UppercaseLevelEncoder()),
		WithLevelKey("level"),
		WithMessageKey("msg"),
		// override
		WithWriteSyncer(zapcore.AddSync(buf)),
	)
	log.Info("hello", "foo", "world", "bar", 42)
	t.Log(strings.TrimSpace(buf.String()))

	var entry struct {
		Level  string `json:"level"`
		Caller string `json:"caller"`
		Msg    string `json:"msg"`
		Foo    string `json:"foo"`
		Bar    int    `json:"bar"`
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
	if want, got := "hello", entry.Msg; got != want {
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

	b := bytes.NewBuffer(nil)
	fs.SetOutput(b)
	fs.Usage()
	t.Logf("%s", b.Bytes())
}
