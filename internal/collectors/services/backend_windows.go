//go:build windows

package services

import (
	"context"
	"fmt"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"

	"tracepoint/internal/state"
)

type winBackend struct{}

func platformBackend() Backend { return winBackend{} }

func stateName(st svc.State) string {
	switch st {
	case windows.SERVICE_RUNNING:
		return "running"
	case windows.SERVICE_STOPPED:
		return "stopped"
	case windows.SERVICE_START_PENDING:
		return "starting"
	case windows.SERVICE_STOP_PENDING:
		return "stopping"
	case windows.SERVICE_PAUSED:
		return "paused"
	default:
		return "unknown"
	}
}

func (winBackend) List(ctx context.Context) ([]state.Service, error) {
	m, err := mgr.Connect()
	if err != nil {
		return nil, fmt.Errorf("connect SCM: %w", err)
	}
	defer m.Disconnect()
	names, err := m.ListServices()
	if err != nil {
		return nil, err
	}
	out := make([]state.Service, 0, len(names))
	for _, name := range names {
		select {
		case <-ctx.Done():
			return out, ctx.Err()
		default:
		}
		s, err := m.OpenService(name)
		if err != nil {
			continue
		}
		cfg, err1 := s.Config()
		st, err2 := s.Query()
		s.Close()
		if err1 != nil || err2 != nil {
			continue
		}
		out = append(out, state.Service{
			Name:        name,
			DisplayName: cfg.DisplayName,
			Description: cfg.Description,
			State:       stateName(st.State),
			Enabled:     cfg.StartType != windows.SERVICE_DISABLED,
		})
	}
	return out, nil
}

func (winBackend) Start(ctx context.Context, name string) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(name)
	if err != nil {
		return err
	}
	defer s.Close()
	return s.Start()
}

func (winBackend) Stop(ctx context.Context, name string) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(name)
	if err != nil {
		return err
	}
	defer s.Close()
	_, err = s.Control(svc.Stop)
	return err
}

func (winBackend) Restart(ctx context.Context, name string) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(name)
	if err != nil {
		return err
	}
	defer s.Close()
	_, err = s.Control(svc.Stop)
	if err != nil {
		return err
	}
	return s.Start()
}
