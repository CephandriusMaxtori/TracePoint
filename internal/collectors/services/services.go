package services

import (
	"context"
	"time"

	"tracepoint/internal/state"
)

type Backend interface {
	List(ctx context.Context) ([]state.Service, error)
	Start(ctx context.Context, name string) error
	Stop(ctx context.Context, name string) error
	Restart(ctx context.Context, name string) error
}

var backend Backend

func SetBackend(b Backend) { backend = b }

type Collector struct {
	st *state.Store
}

func New(st *state.Store) *Collector {
	return &Collector{st: st}
}

func (c *Collector) Run(ctx context.Context) {
	c.refresh(ctx)
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			c.refresh(ctx)
		}
	}
}

func (c *Collector) refresh(ctx context.Context) {
	list, err := List(ctx)
	if err != nil {
		return
	}
	c.st.Update(func(s *state.Store) {
		s.Services = list
	})
}

func List(ctx context.Context) ([]state.Service, error) {
	if backend == nil {
		return nil, errNoBackend
	}
	return backend.List(ctx)
}

func Start(ctx context.Context, name string) error {
	if backend == nil {
		return errNoBackend
	}
	return backend.Start(ctx, name)
}

func Stop(ctx context.Context, name string) error {
	if backend == nil {
		return errNoBackend
	}
	return backend.Stop(ctx, name)
}

func Restart(ctx context.Context, name string) error {
	if backend == nil {
		return errNoBackend
	}
	return backend.Restart(ctx, name)
}
