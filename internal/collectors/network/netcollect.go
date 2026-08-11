package network

import (
	"context"
	"slices"
	"time"

	gsnet "github.com/shirou/gopsutil/v4/net"

	"tracepoint/internal/state"
)

// Collector samples network interfaces and pushes a history series used by
// the overview sparklines.
type Collector struct {
	st      *state.Store
	prev    map[string]gsnet.IOCountersStat
	updated time.Time
}

func New(st *state.Store) *Collector {
	return &Collector{st: st, prev: map[string]gsnet.IOCountersStat{}}
}

func (c *Collector) Run(ctx context.Context) {
	c.sample(ctx)
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			c.sample(ctx)
		}
	}
}

func (c *Collector) sample(ctx context.Context) {
	counters, _ := gsnet.IOCountersWithContext(ctx, true)
	now := time.Now()
	dt := now.Sub(c.updated).Seconds()

	infs, _ := gsnet.Interfaces()
	byName := map[string]gsnet.InterfaceStat{}
	for _, inf := range infs {
		byName[inf.Name] = inf
	}

	up := map[string]bool{}
	var totalRx, totalTx float64
	var ifaces []state.NetIface
	for _, ct := range counters {
		var rxb, txb float64
		if p, ok := c.prev[ct.Name]; ok && dt > 0 {
			rxb = float64(ct.BytesRecv-p.BytesRecv) / dt
			txb = float64(ct.BytesSent-p.BytesSent) / dt
		}
		c.prev[ct.Name] = ct
		totalRx += rxb
		totalTx += txb
		iface := state.NetIface{
			Name:    ct.Name,
			RxBps:   rxb,
			TxBps:   txb,
			RxTotal: ct.BytesRecv,
			TxTotal: ct.BytesSent,
		}
		if inf, ok := byName[ct.Name]; ok {
			iface.Up = ifaceUp(inf)
			iface.MTU = inf.MTU
			for _, a := range inf.Addrs {
				iface.Addrs = append(iface.Addrs, a.Addr)
			}
			up[ct.Name] = true
		}
		ifaces = append(ifaces, iface)
	}
	c.updated = now

	// Interfaces with no traffic yet.
	for _, inf := range infs {
		if up[inf.Name] {
			continue
		}
		iface := state.NetIface{Name: inf.Name, Up: ifaceUp(inf), MTU: inf.MTU}
		for _, a := range inf.Addrs {
			iface.Addrs = append(iface.Addrs, a.Addr)
		}
		ifaces = append(ifaces, iface)
	}

	c.st.Update(func(s *state.Store) {
		s.Net = ifaces
	})

	// Push history based on latest system snapshot.
	var cpuPct, memPct float64
	c.st.Read(func(s *state.Store) {
		cpuPct = s.System.CPUPercent
		memPct = s.System.MemPercent
	})
	c.st.PushHist(cpuPct, memPct, totalRx, totalTx)
}

func ifaceUp(inf gsnet.InterfaceStat) bool {
	return slices.Contains(inf.Flags, "up")
}
