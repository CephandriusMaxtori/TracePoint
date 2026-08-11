package system

import (
	"context"
	"sort"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/process"

	"tracepoint/internal/state"
)

type Collector struct {
	st *state.Store
}

func New(st *state.Store) *Collector {
	return &Collector{st: st}
}

func (c *Collector) Run(ctx context.Context) {
	// Snapshot host info once.
	c.refresh(ctx, true)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	slow := time.NewTicker(30 * time.Second)
	defer slow.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.refresh(ctx, false)
		case <-slow.C:
			c.refresh(ctx, true)
		}
	}
}

func (c *Collector) refresh(ctx context.Context, full bool) {
	now := time.Now()
	sys := state.System{UpdatedAt: now}

	if hi, err := host.Info(); err == nil {
		sys.Hostname = hi.Hostname
		sys.OS = hi.OS
		sys.Platform = hi.Platform
		sys.Kernel = hi.KernelVersion
		sys.Arch = hi.KernelArch
		sys.UptimeSec = hi.Uptime
		sys.BootTimeSec = hi.BootTime
	}

	if la, err := load.Avg(); err == nil {
		sys.Load1, sys.Load5, sys.Load15 = la.Load1, la.Load5, la.Load15
	}

	if percents, err := cpu.PercentWithContext(ctx, 0); err == nil && len(percents) > 0 {
		var total float64
		for _, p := range percents {
			total += p
		}
		sys.CPUPercent = total / float64(len(percents))
	}
	if n, err := cpu.Counts(false); err == nil {
		sys.CPUCount = n
	}

	if vm, err := mem.VirtualMemoryWithContext(ctx); err == nil {
		sys.MemTotal = vm.Total
		sys.MemUsed = vm.Used
		sys.MemPercent = vm.UsedPercent
	}
	if sm, err := mem.SwapMemoryWithContext(ctx); err == nil {
		sys.SwapTotal = sm.Total
		sys.SwapUsed = sm.Used
	}

	if full {
		sys.Disks = disks(ctx)
		sys.Procs = topProcesses(ctx)
	}

	// Network IO rates (delta based).
	c.st.Update(func(s *state.Store) {
		s.System = sys
	})
}

func disks(ctx context.Context) []state.Disk {
	parts, err := disk.PartitionsWithContext(ctx, false)
	if err != nil {
		return nil
	}
	out := make([]state.Disk, 0, len(parts))
	for _, p := range parts {
		u, err := disk.UsageWithContext(ctx, p.Mountpoint)
		if err != nil || u.Total == 0 {
			continue
		}
		out = append(out, state.Disk{
			Mount:   p.Mountpoint,
			FSType:  p.Fstype,
			Total:   u.Total,
			Used:    u.Used,
			Percent: u.UsedPercent,
		})
	}
	return out
}

func topProcesses(ctx context.Context) []state.Proc {
	procs, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return nil
	}
	type scored struct {
		p   state.Proc
		cpu float64
	}
	scores := make([]scored, 0, len(procs))
	for _, p := range procs {
		pp, err := p.CPUPercentWithContext(ctx)
		if err != nil {
			continue
		}
		mp, err := p.MemoryPercentWithContext(ctx)
		if err != nil {
			mp = 0
		}
		name, err := p.Name()
		if err != nil {
			name = "?"
		}
		scores = append(scores, scored{
			p: state.Proc{PID: p.Pid, Name: name, CPUPercent: pp, MemPercent: mp},
			cpu: pp,
		})
	}
	sort.Slice(scores, func(i, j int) bool { return scores[i].cpu > scores[j].cpu })
	if len(scores) > 20 {
		scores = scores[:20]
	}
	out := make([]state.Proc, len(scores))
	for i, s := range scores {
		out[i] = s.p
	}
	return out
}
