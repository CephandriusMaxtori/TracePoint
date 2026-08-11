package apps

import (
	"context"
	"fmt"
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
	t := time.NewTicker(60 * time.Second)
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

// Refresh triggers an immediate package refresh.
func (c *Collector) Refresh(ctx context.Context) { c.refresh(ctx) }

func (c *Collector) refresh(ctx context.Context) {
	name, ver, ok := Detect(ctx)
	pk := state.Packages{
		Backend:   name,
		Version:   ver,
		Available: ok,
		UpdatedAt: time.Now(),
	}
	if ok {
		if installed, err := ListInstalled(ctx, name); err == nil {
			pk.Apps = installed
			pk.Installed = len(installed)
		}
		if outdated, err := ListOutdated(ctx, name); err == nil {
			pk.Outdated = len(outdated)
			byName := map[string]state.App{}
			for _, o := range outdated {
				byName[o.Name] = o
			}
			for i := range pk.Apps {
				if o, ok := byName[pk.Apps[i].Name]; ok {
					pk.Apps[i].Outdated = true
					pk.Apps[i].Latest = o.Latest
				}
			}
		}
	}
	c.st.Update(func(s *state.Store) {
		s.Packages = pk
	})
}

type pkgBackend struct {
	name string
	cmd  string
}

var backends = []pkgBackend{
	{name: "chocolatey", cmd: "choco"},
	{name: "winget", cmd: "winget"},
	{name: "brew", cmd: "brew"},
	{name: "apt", cmd: "apt"},
	{name: "dnf", cmd: "dnf"},
	{name: "pacman", cmd: "pacman"},
}

// Detect finds the first available package manager (choco preferred).
func Detect(ctx context.Context) (name, version string, available bool) {
	for _, b := range backends {
		if osutil.LookPath(b.cmd) == "" {
			continue
		}
		var ver string
		switch b.name {
		case "chocolatey":
			ver, _ = osutil.RunCaptureStd(ctx, b.cmd, []string{"--version"})
		case "winget":
			ver, _ = osutil.RunCaptureStd(ctx, b.cmd, []string{"--version"})
		case "brew":
			ver, _ = osutil.RunCaptureStd(ctx, b.cmd, []string{"--version"})
		default:
			ver = "available"
		}
		return b.name, strings.TrimSpace(ver), true
	}
	return "", "", false
}

func ListInstalled(ctx context.Context, backend string) ([]state.App, error) {
	switch backend {
	case "chocolatey":
		return chocoList(ctx)
	case "winget":
		return wingetList(ctx)
	case "brew":
		return brewList(ctx)
	case "apt":
		return aptList(ctx)
	case "dnf":
		return dnfList(ctx)
	case "pacman":
		return pacmanList(ctx)
	}
	return nil, fmt.Errorf("unsupported backend %q", backend)
}

func ListOutdated(ctx context.Context, backend string) ([]state.App, error) {
	switch backend {
	case "chocolatey":
		out, err := osutil.RunCaptureStd(ctx, "choco", []string{"list", "--outdated", "--limit-output"})
		if err != nil {
			return nil, err
		}
		var apps []state.App
		for _, line := range strings.Split(out, "\n") {
			f := strings.Split(line, "|")
			if len(f) < 3 {
				continue
			}
			apps = append(apps, state.App{Name: f[0], Version: f[1], Outdated: true, Latest: f[2]})
		}
		return apps, nil
	case "brew":
		out, err := osutil.RunCaptureStd(ctx, "brew", []string{"outdated", "--quiet"})
		if err != nil {
			return nil, err
		}
		var apps []state.App
		for _, line := range strings.Split(out, "\n") {
			if line != "" {
				apps = append(apps, state.App{Name: line, Outdated: true})
			}
		}
		return apps, nil
	case "apt", "dnf":
		out, err := osutil.RunCaptureStd(ctx, "apt", []string{"list", "--upgradable"})
		if err != nil {
			return nil, err
		}
		var apps []state.App
		for _, line := range strings.Split(out, "\n") {
			if strings.HasPrefix(line, "Listing") {
				continue
			}
			name := strings.SplitN(line, "/", 2)[0]
			if name != "" {
				apps = append(apps, state.App{Name: name, Outdated: true})
			}
		}
		return apps, nil
	}
	return nil, nil
}

func Search(ctx context.Context, backend, query string, log func(string)) ([]state.App, error) {
	switch backend {
	case "chocolatey":
		out, err := osutil.RunCaptureStd(ctx, "choco", []string{"search", query, "--limit-output", "--no-progress"})
		if err != nil {
			return nil, err
		}
		var apps []state.App
		for _, line := range strings.Split(out, "\n") {
			f := strings.Split(line, "|")
			if len(f) >= 2 {
				apps = append(apps, state.App{Name: f[0], Version: f[1]})
			} else if line != "" {
				apps = append(apps, state.App{Name: line})
			}
		}
		return apps, nil
	case "winget":
		out, err := osutil.RunCaptureStd(ctx, "winget", []string{"search", query, "--accept-source-agreements", "--disable-interactivity"})
		if err != nil {
			return nil, err
		}
		return rawLines(out), nil
	default:
		return nil, fmt.Errorf("search not supported for %s", backend)
	}
}

func Install(ctx context.Context, backend, pkg string, log func(string)) error {
	switch backend {
	case "chocolatey":
		return osutil.RunStream(ctx, "choco", []string{"install", pkg, "-y", "--no-progress"}, log)
	case "winget":
		return osutil.RunStream(ctx, "winget", []string{"install", "-e", "--id", pkg, "--silent", "--accept-package-agreements", "--accept-source-agreements", "--disable-interactivity"}, log)
	case "brew":
		return osutil.RunStream(ctx, "brew", []string{"install", pkg}, log)
	case "apt":
		return osutil.RunStream(ctx, "sudo", []string{"apt", "install", "-y", pkg}, log)
	case "dnf":
		return osutil.RunStream(ctx, "sudo", []string{"dnf", "install", "-y", pkg}, log)
	case "pacman":
		return osutil.RunStream(ctx, "sudo", []string{"pacman", "-S", "--noconfirm", pkg}, log)
	}
	return fmt.Errorf("unsupported backend %q", backend)
}

func Uninstall(ctx context.Context, backend, pkg string, log func(string)) error {
	switch backend {
	case "chocolatey":
		return osutil.RunStream(ctx, "choco", []string{"uninstall", pkg, "-y", "--no-progress"}, log)
	case "winget":
		return osutil.RunStream(ctx, "winget", []string{"uninstall", "--id", pkg, "--silent", "--disable-interactivity"}, log)
	case "brew":
		return osutil.RunStream(ctx, "brew", []string{"uninstall", pkg}, log)
	case "apt":
		return osutil.RunStream(ctx, "sudo", []string{"apt", "remove", "-y", pkg}, log)
	case "dnf":
		return osutil.RunStream(ctx, "sudo", []string{"dnf", "remove", "-y", pkg}, log)
	case "pacman":
		return osutil.RunStream(ctx, "sudo", []string{"pacman", "-R", "--noconfirm", pkg}, log)
	}
	return fmt.Errorf("unsupported backend %q", backend)
}

func Upgrade(ctx context.Context, backend, pkg string, log func(string)) error {
	switch backend {
	case "chocolatey":
		return osutil.RunStream(ctx, "choco", []string{"upgrade", pkg, "-y", "--no-progress"}, log)
	case "brew":
		return osutil.RunStream(ctx, "brew", []string{"upgrade", pkg}, log)
	case "apt":
		return osutil.RunStream(ctx, "sudo", []string{"apt", "upgrade", "-y", pkg}, log)
	case "dnf":
		return osutil.RunStream(ctx, "sudo", []string{"dnf", "upgrade", "-y", pkg}, log)
	case "pacman":
		return osutil.RunStream(ctx, "sudo", []string{"pacman", "-Syu", "--noconfirm", pkg}, log)
	}
	return fmt.Errorf("upgrade not supported for %s", backend)
}

func UpgradeAll(ctx context.Context, backend string, log func(string)) error {
	switch backend {
	case "chocolatey":
		return osutil.RunStream(ctx, "choco", []string{"upgrade", "all", "-y", "--no-progress"}, log)
	case "brew":
		return osutil.RunStream(ctx, "brew", []string{"upgrade"}, log)
	case "apt":
		return osutil.RunStream(ctx, "sudo", []string{"apt", "upgrade", "-y"}, log)
	case "dnf":
		return osutil.RunStream(ctx, "sudo", []string{"dnf", "upgrade", "-y"}, log)
	case "pacman":
		return osutil.RunStream(ctx, "sudo", []string{"pacman", "-Syu", "--noconfirm"}, log)
	}
	return fmt.Errorf("upgrade all not supported for %s", backend)
}

func chocoList(ctx context.Context) ([]state.App, error) {
	out, err := osutil.RunCaptureStd(ctx, "choco", []string{"list", "--limit-output", "--no-progress"})
	if err != nil {
		return nil, err
	}
	var apps []state.App
	for _, line := range strings.Split(out, "\n") {
		f := strings.Split(line, "|")
		if len(f) >= 2 {
			apps = append(apps, state.App{Name: f[0], Version: f[1]})
		}
	}
	return apps, nil
}

func wingetList(ctx context.Context) ([]state.App, error) {
	out, err := osutil.RunCaptureStd(ctx, "winget", []string{"list", "--accept-source-agreements", "--disable-interactivity"})
	if err != nil {
		return nil, err
	}
	return rawLines(out), nil
}

func brewList(ctx context.Context) ([]state.App, error) {
	out, err := osutil.RunCaptureStd(ctx, "brew", []string{"list", "--versions"})
	if err != nil {
		return nil, err
	}
	var apps []state.App
	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(line)
		if len(f) >= 1 {
			apps = append(apps, state.App{Name: f[0], Version: strings.Join(f[1:], " ")})
		}
	}
	return apps, nil
}

func aptList(ctx context.Context) ([]state.App, error) {
	out, err := osutil.RunCaptureStd(ctx, "apt", []string{"list", "--installed"})
	if err != nil {
		return nil, err
	}
	var apps []state.App
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "Listing") {
			continue
		}
		name := strings.SplitN(line, "/", 2)[0]
		if name != "" {
			apps = append(apps, state.App{Name: name})
		}
	}
	return apps, nil
}

func dnfList(ctx context.Context) ([]state.App, error) {
	out, err := osutil.RunCaptureStd(ctx, "dnf", []string{"list", "--installed"})
	if err != nil {
		return nil, err
	}
	var apps []state.App
	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(line)
		if len(f) >= 2 {
			apps = append(apps, state.App{Name: f[0], Version: f[1]})
		}
	}
	return apps, nil
}

func pacmanList(ctx context.Context) ([]state.App, error) {
	out, err := osutil.RunCaptureStd(ctx, "pacman", []string{"-Q"})
	if err != nil {
		return nil, err
	}
	var apps []state.App
	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(line)
		if len(f) >= 1 {
			apps = append(apps, state.App{Name: f[0], Version: strings.Join(f[1:], " ")})
		}
	}
	return apps, nil
}

func rawLines(out string) []state.App {
	var apps []state.App
	for _, line := range strings.Split(out, "\n") {
		if line != "" {
			apps = append(apps, state.App{Name: line})
		}
	}
	return apps
}

func PlatformHint() string {
	switch runtime.GOOS {
	case "windows":
		return "Chocolatey (choco) is recommended. Install it at https://chocolatey.org/install"
	case "darwin":
		return "Homebrew (brew) is used as the package manager."
	default:
		return "apt/dnf/pacman are used as the package manager."
	}
}
