//go:build darwin

package services

import (
	"context"
	"strings"

	"tracepoint/internal/osutil"
	"tracepoint/internal/state"
)

type darwinBackend struct{}

func platformBackend() Backend { return darwinBackend{} }

func (darwinBackend) List(ctx context.Context) ([]state.Service, error) {
	out, err := osutil.RunCaptureStd(ctx, "launchctl", []string{"list"})
	if err != nil {
		return nil, err
	}
	var list []state.Service
	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(line)
		if len(f) < 3 {
			continue
		}
		stateName := "stopped"
		if f[0] != "-" {
			stateName = "running"
		}
		list = append(list, state.Service{
			Name:        f[2],
			DisplayName: f[2],
			State:       stateName,
		})
	}
	return list, nil
}

func (darwinBackend) Start(ctx context.Context, name string) error {
	return osutil.RunStream(ctx, "launchctl", []string{"start", name}, nil)
}

func (darwinBackend) Stop(ctx context.Context, name string) error {
	return osutil.RunStream(ctx, "launchctl", []string{"stop", name}, nil)
}

func (darwinBackend) Restart(ctx context.Context, name string) error {
	_ = osutil.RunStream(ctx, "launchctl", []string{"stop", name}, nil)
	return osutil.RunStream(ctx, "launchctl", []string{"start", name}, nil)
}
