package network

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Chrisyhjiang/nmap-go/pkg"
)

// ImportantPorts is the default set of commonly-scanned service ports,
// provided by github.com/Chrisyhjiang/nmap-go (MIT).
var ImportantPorts = pkg.ImportantPorts

// ParsePortSpec parses an nmap-style port specification such as "22,80,443",
// "1-1000" or "22,80,1000-2000". An empty spec returns the common ports set.
func ParsePortSpec(spec string) ([]int, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		out := make([]int, 0, len(ImportantPorts))
		for p := range ImportantPorts {
			if p > 0 {
				out = append(out, p)
			}
		}
		sort.Ints(out)
		return out, nil
	}
	seen := map[int]bool{}
	var out []int
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "-") {
			bounds := strings.SplitN(part, "-", 2)
			lo, err1 := strconv.Atoi(strings.TrimSpace(bounds[0]))
			hi, err2 := strconv.Atoi(strings.TrimSpace(bounds[1]))
			if err1 != nil || err2 != nil || lo < 1 || hi > 65535 || lo > hi {
				return nil, fmt.Errorf("invalid port range %q", part)
			}
			for p := lo; p <= hi; p++ {
				if !seen[p] {
					seen[p] = true
					out = append(out, p)
				}
			}
		} else {
			p, err := strconv.Atoi(part)
			if err != nil || p < 1 || p > 65535 {
				return nil, fmt.Errorf("invalid port %q", part)
			}
			if !seen[p] {
				seen[p] = true
				out = append(out, p)
			}
		}
	}
	sort.Ints(out)
	return out, nil
}

// PortScanConcurrent is an nmap-style TCP connect scan using a bounded worker
// pool, adapted from github.com/Chrisyhjiang/nmap-go (MIT). Progress and open
// ports are reported through the optional log callback.
func PortScanConcurrent(ctx context.Context, host string, ports []int, log func(format string, args ...any)) ([]PortResult, error) {
	if len(ports) == 0 {
		return nil, fmt.Errorf("no ports to scan")
	}
	workers := runtime.NumCPU() * 4
	if workers > 100 {
		workers = 100
	}
	jobs := make(chan int)
	var (
		mu      sync.Mutex
		seen    int
		left    = len(ports)
		wg      sync.WaitGroup
		results []PortResult
	)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for port := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
				}
				open := isPortOpen(ctx, host, port)
				mu.Lock()
				seen++
				n := seen
				if open {
					results = append(results, PortResult{Port: port, Open: true, Service: serviceName(port)})
				}
				mu.Unlock()
				if log != nil {
					if open {
						log("OPEN  %5d/tcp  %s", port, serviceName(port))
					}
					if n%50 == 0 {
						log("scanning... %d/%d ports", n, left)
					}
				}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for _, p := range ports {
			select {
			case <-ctx.Done():
				return
			case jobs <- p:
			}
		}
	}()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-ctx.Done():
		return results, ctx.Err()
	case <-done:
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Port < results[j].Port })
	return results, nil
}

func isPortOpen(ctx context.Context, host string, port int) bool {
	d := net.Dialer{Timeout: 700 * time.Millisecond}
	conn, err := d.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
