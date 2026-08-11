package network

import (
	"context"
	"fmt"
	"net"
	"sort"
	"time"

	probing "github.com/prometheus-community/pro-bing"
)

// PingResult is the outcome of an ICMP ping.
type PingResult struct {
	Host    string
	Sent    int
	Recv    int
	LossPct float64
	MinMs   float64
	AvgMs   float64
	MaxMs   float64
	Err     string
}

// Ping sends count ICMP pings to host. On platforms requiring raw sockets
// without privileges, the returned error explains the situation.
func Ping(ctx context.Context, host string, count int) (*PingResult, error) {
	p, err := probing.NewPinger(host)
	if err != nil {
		return nil, err
	}
	p.Count = count
	p.Timeout = time.Duration(count) * 2 * time.Second
	res := &PingResult{Host: host}
	var durations []time.Duration
	p.OnRecv = func(pkt *probing.Packet) {
		durations = append(durations, pkt.Rtt)
	}
	err = p.RunWithContext(ctx)
	if err != nil {
		return nil, err
	}
	res.Sent = p.PacketsSent
	res.Recv = p.PacketsRecv
	if res.Sent > 0 {
		res.LossPct = float64(res.Sent-res.Recv) / float64(res.Sent) * 100
	}
	if len(durations) > 0 {
		sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
		res.MinMs = float64(durations[0].Microseconds()) / 1000
		res.MaxMs = float64(durations[len(durations)-1].Microseconds()) / 1000
		var sum time.Duration
		for _, d := range durations {
			sum += d
		}
		res.AvgMs = float64(sum.Microseconds()) / float64(len(durations)) / 1000
	}
	return res, nil
}

type PortResult struct {
	Port    int
	Open    bool
	Service string
}

func PortScan(ctx context.Context, host string, start, end int) ([]PortResult, error) {
	if start < 1 {
		start = 1
	}
	if end > 65535 {
		end = 65535
	}
	if start > end {
		return nil, fmt.Errorf("invalid port range %d-%d", start, end)
	}
	var out []PortResult
	for p := start; p <= end; p++ {
		select {
		case <-ctx.Done():
			return out, ctx.Err()
		default:
		}
		addr := fmt.Sprintf("%s:%d", host, p)
		conn, err := net.DialTimeout("tcp", addr, 700*time.Millisecond)
		if err == nil {
			conn.Close()
			out = append(out, PortResult{Port: p, Open: true, Service: serviceName(p)})
		}
	}
	return out, nil
}

func LookupHost(ctx context.Context, host string) ([]string, error) {
	return net.DefaultResolver.LookupHost(ctx, host)
}

func LookupAddr(ctx context.Context, ip string) ([]string, error) {
	return net.DefaultResolver.LookupAddr(ctx, ip)
}

func LookupMX(ctx context.Context, host string) ([]string, error) {
	mx, err := net.DefaultResolver.LookupMX(ctx, host)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(mx))
	for _, m := range mx {
		out = append(out, fmt.Sprintf("%s (prio %d)", m.Host, m.Pref))
	}
	return out, nil
}

func serviceName(port int) string {
	names := map[int]string{
		21: "FTP", 22: "SSH", 23: "Telnet", 25: "SMTP", 53: "DNS",
		80: "HTTP", 110: "POP3", 111: "RPC", 123: "NTP", 135: "RPC/DCOM",
		139: "NetBIOS", 143: "IMAP", 443: "HTTPS", 445: "SMB", 465: "SMTPS",
		514: "Syslog", 587: "SMTP", 631: "IPP", 993: "IMAPS", 995: "POP3S",
		1433: "MSSQL", 1521: "Oracle", 2049: "NFS", 2375: "Docker", 2376: "Docker TLS",
		3306: "MySQL", 3389: "RDP", 5432: "PostgreSQL", 5900: "VNC", 6379: "Redis",
		8080: "HTTP-Alt", 8443: "HTTPS-Alt", 9092: "Kafka", 9200: "Elasticsearch",
		27017: "MongoDB", 3333: "KMS",
	}
	if s, ok := names[port]; ok {
		return s
	}
	return ""
}
