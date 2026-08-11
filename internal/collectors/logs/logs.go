package logs

import (
	"bufio"
	"context"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

const maxLines = 3000

// Tailer follows a file, keeping a ring buffer of recent lines. It tolerates
// the file being replaced (rotation) by re-opening on size decrease.
type Tailer struct {
	mu     sync.Mutex
	path   string
	lines  []string
	filter string
	err    string
	follow bool

	cancel context.CancelFunc
	active bool
}

func NewTailer(path string, filter string) *Tailer {
	return &Tailer{path: path, filter: filter}
}

func (t *Tailer) Path() string { return t.path }

func (t *Tailer) SetPath(path string) {
	t.mu.Lock()
	t.path = path
	t.lines = nil
	t.err = ""
	t.mu.Unlock()
}

func (t *Tailer) SetFilter(f string) {
	t.mu.Lock()
	t.filter = f
	t.mu.Unlock()
}

func (t *Tailer) Filter() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.filter
}

func (t *Tailer) Active() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.active
}

func (t *Tailer) Err() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.err
}

func (t *Tailer) Lines() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]string, len(t.lines))
	copy(out, t.lines)
	return out
}

func (t *Tailer) append(line string) {
	f := t.filter
	if f != "" && !strings.Contains(line, f) {
		return
	}
	t.lines = append(t.lines, line)
	if len(t.lines) > maxLines {
		t.lines = t.lines[len(t.lines)-maxLines:]
	}
}

func (t *Tailer) Start(ctx context.Context) {
	t.mu.Lock()
	if t.active {
		t.mu.Unlock()
		return
	}
	t.active = true
	t.mu.Unlock()
	go t.run(ctx)
}

func (t *Tailer) Stop() {
	t.mu.Lock()
	t.active = false
	t.mu.Unlock()
}

func (t *Tailer) Clear() {
	t.mu.Lock()
	t.lines = nil
	t.mu.Unlock()
}

func (t *Tailer) run(ctx context.Context) {
	path := t.Path()
	f, err := os.Open(path)
	if err != nil {
		t.mu.Lock()
		t.err = err.Error()
		t.mu.Unlock()
		return
	}
	defer f.Close()

	// Seed with the last ~1MB of the file so we start near the tail.
	if st, err := f.Stat(); err == nil {
		start := int64(0)
		if st.Size() > 1<<20 {
			start = st.Size() - 1<<20
		}
		f.Seek(start, io.SeekStart)
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for sc.Scan() {
			t.mu.Lock()
			t.append(sc.Text())
			t.mu.Unlock()
		}
	}

	// Follow for appended bytes.
	lastSize, _ := f.Seek(0, io.SeekEnd)
	buf := make([]byte, 32*1024)
	lineBuf := make([]byte, 0, 4096)
	tick := time.NewTicker(400 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
		t.mu.Lock()
		active := t.active
		path = t.path
		t.mu.Unlock()
		if !active {
			return
		}
		_ = path

		st, err := f.Stat()
		if err != nil {
			continue
		}
		if st.Size() < lastSize {
			// File rotated/truncated: reopen.
			t.mu.Lock()
			t.lines = nil
			t.mu.Unlock()
			f2, err := os.Open(t.Path())
			if err != nil {
				t.mu.Lock()
				t.err = err.Error()
				t.mu.Unlock()
				return
			}
			f.Close()
			f = f2
			lastSize, _ = f.Seek(0, io.SeekEnd)
			lineBuf = lineBuf[:0]
			continue
		}
		if st.Size() == lastSize {
			continue
		}
		n, err := f.ReadAt(buf, lastSize)
		if n > 0 {
			lastSize += int64(n)
			for _, b := range buf[:n] {
				if b == '\n' {
					t.mu.Lock()
					t.append(string(lineBuf))
					t.mu.Unlock()
					lineBuf = lineBuf[:0]
				} else {
					lineBuf = append(lineBuf, b)
				}
			}
		}
		if err != nil && err != io.EOF {
			return
		}
	}
}
