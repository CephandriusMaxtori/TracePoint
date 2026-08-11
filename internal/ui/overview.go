package ui

import (
	"fmt"
	"image"
	"image/color"
	"time"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"tracepoint/internal/state"
)

type overviewState struct {
	procsList widget.List
	disksList widget.List
}

func fmtBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func fmtPct(v float64) string { return fmt.Sprintf("%.0f%%", v) }

func fmtRate(bps float64) string {
	if bps >= 1<<20 {
		return fmt.Sprintf("%.1f MB/s", bps/(1<<20))
	}
	if bps >= 1<<10 {
		return fmt.Sprintf("%.0f KB/s", bps/(1<<10))
	}
	return fmt.Sprintf("%.0f B/s", bps)
}

func fmtUptime(sec uint64) string {
	d := sec / 86400
	h := (sec % 86400) / 3600
	m := (sec % 3600) / 60
	return fmt.Sprintf("%dd %dh %dm", d, h, m)
}

func (ui *UI) overviewPage(gtx layout.Context) layout.Dimensions {
	var sys state.System
	var pk state.Packages
	var disks []state.Disk
	var procs []state.Proc
	var cpuHist, memHist, inHist, outHist []float64
	var uptime uint64
	ui.st.Read(func(s *state.Store) {
		sys = s.System
		pk = s.Packages
		disks = s.System.Disks
		procs = s.System.Procs
		cpuHist = s.CPUHist
		memHist = s.MemHist
		inHist = s.NetInHist
		outHist = s.NetOutHist
		uptime = s.System.UptimeSec
	})

	pad := unit.Dp(14)
	body := func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.gaugeRow(gtx, sys)
			}),
			layout.Rigid(layout.Spacer{Height: pad}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.sparkRow(gtx, cpuHist, memHist, inHist, outHist)
			}),
			layout.Rigid(layout.Spacer{Height: pad}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.infoRow(gtx, sys, pk, uptime)
			}),
			layout.Rigid(layout.Spacer{Height: pad}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.procsCard(gtx, procs)
			}),
			layout.Rigid(layout.Spacer{Height: pad}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.disksCard(gtx, disks)
			}),
		)
	}
	return ui.pageInset(gtx, body)
}

func (ui *UI) pageInset(gtx layout.Context, w layout.Widget) layout.Dimensions {
	return layout.Inset{Top: 20, Bottom: 24, Left: 22, Right: 22}.Layout(gtx, w)
}

func (ui *UI) gaugeRow(gtx layout.Context, sys state.System) layout.Dimensions {
	disk := rootDisk(sys.Disks)
	cards := []func(gtx layout.Context) layout.Dimensions{
		func(gtx layout.Context) layout.Dimensions {
			return ui.gaugeCard(gtx, "CPU", fmtPct(sys.CPUPercent), float32(sys.CPUPercent/100), ui.th.valueColor(sys.CPUPercent))
		},
		func(gtx layout.Context) layout.Dimensions {
			return ui.gaugeCard(gtx, "Memory", fmtPct(sys.MemPercent), float32(sys.MemPercent/100), ui.th.valueColor(sys.MemPercent))
		},
		func(gtx layout.Context) layout.Dimensions {
			v := disk.Percent
			return ui.gaugeCard(gtx, "Disk ("+disk.Mount+")", fmtPct(v), float32(v/100), ui.th.valueColor(v))
		},
		func(gtx layout.Context) layout.Dimensions {
			return ui.statCard(gtx, "Uptime", fmtUptime(sys.UptimeSec), fmt.Sprintf("%.2f avg load", sys.Load1))
		},
	}
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, func() []layout.FlexChild {
		children := make([]layout.FlexChild, 0, len(cards))
		for _, c := range cards {
			children = append(children, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Right: 14}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return c(gtx)
				})
			}))
		}
		return children
	}()...)
}

func rootDisk(disks []state.Disk) state.Disk {
	var best state.Disk
	for _, d := range disks {
		if d.Total > best.Total {
			best = d
		}
	}
	return best
}

func (ui *UI) gaugeCard(gtx layout.Context, title, value string, frac float32, c color.NRGBA) layout.Dimensions {
	return ui.card(gtx, ui.th.Pal.Card, radiusCard, 16, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.caption(gtx, title)
			}),
			layout.Rigid(layout.Spacer{Height: 10}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.ring(gtx, frac, c, 74)
					}),
					layout.Rigid(layout.Spacer{Width: 14}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						l := material.Label(ui.th.Theme, unit.Sp(22), value)
						l.Font.Weight = 700
						l.Color = c
						return l.Layout(gtx)
					}),
				)
			}),
		)
	})
}

func (ui *UI) statCard(gtx layout.Context, title, value, sub string) layout.Dimensions {
	return ui.card(gtx, ui.th.Pal.Card, radiusCard, 16, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.caption(gtx, title)
			}),
			layout.Rigid(layout.Spacer{Height: 12}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				l := material.Label(ui.th.Theme, unit.Sp(22), value)
				l.Font.Weight = 700
				return l.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: 6}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.muted(gtx, sub)
			}),
		)
	})
}

func (ui *UI) sparkRow(gtx layout.Context, cpu, mem, in, out []float64) layout.Dimensions {
	cards := []func(gtx layout.Context) layout.Dimensions{
		func(gtx layout.Context) layout.Dimensions {
			return ui.sparkCard(gtx, "CPU History", cpu, ui.th.Pal.Accent)
		},
		func(gtx layout.Context) layout.Dimensions {
			return ui.sparkCard(gtx, "Memory History", mem, ui.th.Pal.Success)
		},
		func(gtx layout.Context) layout.Dimensions {
			return ui.sparkCard2(gtx, "Network Throughput", in, out)
		},
	}
	children := make([]layout.FlexChild, 0, len(cards))
	for _, c := range cards {
		children = append(children, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Right: 14}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return c(gtx)
			})
		}))
	}
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, children...)
}

func (ui *UI) sparkCard(gtx layout.Context, title string, data []float64, c color.NRGBA) layout.Dimensions {
	return ui.card(gtx, ui.th.Pal.Card, radiusCard, 14, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.caption(gtx, title)
			}),
			layout.Rigid(layout.Spacer{Height: 10}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				w := gtx.Constraints.Max.X
				h := gtx.Dp(unit.Dp(56))
				sparkline(gtx.Ops, data, c, w, h)
				return layout.Dimensions{Size: image.Pt(w, h)}
			}),
		)
	})
}

func (ui *UI) sparkCard2(gtx layout.Context, title string, in, out []float64) layout.Dimensions {
	return ui.card(gtx, ui.th.Pal.Card, radiusCard, 14, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.caption(gtx, title)
			}),
			layout.Rigid(layout.Spacer{Height: 10}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				w := gtx.Constraints.Max.X
				h := gtx.Dp(unit.Dp(56))
				sparkline(gtx.Ops, in, ui.th.Pal.Success, w, h)
				sparkline(gtx.Ops, out, ui.th.Pal.Accent, w, h)
				return layout.Dimensions{Size: image.Pt(w, h)}
			}),
			layout.Rigid(layout.Spacer{Height: 6}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.statusDot(gtx, ui.th.Pal.Success, 7)
					}),
					layout.Rigid(layout.Spacer{Width: 6}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.caption(gtx, "in")
					}),
					layout.Rigid(layout.Spacer{Width: 12}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.statusDot(gtx, ui.th.Pal.Accent, 7)
					}),
					layout.Rigid(layout.Spacer{Width: 6}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.caption(gtx, "out")
					}),
				)
			}),
		)
	})
}

func (ui *UI) infoRow(gtx layout.Context, sys state.System, pk state.Packages, uptime uint64) layout.Dimensions {
	host := ui.card(gtx, ui.th.Pal.Card, radiusCard, 16, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.sectionTitle(gtx, "System")
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.kv(gtx, "Hostname", sys.Hostname)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.kv(gtx, "OS", fmt.Sprintf("%s %s (%s)", sys.OS, sys.Platform, sys.Arch))
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.kv(gtx, "Kernel", sys.Kernel)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.kv(gtx, "CPU Cores", fmt.Sprintf("%d", sys.CPUCount))
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.kv(gtx, "Memory", fmt.Sprintf("%s / %s", fmtBytes(sys.MemUsed), fmtBytes(sys.MemTotal)))
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.kv(gtx, "Boot Time", time.Unix(int64(sys.BootTimeSec), 0).Format("2006-01-02 15:04"))
			}),
		)
	})
	pkCard := ui.card(gtx, ui.th.Pal.Card, radiusCard, 16, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.sectionTitle(gtx, "Package Manager")
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.kv(gtx, "Backend", pk.Backend)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.kv(gtx, "Version", pk.Version)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.kv(gtx, "Installed", fmt.Sprintf("%d", pk.Installed))
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.kv(gtx, "Outdated", fmt.Sprintf("%d", pk.Outdated))
			}),
			layout.Rigid(layout.Spacer{Height: 12}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if !pk.Available {
					return ui.muted(gtx, "No package manager detected. Chocolatey is recommended on Windows.")
				}
				return layout.Dimensions{}
			}),
		)
	})
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Right: 14}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return host(gtx)
			})
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return pkCard(gtx)
		}),
	)
}

func (ui *UI) kv(gtx layout.Context, k, v string) layout.Dimensions {
	return layout.Inset{Top: 4, Bottom: 4}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Max.X = gtx.Dp(unit.Dp(120))
				return ui.muted(gtx, k)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				l := material.Label(ui.th.Theme, unit.Sp(13), v)
				l.Color = ui.th.Pal.Fg
				return l.Layout(gtx)
			}),
		)
	})
}

func (ui *UI) procsCard(gtx layout.Context, procs []state.Proc) layout.Dimensions {
	return ui.card(gtx, ui.th.Pal.Card, radiusCard, 16, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.sectionTitle(gtx, "Top Processes by CPU")
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(300))
				if len(procs) == 0 {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return ui.muted(gtx, "No process data")
					})
				}
				return material.List(ui.th.Theme, &ui.overview.procsList).Layout(gtx, len(procs), func(gtx layout.Context, i int) layout.Dimensions {
					return ui.procRow(gtx, procs[i])
				})
			}),
		)
	})
}

func (ui *UI) procRow(gtx layout.Context, p state.Proc) layout.Dimensions {
	return layout.Inset{Top: 5, Bottom: 5}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				l := material.Label(ui.th.Theme, unit.Sp(12), fmt.Sprintf("%d", p.PID))
				l.Color = ui.th.Pal.Muted
				gtx.Constraints.Max.X = gtx.Dp(unit.Dp(60))
				return l.Layout(gtx)
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				l := material.Label(ui.th.Theme, unit.Sp(13), p.Name)
				l.Color = ui.th.Pal.Fg
				return l.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Max.X = gtx.Dp(unit.Dp(70))
				return hbar(gtx, float32(p.CPUPercent/100), ui.th.Pal.Accent, ui.th.Pal.CardAlt, 70, 6)
			}),
			layout.Rigid(layout.Spacer{Width: 10}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Max.X = gtx.Dp(unit.Dp(56))
				l := material.Label(ui.th.Theme, unit.Sp(12), fmt.Sprintf("%.1f%%", p.CPUPercent))
				l.Color = ui.th.Pal.Fg
				return l.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Width: 10}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Max.X = gtx.Dp(unit.Dp(56))
				l := material.Label(ui.th.Theme, unit.Sp(12), fmt.Sprintf("%.1f%%", p.MemPercent))
				l.Color = ui.th.Pal.Muted
				return l.Layout(gtx)
			}),
		)
	})
}

func (ui *UI) disksCard(gtx layout.Context, disks []state.Disk) layout.Dimensions {
	return ui.card(gtx, ui.th.Pal.Card, radiusCard, 16, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.sectionTitle(gtx, "Disk Usage")
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				if len(disks) == 0 {
					return layout.Dimensions{}
				}
				gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(220))
				return material.List(ui.th.Theme, &ui.overview.disksList).Layout(gtx, len(disks), func(gtx layout.Context, i int) layout.Dimensions {
					return ui.diskRow(gtx, disks[i])
				})
			}),
		)
	})
}

func (ui *UI) diskRow(gtx layout.Context, d state.Disk) layout.Dimensions {
	return layout.Inset{Top: 6, Bottom: 6}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						l := material.Label(ui.th.Theme, unit.Sp(13), d.Mount)
						l.Font.Weight = 600
						l.Color = ui.th.Pal.Fg
						return l.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Width: 10}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.caption(gtx, d.FSType)
					}),
					layout.Flexed(1, layout.Spacer{Width: 0}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.caption(gtx, fmt.Sprintf("%s used of %s", fmtBytes(d.Used), fmtBytes(d.Total)))
					}),
				)
			}),
			layout.Rigid(layout.Spacer{Height: 6}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				w := gtx.Constraints.Max.X
				return hbar(gtx, float32(d.Percent/100), ui.th.valueColor(d.Percent), ui.th.Pal.CardAlt, w, 8)
			}),
		)
	})
}
