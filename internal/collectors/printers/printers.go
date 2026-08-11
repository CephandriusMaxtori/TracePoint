package printers

import (
	"context"
	"encoding/json"
	"runtime"
	"strings"
	"time"

	"tracepoint/internal/osutil"
	"tracepoint/internal/state"
)

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
	c.st.Update(func(s *state.Store) {
		s.Printers = list
		if err != nil {
			s.Printers = nil
		}
	})
}

type winPrinter struct {
	Name         string `json:"Name"`
	DriverName   string `json:"DriverName"`
	PrinterStatus uint32 `json:"PrinterStatus"`
	Default      bool   `json:"Default"`
}

func List(ctx context.Context) ([]state.Printer, error) {
	switch runtime.GOOS {
	case "windows":
		return listWindows(ctx)
	default:
		return listCups(ctx)
	}
}

func listWindows(ctx context.Context) ([]state.Printer, error) {
	script := `Get-Printer | Select-Object Name, DriverName, PrinterStatus, Default | ConvertTo-Json`
	out, err := osutil.RunCaptureStd(ctx, "powershell", []string{"-NoProfile", "-NonInteractive", "-Command", script})
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	var one winPrinter
	var many []winPrinter
	if err := json.Unmarshal([]byte(out), &many); err != nil {
		if err2 := json.Unmarshal([]byte(out), &one); err2 != nil {
			return nil, err
		}
		many = []winPrinter{one}
	}
	var list []state.Printer
	for _, p := range many {
		status := "unknown"
		switch p.PrinterStatus {
		case 3:
			status = "idle"
		case 4:
			status = "printing"
		case 5:
			status = "warmup"
		case 6:
			status = "stopped"
		case 7:
			status = "offline"
		case 8:
			status = "error"
		}
		list = append(list, state.Printer{
			Name:    p.Name,
			Driver:  p.DriverName,
			Status:  status,
			Default: p.Default,
		})
	}
	return list, nil
}

func listCups(ctx context.Context) ([]state.Printer, error) {
	out, err := osutil.RunCaptureStd(ctx, "lpstat", []string{"-p", "-e"})
	if err != nil {
		return nil, err
	}
	statuses := map[string]string{}
	for _, line := range strings.Split(out, "\n") {
		if !strings.HasPrefix(line, "printer ") {
			continue
		}
		rest := strings.TrimPrefix(line, "printer ")
		parts := strings.SplitN(rest, " ", 2)
		if len(parts) < 2 {
			continue
		}
		statuses[parts[0]] = parts[1]
	}
	defaultPrinter := ""
	if d, err := osutil.RunCaptureStd(ctx, "lpstat", []string{"-d"}); err == nil {
		// "system default destination: MyPrinter"
		if idx := strings.LastIndex(d, ": "); idx >= 0 {
			defaultPrinter = strings.TrimSpace(d[idx+2:])
		}
	}
	var list []state.Printer
	for name, st := range statuses {
		status := "unknown"
		low := strings.ToLower(st)
		switch {
		case strings.Contains(low, "idle"):
			status = "idle"
		case strings.Contains(low, "disabled") || strings.Contains(low, "stopped") || strings.Contains(low, "down"):
			status = "stopped"
		case strings.Contains(low, "printing") || strings.Contains(low, "active"):
			status = "printing"
		case strings.Contains(low, "error") || strings.Contains(low, "offline"):
			status = "error"
		}
		list = append(list, state.Printer{
			Name:    name,
			Status:  status,
			Default: name == defaultPrinter,
		})
	}
	return list, nil
}
