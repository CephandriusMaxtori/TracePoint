package ui

import (
	"context"
	"fmt"
	"sync"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"tracepoint/internal/collectors/network"
	"tracepoint/internal/state"
)

type networkState struct {
	mu sync.Mutex

	ifacesList widget.List

	pingEditor widget.Editor
	pingBtn    widget.Clickable
	pingRes    *network.PingResult
	pingErr    string
	pingBusy   bool

	scanHostEditor widget.Editor
	scanSpecEditor widget.Editor
	scanBtn        widget.Clickable
	scanRes        []network.PortResult
	scanErr        string
	scanBusy       bool
	scanList       widget.List

	lookupEditor widget.Editor
	lookupBtn    widget.Clickable
	lookupRes    []string
	lookupErr    string
}

func (ui *UI) networkPage(gtx layout.Context) layout.Dimensions {
	ns := &ui.network
	if ns.pingEditor.Text() == "" && ns.pingEditor.Len() == 0 {
		ns.pingEditor.SetText("8.8.8.8")
	}
	if ns.scanHostEditor.Text() == "" && ns.scanHostEditor.Len() == 0 {
		ns.scanHostEditor.SetText("127.0.0.1")
	}
	if ns.scanSpecEditor.Text() == "" && ns.scanSpecEditor.Len() == 0 {
		ns.scanSpecEditor.SetText("22,80,443,8080")
	}

	ui.handlePing(gtx)
	ui.handleScan(gtx)
	ui.handleLookup(gtx)

	return ui.pageInset(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.ifacesCard(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: 14}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.pingCard(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: 14}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.scanCard(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: 14}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.lookupCard(gtx)
			}),
		)
	})
}

func (ui *UI) handlePing(gtx layout.Context) {
	ns := &ui.network
	if ns.pingBtn.Clicked(gtx) {
		host := ns.pingEditor.Text()
		if host == "" {
			host = "8.8.8.8"
		}
		ns.mu.Lock()
		ns.pingBusy = true
		ns.pingRes = nil
		ns.pingErr = ""
		ns.mu.Unlock()
		ui.acts.Run("Ping "+host, func(ctx context.Context, log func(format string, args ...any)) {
			res, err := network.Ping(ctx, host, 4)
			ns.mu.Lock()
			defer ns.mu.Unlock()
			ns.pingBusy = false
			if err != nil {
				ns.pingErr = err.Error()
				return
			}
			ns.pingRes = res
		})
	}
}

func (ui *UI) handleScan(gtx layout.Context) {
	ns := &ui.network
	if ns.scanBtn.Clicked(gtx) {
		host := ns.scanHostEditor.Text()
		spec := ns.scanSpecEditor.Text()
		ports, err := network.ParsePortSpec(spec)
		if err != nil {
			ns.mu.Lock()
			ns.scanErr = err.Error()
			ns.scanBusy = false
			ns.mu.Unlock()
			return
		}
		ns.mu.Lock()
		ns.scanBusy = true
		ns.scanRes = nil
		ns.scanErr = ""
		ns.mu.Unlock()
		ui.acts.Run(fmt.Sprintf("Port scan %s (%d ports)", host, len(ports)), func(ctx context.Context, log func(format string, args ...any)) {
			res, err := network.PortScanConcurrent(ctx, host, ports, log)
			ns.mu.Lock()
			defer ns.mu.Unlock()
			ns.scanBusy = false
			if err != nil {
				ns.scanErr = err.Error()
				return
			}
			ns.scanRes = res
		})
	}
}

func (ui *UI) handleLookup(gtx layout.Context) {
	ns := &ui.network
	if ns.lookupBtn.Clicked(gtx) {
		host := ns.lookupEditor.Text()
		if host == "" {
			return
		}
		ns.mu.Lock()
		ns.lookupRes = nil
		ns.lookupErr = ""
		ns.mu.Unlock()
		ui.acts.Run("Lookup "+host, func(ctx context.Context, log func(format string, args ...any)) {
			ips, err := network.LookupHost(ctx, host)
			ns.mu.Lock()
			defer ns.mu.Unlock()
			if err != nil {
				ns.lookupErr = err.Error()
				return
			}
			ns.lookupRes = ips
		})
	}
}

func (ui *UI) ifacesCard(gtx layout.Context) layout.Dimensions {
	var ifaces []state.NetIface
	ui.st.Read(func(s *state.Store) { ifaces = s.Net })
	return ui.card(gtx, ui.th.Pal.Card, radiusCard, 16, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.sectionTitle(gtx, "Network Interfaces")
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				if len(ifaces) == 0 {
					return layout.Dimensions{}
				}
				gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(240))
				return material.List(ui.th.Theme, &ui.network.ifacesList).Layout(gtx, len(ifaces), func(gtx layout.Context, i int) layout.Dimensions {
					return ui.ifaceRow(gtx, ifaces[i])
				})
			}),
		)
	})
}

func (ui *UI) ifaceRow(gtx layout.Context, iface state.NetIface) layout.Dimensions {
	return layout.Inset{Top: 6, Bottom: 6}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						c := ui.th.Pal.Muted
						if iface.Up {
							c = ui.th.Pal.Success
						}
						return ui.statusDot(gtx, c, 8)
					}),
					layout.Rigid(layout.Spacer{Width: 8}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						l := material.Label(ui.th.Theme, unit.Sp(13), iface.Name)
						l.Font.Weight = 600
						l.Color = ui.th.Pal.Fg
						return l.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Width: 10}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.caption(gtx, fmt.Sprintf("MTU %d", iface.MTU))
					}),
					layout.Flexed(1, layout.Spacer{Width: 0}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.caption(gtx, fmt.Sprintf("in %s   out %s", fmtRate(iface.RxBps), fmtRate(iface.TxBps)))
					}),
				)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if len(iface.Addrs) == 0 {
					return layout.Dimensions{}
				}
				return layout.Inset{Left: 16, Top: 2}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					l := material.Label(ui.th.Theme, unit.Sp(11), fmt.Sprintf("%s", iface.Addrs))
					l.Color = ui.th.Pal.Muted
					return l.Layout(gtx)
				})
			}),
		)
	})
}

func (ui *UI) pingCard(gtx layout.Context) layout.Dimensions {
	ns := &ui.network
	return ui.card(gtx, ui.th.Pal.Card, radiusCard, 16, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.sectionTitle(gtx, "Ping")
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						ed := material.Editor(ui.th.Theme, &ns.pingEditor, "host or IP")
						ed.Color = ui.th.Pal.Fg
						ed.HintColor = ui.th.Pal.Muted
						return ed.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Width: 12}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if ns.pingBusy {
							return ui.ghostButton(gtx, &ns.pingBtn, "Pinging…")
						}
						return ui.primaryButton(gtx, &ns.pingBtn, "Ping")
					}),
				)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.pingResult(gtx)
			}),
		)
	})
}

func (ui *UI) pingResult(gtx layout.Context) layout.Dimensions {
	ns := &ui.network
	ns.mu.Lock()
	defer ns.mu.Unlock()
	if ns.pingErr != "" {
		return layout.Inset{Top: 12}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			l := material.Label(ui.th.Theme, unit.Sp(12), ns.pingErr)
			l.Color = ui.th.Pal.Danger
			return l.Layout(gtx)
		})
	}
	res := ns.pingRes
	if res == nil {
		return layout.Dimensions{}
	}
	color := ui.th.Pal.Success
	if res.LossPct >= 50 {
		color = ui.th.Pal.Danger
	} else if res.LossPct > 0 {
		color = ui.th.Pal.Warn
	}
	return layout.Inset{Top: 12}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				l := material.Label(ui.th.Theme, unit.Sp(13), res.Host)
				l.Font.Weight = 600
				l.Color = color
				return l.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.caption(gtx, fmt.Sprintf("%d/%d packets · %.0f%% loss · min/avg/max %.1f/%.1f/%.1f ms",
					res.Recv, res.Sent, res.LossPct, res.MinMs, res.AvgMs, res.MaxMs))
			}),
		)
	})
}

func (ui *UI) scanCard(gtx layout.Context) layout.Dimensions {
	ns := &ui.network
	return ui.card(gtx, ui.th.Pal.Card, radiusCard, 16, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.sectionTitle(gtx, "Port Scan")
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Max.X = gtx.Dp(unit.Dp(160))
						ed := material.Editor(ui.th.Theme, &ns.scanHostEditor, "host or IP")
						ed.Color = ui.th.Pal.Fg
						ed.HintColor = ui.th.Pal.Muted
						return ed.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Width: 10}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						ed := material.Editor(ui.th.Theme, &ns.scanSpecEditor, "ports, e.g. 22,80,443 or 1-1024")
						ed.Color = ui.th.Pal.Fg
						ed.HintColor = ui.th.Pal.Muted
						return ed.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Width: 12}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if ns.scanBusy {
							return ui.ghostButton(gtx, &ns.scanBtn, "Scanning…")
						}
						return ui.primaryButton(gtx, &ns.scanBtn, "Scan")
					}),
				)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.scanResults(gtx)
			}),
		)
	})
}

func (ui *UI) scanResults(gtx layout.Context) layout.Dimensions {
	ns := &ui.network
	ns.mu.Lock()
	defer ns.mu.Unlock()
	if ns.scanErr != "" {
		return layout.Inset{Top: 12}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			l := material.Label(ui.th.Theme, unit.Sp(12), ns.scanErr)
			l.Color = ui.th.Pal.Danger
			return l.Layout(gtx)
		})
	}
	if len(ns.scanRes) == 0 {
		return layout.Dimensions{}
	}
	return layout.Inset{Top: 12}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.caption(gtx, fmt.Sprintf("%d open port(s)", len(ns.scanRes)))
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(150))
				return material.List(ui.th.Theme, &ns.scanList).Layout(gtx, len(ns.scanRes), func(gtx layout.Context, i int) layout.Dimensions {
					r := ns.scanRes[i]
					l := material.Label(ui.th.Theme, unit.Sp(12), fmt.Sprintf("%d/tcp  %s", r.Port, r.Service))
					l.Color = ui.th.Pal.Success
					return layout.Inset{Top: 2, Bottom: 2}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return l.Layout(gtx)
					})
				})
			}),
		)
	})
}

func (ui *UI) lookupCard(gtx layout.Context) layout.Dimensions {
	ns := &ui.network
	return ui.card(gtx, ui.th.Pal.Card, radiusCard, 16, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.sectionTitle(gtx, "DNS Lookup")
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						ed := material.Editor(ui.th.Theme, &ns.lookupEditor, "hostname, e.g. example.com")
						ed.Color = ui.th.Pal.Fg
						ed.HintColor = ui.th.Pal.Muted
						return ed.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Width: 12}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.primaryButton(gtx, &ns.lookupBtn, "Lookup")
					}),
				)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				ns.mu.Lock()
				defer ns.mu.Unlock()
				if ns.lookupErr != "" {
					return layout.Inset{Top: 12}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						l := material.Label(ui.th.Theme, unit.Sp(12), ns.lookupErr)
						l.Color = ui.th.Pal.Danger
						return l.Layout(gtx)
					})
				}
				if len(ns.lookupRes) == 0 {
					return layout.Dimensions{}
				}
				return layout.Inset{Top: 12}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx, func() []layout.FlexChild {
						children := make([]layout.FlexChild, 0, len(ns.lookupRes))
						for _, ip := range ns.lookupRes {
							ip := ip
							children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{Top: 2, Bottom: 2}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return material.Label(ui.th.Theme, unit.Sp(12), ip).Layout(gtx)
								})
							}))
						}
						return children
					}()...)
				})
			}),
		)
	})
}
