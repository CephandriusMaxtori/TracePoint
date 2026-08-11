package osutil

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"
)

type LineWriter struct {
	buf   []byte
	lines int
	log   func(string)
}

func NewLineWriter(log func(string)) *LineWriter {
	return &LineWriter{log: log}
}

func (w *LineWriter) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)
	for {
		i := indexByte(w.buf, '\n')
		if i < 0 {
			break
		}
		line := strings.TrimRight(string(w.buf[:i]), "\r")
		w.buf = w.buf[i+1:]
		if line != "" {
			w.log(line)
			w.lines++
		}
	}
	return len(p), nil
}

func (w *LineWriter) Flush() {
	if len(w.buf) > 0 {
		line := strings.TrimRight(string(w.buf), "\r\n")
		w.buf = nil
		if line != "" {
			w.log(line)
			w.lines++
		}
	}
}

func indexByte(b []byte, c byte) int {
	for i, x := range b {
		if x == c {
			return i
		}
	}
	return -1
}

func IsWindows() bool  { return runtime.GOOS == "windows" }
func IsLinux() bool    { return runtime.GOOS == "linux" }
func IsDarwin() bool   { return runtime.GOOS == "darwin" }

func LookPath(name string) string {
	p, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	return p
}

func RunStream(ctx context.Context, name string, args []string, log func(string)) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = NewLineWriter(log)
	cmd.Stderr = NewLineWriter(log)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	err := cmd.Wait()
	return err
}

func RunCapture(ctx context.Context, name string, args []string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return strings.TrimSpace(string(ee.Stderr)) + "\n" + strings.TrimSpace(string(out)), err
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func RunCaptureStd(ctx context.Context, name string, args []string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(string(out)), nil
}

func CopyStream(dst io.Writer, r io.Reader) {
	if _, err := io.Copy(dst, r); err != nil {
		return
	}
}
