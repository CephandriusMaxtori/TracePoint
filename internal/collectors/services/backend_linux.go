//go:build linux

package services

import (
	"context"
	"strings"

	"tracepoint/internal/osutil"
	"tracepoint/internal/state"
)

type linuxBackend struct{}

func platformBackend() Backend { return linuxBackend{} }

func (linuxBackend) List(ctx context.Context) ([]state.Service, error) {
	units, err := osutil.RunCaptureStd(ctx, "systemctl", []string{"list-units", "--type=service", "--all", "--no-legend", "--no-pager", "--plain"})
	if err != nil {
		return nil, err
	}
	enabled := map[string]bool{}
	if files, err := osutil.RunCaptureStd(ctx, "systemctl", []string{"list-unit-files", "--type=service", "--no-legend", "--no-pager"}); err == nil {
		for _, line := range strings.Split(files, "\n") {
			f := strings.Fields(line)
			if len(f) >= 2 && strings.HasSuffix(f[0], ".service") {
				enabled[f[0]] = f[1] == "enabled" || f[1] == "static"
			}
		}
	}
	var out []state.Service
	for _, line := range strings.Split(units, "\n") {
		f := strings.Fields(line)
		if len(f) < 4 {
			continue
		}
		out = append(out, state.Service{
			Name:        f[0],
			DisplayName: strings.Join(f[4:], " "),
			State:       f[3],
			Enabled:     enabled[f[0]],
		})
	}
	return out, nil
}

func (linuxBackend) Start(ctx context.Context, name string) error {
	return osutil.RunStream(ctx, "systemctl", []string{"start", unit(name)}, nil)
}

func (linuxBackend) Stop(ctx context.Context, name string) error {
	return osutil.RunStream(ctx, "systemctl", []string{"stop", unit(name)}, nil)
}

func (linuxBackend) Restart(ctx context.Context, name string) error {
	return osutil.RunStream(ctx, "systemctl", []string{"restart", unit(name)}, nil)
}

func unit(name string) string {
	if strings.HasSuffix(name, ".service") {
		return name
	}
	return name + ".service"
}
