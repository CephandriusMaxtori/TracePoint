package actions

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Status string

const (
	StatusRunning Status = "running"
	StatusDone    Status = "done"
	StatusError   Status = "error"
)

type Op struct {
	ID       string
	Label    string
	Status   Status
	Log      []string
	Started  time.Time
	Finished time.Time
	Err      error
}

type Manager struct {
	mu     sync.Mutex
	ops    []*Op
	limit  int
	nextID int
	notify func()
	muBg   sync.Mutex
	bg     []*Op
}

func NewManager(notify func()) *Manager {
	return &Manager{limit: 50, notify: notify}
}

func (m *Manager) Ops() []*Op {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*Op, len(m.ops))
	copy(out, m.ops)
	return out
}

func (m *Manager) Running() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, o := range m.ops {
		if o.Status == StatusRunning {
			return true
		}
	}
	return false
}

func (m *Manager) add(op *Op) {
	m.mu.Lock()
	m.nextID++
	op.ID = fmt.Sprintf("op-%d", m.nextID)
	m.ops = append(m.ops, op)
	if len(m.ops) > m.limit {
		m.ops = m.ops[len(m.ops)-m.limit:]
	}
	m.mu.Unlock()
	m.notify()
}

func (m *Manager) appendLog(op *Op, line string) {
	m.mu.Lock()
	op.Log = append(op.Log, line)
	if len(op.Log) > 2000 {
		op.Log = op.Log[len(op.Log)-2000:]
	}
	m.mu.Unlock()
	m.notify()
}

func (m *Manager) finish(op *Op, err error) {
	m.mu.Lock()
	op.Finished = time.Now()
	if err != nil {
		op.Status = StatusError
		op.Err = err
	} else {
		op.Status = StatusDone
	}
	m.mu.Unlock()
	m.notify()
}

func (m *Manager) Log(op *Op, format string, args ...any) {
	line := fmt.Sprintf(format, args...)
	m.appendLog(op, line)
}

// Run executes fn in a goroutine. fn receives a context that is cancelled if
// the operation is aborted, and a log callback for streaming output lines.
func (m *Manager) Run(label string, fn func(ctx context.Context, log func(format string, args ...any))) {
	op := &Op{Label: label, Status: StatusRunning, Started: time.Now()}
	m.add(op)
	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		defer func() {
			if r := recover(); r != nil {
				m.finish(op, fmt.Errorf("panic: %v", r))
			}
		}()
		fn(ctx, func(format string, args ...any) {
			m.Log(op, format, args...)
		})
		m.finish(op, op.Err)
	}()
}

func (m *Manager) RunErr(label string, fn func(ctx context.Context, log func(format string, args ...any)) error) {
	op := &Op{Label: label, Status: StatusRunning, Started: time.Now()}
	m.add(op)
	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		var err error
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic: %v", r)
			}
			m.finish(op, err)
		}()
		err = fn(ctx, func(format string, args ...any) {
			m.Log(op, format, args...)
		})
	}()
}
