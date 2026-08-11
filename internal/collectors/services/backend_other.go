//go:build !windows && !linux && !darwin

package services

import (
	"context"

	"tracepoint/internal/state"
)

type otherBackend struct{}

func platformBackend() Backend { return otherBackend{} }

func (otherBackend) List(ctx context.Context) ([]state.Service, error) {
	return nil, errNoBackend
}
func (otherBackend) Start(ctx context.Context, name string) error { return errNoBackend }
func (otherBackend) Stop(ctx context.Context, name string) error  { return errNoBackend }
func (otherBackend) Restart(ctx context.Context, name string) error {
	return errNoBackend
}
