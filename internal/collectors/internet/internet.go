package internet

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"tracepoint/internal/state"
)

type Collector struct {
	st     *state.Store
	client *http.Client
}

func New(st *state.Store) *Collector {
	return &Collector{
		st: st,
		client: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				DialContext: (&net.Dialer{Timeout: 4 * time.Second}).DialContext,
			},
		},
	}
}

func (c *Collector) Run(ctx context.Context) {
	c.Check(ctx)
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			c.Check(ctx)
		}
	}
}

func (c *Collector) Check(ctx context.Context) {
	var checks []state.CheckResult

	checks = append(checks, c.checkTCP(ctx, "Cloudflare DNS", "1.1.1.1:53"))
	checks = append(checks, c.checkTCP(ctx, "Google DNS", "8.8.8.8:53"))
	checks = append(checks, c.checkDNS(ctx, "DNS resolution (google.com)"))
	checks = append(checks, c.checkHTTPS(ctx, "HTTPS (google.com)", "https://www.google.com"))

	c.st.Update(func(s *state.Store) {
		s.Internet = state.Internet{Checks: checks, UpdatedAt: time.Now()}
	})
}

func (c *Collector) checkTCP(ctx context.Context, name, addr string) state.CheckResult {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", addr, 4*time.Second)
	lat := float64(time.Since(start).Microseconds()) / 1000
	res := state.CheckResult{Name: name, Latency: lat}
	if err != nil {
		res.Status = state.StatusFail
		res.Detail = err.Error()
		return res
	}
	conn.Close()
	res.Status = state.StatusOK
	res.Detail = "reachable"
	return res
}

func (c *Collector) checkDNS(ctx context.Context, name string) state.CheckResult {
	start := time.Now()
	_, err := net.DefaultResolver.LookupHost(ctx, "google.com")
	lat := float64(time.Since(start).Microseconds()) / 1000
	res := state.CheckResult{Name: name, Latency: lat}
	if err != nil {
		res.Status = state.StatusFail
		res.Detail = err.Error()
		return res
	}
	res.Status = state.StatusOK
	res.Detail = "resolved"
	return res
}

func (c *Collector) checkHTTPS(ctx context.Context, name, url string) state.CheckResult {
	start := time.Now()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := c.client.Do(req)
	lat := float64(time.Since(start).Microseconds()) / 1000
	res := state.CheckResult{Name: name, Latency: lat}
	if err != nil {
		res.Status = state.StatusFail
		res.Detail = err.Error()
		return res
	}
	defer resp.Body.Close()
	res.Status = state.StatusOK
	if resp.StatusCode >= 500 {
		res.Status = state.StatusWarn
	}
	res.Detail = fmt.Sprintf("HTTP %d", resp.StatusCode)
	return res
}

func Overall(checks []state.CheckResult) state.Status {
	if len(checks) == 0 {
		return state.StatusUnknown
	}
	ok := 0
	for _, ch := range checks {
		if ch.Status == state.StatusOK {
			ok++
		}
	}
	switch {
	case ok == len(checks):
		return state.StatusOK
	case ok >= 1:
		return state.StatusWarn
	default:
		return state.StatusFail
	}
}
