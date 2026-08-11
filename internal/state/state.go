package state

import (
	"sync"
	"time"
)

type Status int

const (
	StatusUnknown Status = iota
	StatusOK
	StatusWarn
	StatusFail
)

func (s Status) String() string {
	switch s {
	case StatusOK:
		return "OK"
	case StatusWarn:
		return "Warning"
	case StatusFail:
		return "Failed"
	default:
		return "Unknown"
	}
}

type System struct {
	Hostname    string
	OS          string
	Platform    string
	Kernel      string
	Arch        string
	UptimeSec   uint64
	BootTimeSec uint64
	Load1       float64
	Load5       float64
	Load15      float64
	CPUPercent  float64
	CPUCount    int
	MemTotal    uint64
	MemUsed     uint64
	MemPercent  float64
	SwapTotal   uint64
	SwapUsed    uint64
	Disks       []Disk
	Procs       []Proc
	UpdatedAt   time.Time
}

type Disk struct {
	Mount   string
	FSType  string
	Total   uint64
	Used    uint64
	Percent float64
}

type Proc struct {
	PID        int32
	Name       string
	CPUPercent float64
	MemPercent float64
}

type NetIface struct {
	Name    string
	Addrs   []string
	Up      bool
	MTU     int
	RxBps   float64
	TxBps   float64
	RxTotal uint64
	TxTotal uint64
}

type CheckResult struct {
	Name    string
	Status  Status
	Detail  string
	Latency float64
}

type Internet struct {
	Checks    []CheckResult
	UpdatedAt time.Time
}

type DockerContainer struct {
	ID      string
	Name    string
	Image   string
	State   string
	Status  string
	CPU     float64
	MemPct  float64
	Running bool
}

type Docker struct {
	Connected  bool
	Version    string
	Containers []DockerContainer
	Err        string
	UpdatedAt  time.Time
}

type Printer struct {
	Name    string
	Status  string
	Default bool
	Driver  string
}

type Service struct {
	Name        string
	DisplayName string
	State       string
	Enabled     bool
	Description string
}

type App struct {
	Name     string
	Version  string
	Outdated bool
	Latest   string
}

type Packages struct {
	Backend       string
	Available     bool
	Version       string
	Installed     int
	Outdated      int
	Apps          []App
	SearchBusy    bool
	SearchQuery   string
	SearchResults []App
	UpdatedAt     time.Time
}

type Store struct {
	mu         sync.RWMutex
	version    uint64
	System     System
	Net        []NetIface
	Internet   Internet
	Docker     Docker
	Printers   []Printer
	Services   []Service
	Packages   Packages
	CPUHist    []float64
	MemHist    []float64
	NetInHist  []float64
	NetOutHist []float64
}

func New() *Store {
	return &Store{}
}

func (s *Store) Update(fn func(*Store)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn(s)
	s.version++
}

func (s *Store) Read(fn func(*Store)) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fn(s)
}

func (s *Store) Version() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.version
}

func (s *Store) PushHist(cpu, mem float64, netIn, netOut float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	const max = 180
	s.CPUHist = append(s.CPUHist, cpu)
	s.MemHist = append(s.MemHist, mem)
	s.NetInHist = append(s.NetInHist, netIn)
	s.NetOutHist = append(s.NetOutHist, netOut)
	if len(s.CPUHist) > max {
		s.CPUHist = s.CPUHist[len(s.CPUHist)-max:]
	}
	if len(s.MemHist) > max {
		s.MemHist = s.MemHist[len(s.MemHist)-max:]
	}
	if len(s.NetInHist) > max {
		s.NetInHist = s.NetInHist[len(s.NetInHist)-max:]
	}
	if len(s.NetOutHist) > max {
		s.NetOutHist = s.NetOutHist[len(s.NetOutHist)-max:]
	}
	s.version++
}
