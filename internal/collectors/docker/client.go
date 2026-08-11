package docker

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"time"

	winio "github.com/Microsoft/go-winio"
)

// Client is a minimal Docker Engine API client over the local unix socket
// (Linux) or named pipe (Windows). No CGO required.
type Client struct {
	httpc *http.Client
	base  string
}

// NewClient creates a Docker Engine API client over the local unix socket
// (Linux) or named pipe (Windows). No CGO required.
func NewClient() *Client {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			if runtime.GOOS == "windows" {
				return winio.DialPipeContext(ctx, `\\.\pipe\docker_engine`)
			}
			var d net.Dialer
			return d.DialContext(ctx, "unix", "/var/run/docker.sock")
		},
	}
	return &Client{
		httpc: &http.Client{Transport: transport, Timeout: 15 * time.Second},
		base:  "http://docker",
	}
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, body io.Reader) (*http.Response, error) {
	u := c.base + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpc.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) Ping(ctx context.Context) (string, error) {
	resp, err := c.do(ctx, http.MethodGet, "/_ping", nil, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("docker ping: HTTP %d", resp.StatusCode)
	}
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return string(b), nil
}

func (c *Client) Version(ctx context.Context) (string, error) {
	resp, err := c.do(ctx, http.MethodGet, "/version", nil, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var v struct {
		Version string `json:"Version"`
		APIVer  string `json:"ApiVersion"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s (API %s)", v.Version, v.APIVer), nil
}

type Container struct {
	ID     string   `json:"Id"`
	Names  []string `json:"Names"`
	Image  string   `json:"Image"`
	State  string   `json:"State"`
	Status string   `json:"Status"`
}

func (c *Client) List(ctx context.Context, all bool) ([]Container, error) {
	q := url.Values{}
	if all {
		q.Set("all", "1")
	}
	resp, err := c.do(ctx, http.MethodGet, "/containers/json", q, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list containers: HTTP %d", resp.StatusCode)
	}
	var out []Container
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

type Stats struct {
	CPU    float64
	MemPct float64
}

// StatsFor computes CPU% against the previous sample cached on the client.
func (c *Client) StatsFor(ctx context.Context, id string, prev *statsSample) (Stats, *statsSample, error) {
	q := url.Values{}
	q.Set("stream", "false")
	q.Set("one-shot", "1")
	resp, err := c.do(ctx, http.MethodGet, "/containers/"+id+"/stats", q, nil)
	if err != nil {
		return Stats{}, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Stats{}, nil, fmt.Errorf("stats %s: HTTP %d", id, resp.StatusCode)
	}
	var st struct {
		CPUStats cpuStats `json:"cpu_stats"`
		PreCPU   cpuStats `json:"precpu_stats"`
		Memory   memStats `json:"memory_stats"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&st); err != nil {
		return Stats{}, nil, err
	}
	cur := &statsSample{total: st.CPUStats.Usage.Total, system: st.CPUStats.SystemUsage, online: st.CPUStats.OnlineCPUs}
	var s Stats
	if prev != nil && prev.system > 0 && cur.system > prev.system && cur.total > 0 {
		dCpu := cur.total - prev.total
		dSys := cur.system - prev.system
		if dSys > 0 {
			online := cur.online
			if online == 0 {
				online = 1
			}
			s.CPU = float64(dCpu) / float64(dSys) * float64(online) * 100
		}
	}
	if st.Memory.Limit > 0 {
		s.MemPct = float64(st.Memory.Usage) / float64(st.Memory.Limit) * 100
	}
	return s, cur, nil
}

type statsSample struct {
	total  uint64
	system uint64
	online uint32
}

type cpuStats struct {
	Usage struct {
		Total uint64 `json:"total_usage"`
	} `json:"cpu_usage"`
	SystemUsage uint64 `json:"system_cpu_usage"`
	OnlineCPUs  uint32 `json:"online_cpus"`
}

type memStats struct {
	Usage uint64 `json:"usage"`
	Limit uint64 `json:"limit"`
}

func (c *Client) Logs(ctx context.Context, id string, tail int) (string, error) {
	q := url.Values{}
	q.Set("stdout", "1")
	q.Set("stderr", "1")
	q.Set("timestamps", "1")
	if tail > 0 {
		q.Set("tail", fmt.Sprintf("%d", tail))
	} else {
		q.Set("tail", "200")
	}
	resp, err := c.do(ctx, http.MethodGet, "/containers/"+id+"/logs", q, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("logs %s: HTTP %d", id, resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", err
	}
	return demuxLogs(data), nil
}

func (c *Client) Start(ctx context.Context, id string) error {
	return c.action(ctx, id, "start")
}

func (c *Client) Stop(ctx context.Context, id string) error {
	q := url.Values{}
	q.Set("t", "10")
	return c.action(ctx, id, "stop", q)
}

func (c *Client) Restart(ctx context.Context, id string) error {
	q := url.Values{}
	q.Set("t", "5")
	return c.action(ctx, id, "restart", q)
}

func (c *Client) action(ctx context.Context, id, action string, qs ...url.Values) error {
	var q url.Values
	if len(qs) > 0 {
		q = qs[0]
	}
	resp, err := c.do(ctx, http.MethodPost, "/containers/"+id+"/"+action, q, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotModified {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("%s %s: HTTP %d: %s", action, id, resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

// demuxLogs strips the Docker multiplexed stream headers from raw log bytes.
func demuxLogs(data []byte) string {
	var b strings.Builder
	for len(data) > 8 {
		header := data[:8]
		size := binary.BigEndian.Uint32(header[4:8])
		if int(size) > len(data)-8 {
			break
		}
		payload := data[8 : 8+size]
		b.Write(payload)
		data = data[8+size:]
	}
	if len(data) > 0 {
		b.Write(data)
	}
	return b.String()
}
