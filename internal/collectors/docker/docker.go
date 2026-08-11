package docker

import (
	"context"
	"strings"
	"time"

	"tracepoint/internal/state"
)

type Collector struct {
	st     *state.Store
	client *Client
	prev   map[string]*statsSample
}

func New(st *state.Store) *Collector {
	return &Collector{
		st:     st,
		client: New(),
		prev:   map[string]*statsSample{},
	}
}

func (c *Collector) Run(ctx context.Context) {
	c.refresh(ctx)
	t := time.NewTicker(5 * time.Second)
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
	res := state.Docker{UpdatedAt: time.Now()}
	ver, err := c.client.Version(ctx)
	if err != nil {
		res.Connected = false
		res.Err = err.Error()
	} else {
		res.Connected = true
		res.Version = ver
		containers, err := c.client.List(ctx, true)
		if err != nil {
			res.Err = err.Error()
		}
		for _, ct := range containers {
			id := ct.ID
			name := ""
			if len(ct.Names) > 0 {
				name = strings.TrimPrefix(ct.Names[0], "/")
			}
			dc := state.DockerContainer{
				ID:      id[:12],
				Name:    name,
				Image:   ct.Image,
				State:   ct.State,
				Status:  ct.Status,
				Running: ct.State == "running",
			}
			if ct.State == "running" {
				if st, cur, err := c.client.StatsFor(ctx, id, c.prev[id]); err == nil {
					dc.CPU = st.CPU
					dc.MemPct = st.MemPct
					c.prev[id] = cur
				}
			} else {
				delete(c.prev, id)
			}
			res.Containers = append(res.Containers, dc)
		}
	}
	c.st.Update(func(s *state.Store) {
		s.Docker = res
	})
}

// Actions
func (c *Collector) Start(ctx context.Context, id string) error {
	return c.client.Start(ctx, id)
}

func (c *Collector) Stop(ctx context.Context, id string) error {
	return c.client.Stop(ctx, id)
}

func (c *Collector) Restart(ctx context.Context, id string) error {
	return c.client.Restart(ctx, id)
}

func (c *Collector) Logs(ctx context.Context, id string, tail int) (string, error) {
	return c.client.Logs(ctx, id, tail)
}
