package ui

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"time"

	"gio.tools/icons"
	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"tracepoint/internal/actions"
	"tracepoint/internal/state"
)

type Page int

const (
	PageOverview Page = iota
	PageNetwork
	PageInternet
	PageServices
	PageLogs
	PageDocker
	PagePrinters
	PageApps
)

type navItem struct {
	icon  *widget.Icon
	label string
	page  Page
	click widget.Clickable
}

type dialog struct {
	open         bool
	title        string
	message      string
	confirmLabel string
	danger       bool
	onConfirm    func()
	confirm      widget.Clickable
	cancel       widget.Clickable
	scrim        widget.Clickable
}

type UI struct {
	th   *Theme
	st   *state.Store
	acts *actions.Manager
	ctx  context.Context
	win  *app.Window

	current Page
	nav     []navItem

	dialog  dialog
	opsOpen bool

	opHeader map[string]*widget.Clickable
	opLogs   map[string]*widget.List
	opOpen   map[string]bool

	pageList widget.List
	opsList  widget.List

	overview overviewState
	network  networkState
	internet internetState
	services servicesState
	logs     logsState
	docker   dockerState
	printers printersState
	apps     appsState
}

func New(th *Theme, st *state.Store, acts *actions.Manager, ctx context.Context) *UI {
	ui := &UI{
		th:       th,
		st:       st,
		acts:     acts,
		ctx:      ctx,
		opHeader: map[string]*widget.Clickable{},
		opLogs:   map[string]*widget.List{},
		opOpen:   map[string]bool{},
	}
	ui.nav = []navItem{
		{icon: icons.ActionDashboard, label: "Overview", page: PageOverview},
		{icon: icons.DeviceNetworkWifi, label: "Network", page: PageNetwork},
		{icon: icons.AVWeb, label: "Internet", page: PageInternet},
		{icon: icons.ActionSettings, label: "Services", page: PageServices},
		{icon: icons.ActionList, label: "Logs", page: PageLogs},
		{icon: icons.HardwareDevicesOther, label: "Docker", page: PageDocker},
		{icon: icons.ActionPrint, label: "Printers", page: PagePrinters},
		{icon: icons.ActionStore, label: "Apps", page: PageApps},
	}
	return ui
}

func (ui *UI) SetWindow(w *app.Window) { ui.win = w }

func (ui *UI) Layout(gtx layout.Context) layout.Dimensions {
	for i := range ui.nav {
		if ui.nav[i].click.Clicked(gtx) {
			ui.current = ui.nav[i].page
			ui.win.Invalidate()
		}
	}
	if ui.opsToggle.Clicked(gtx) {
		ui.opsOpen = !ui.opsOpen
		ui.win.Invalidate()
	}
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints = layout.Exact(image.Pt(gtx.Dp(unit.Dp(228)), gtx.Constraints.Max.Y))
					return ui.sidebar(gtx)
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return ui.content(gtx)
				}),
			)
		}),
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			if ui.dialog.open {
				return ui.renderDialog(gtx)
			}
			return layout.Dimensions{}
		}),
	)
}

func (ui *UI) sidebar(gtx layout.Context) layout.Dimensions {
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			paint.FillShape(gtx.Ops, ui.th.Pal.Sidebar, clip.Rect{Max: gtx.Constraints.Max}.Op())
			return layout.Dimensions{Size: gtx.Constraints.Max}
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: 22, Bottom: 16, Left: 16, Right: 16}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(ui.sidebarHeader),
					layout.Rigid(layout.Spacer{Height: 26}),
					layout.Rigid(ui.sidebarNav),
					layout.Flexed(1, layout.Spacer{Height: 0}.Layout),
					layout.Rigid(ui.sidebarFooter),
				)
			})
		}),
	)
}

func (ui *UI) sidebarHeader(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Background{}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					rr := clip.UniformRRect(image.Rect(0, 0, 40, 40), 11)
					paint.FillShape(gtx.Ops, ui.th.Pal.Accent, rr.Op(gtx.Ops))
					return layout.Dimensions{Size: image.Pt(40, 40)}
				},
				func(gtx layout.Context) layout.Dimensions {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return icons.ActionVerifiedUser.Layout(gtx, ui.th.Pal.TextOnAccent)
					})
				},
			)
		}),
		layout.Rigid(layout.Spacer{Width: 12}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					l := material.Label(ui.th, unit.Sp(17), "TracePoint")
					l.Font.Weight = 700
					l.Color = ui.th.Pal.Fg
					return l.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					l := material.Label(ui.th, unit.Sp(11), "Sysadmin Dashboard")
					l.Color = ui.th.Pal.Muted
					return l.Layout(gtx)
				}),
			)
		}),
	)
}

func (ui *UI) sidebarNav(gtx layout.Context) layout.Dimensions {
	children := make([]layout.FlexChild, 0, len(ui.nav))
	for i := range ui.nav {
		item := &ui.nav[i]
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return ui.navRow(gtx, item)
		}))
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

func (ui *UI) navRow(gtx layout.Context, item *navItem) layout.Dimensions {
	selected := ui.current == item.page
	bg := color.NRGBA{}
	fg := ui.th.Pal.Muted
	if selected {
		bg = ui.th.Pal.Accent
		fg = ui.th.Pal.TextOnAccent
	}
	btn := material.Button(ui.th, &item.click, "")
	btn.Background = bg
	btn.CornerRadius = unit.Dp(radiusControl)
	btn.Inset = layout.Inset{Top: 10, Bottom: 10, Left: 12, Right: 12}
	return layout.Inset{Top: 2, Bottom: 2}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return btn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return item.icon.Layout(gtx, fg)
				}),
				layout.Rigid(layout.Spacer{Width: 12}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					l := material.Label(ui.th, unit.Sp(14), item.label)
					l.Color = ui.th.Pal.Fg
					if selected {
						l.Color = ui.th.Pal.TextOnAccent
						l.Font.Weight = 600
					}
					return l.Layout(gtx)
				}),
			)
		})
	})
}

func (ui *UI) sidebarFooter(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			rr := clip.UniformRRect(image.Rect(0, 0, gtx.Constraints.Max.X, 1), 0)
			paint.FillShape(gtx.Ops, ui.th.Pal.Border, rr.Op(gtx.Ops))
			return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, 1)}
		}),
		layout.Rigid(layout.Spacer{Height: 12}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return ui.statusDot(gtx, ui.th.Pal.Success, 8)
				}),
				layout.Rigid(layout.Spacer{Width: 8}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					l := material.Label(ui.th, unit.Sp(12), "Live")
					l.Color = ui.th.Pal.Muted
					return l.Layout(gtx)
				}),
				layout.Flexed(1, layout.Spacer{Width: 0}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					l := material.Label(ui.th, unit.Sp(11), "v1.0.0")
					l.Color = ui.th.Pal.Muted
					return l.Layout(gtx)
				}),
			)
		}),
	)
}

func (ui *UI) content(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(ui.topbar),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return material.List(ui.th, &ui.pageList).Layout(gtx, 1, func(gtx layout.Context, _ int) layout.Dimensions {
				switch ui.current {
				case PageNetwork:
					return ui.networkPage(gtx)
				case PageInternet:
					return ui.internetPage(gtx)
				case PageServices:
					return ui.servicesPage(gtx)
				case PageLogs:
					return ui.logsPage(gtx)
				case PageDocker:
					return ui.dockerPage(gtx)
				case PagePrinters:
					return ui.printersPage(gtx)
				case PageApps:
					return ui.appsPage(gtx)
				default:
					return ui.overviewPage(gtx)
				}
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if ui.opsOpen {
				return ui.opsPanel(gtx)
			}
			return layout.Dimensions{}
		}),
	)
}

func (ui *UI) topbar(gtx layout.Context) layout.Dimensions {
	title := map[Page]string{
		PageOverview: "Overview", PageNetwork: "Network Diagnostics",
		PageInternet: "Internet Connectivity", PageServices: "Services",
		PageLogs: "Log Viewer", PageDocker: "Docker",
		PagePrinters: "Printers", PageApps: "Applications",
	}[ui.current]
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			paint.FillShape(gtx.Ops, ui.th.Pal.Bg, clip.Rect{Max: gtx.Constraints.Max}.Op())
			return layout.Dimensions{Size: gtx.Constraints.Max}
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: 12, Bottom: 12, Left: 22, Right: 16}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						l := material.Label(ui.th, unit.Sp(19), title)
						l.Font.Weight = 700
						return l.Layout(gtx)
					}),
					layout.Flexed(1, layout.Spacer{Width: 0}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.statusDot(gtx, ui.liveColor(), 8)
					}),
					layout.Rigid(layout.Spacer{Width: 8}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						l := material.Label(ui.th, unit.Sp(12), ui.liveLabel())
						l.Color = ui.th.Pal.Muted
						return l.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Width: 16}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.iconButton(gtx, &ui.opsToggle, icons.ActionList, "Operations")
					}),
				)
			})
		}),
	)
}

func (ui *UI) liveColor() color.NRGBA {
	if ui.acts.Running() {
		return ui.th.Pal.Accent
	}
	return ui.th.Pal.Success
}

func (ui *UI) liveLabel() string {
	if ui.acts.Running() {
		return "working…"
	}
	return "live"
}

func (ui *UI) opsPanel(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(300))
	return ui.card(gtx, ui.th.Pal.Card, radiusCard, 0, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: 12, Left: 16, Right: 16, Bottom: 8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							l := material.Label(ui.th, unit.Sp(14), "Operations")
							l.Font.Weight = 600
							return l.Layout(gtx)
						}),
						layout.Flexed(1, layout.Spacer{Width: 0}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							btn := material.IconButton(ui.th, &ui.opsToggle, icons.NavigationClose, "Close")
							btn.Background = ui.th.Pal.CardAlt
							btn.Color = ui.th.Pal.Muted
							btn.Size = unit.Dp(16)
							return btn.Layout(gtx)
						}),
					)
				})
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				ops := ui.acts.Ops()
				items := len(ops)
				if items > 0 {
					// newest first
					return material.List(ui.th, &ui.opsList).Layout(gtx, items, func(gtx layout.Context, i int) layout.Dimensions {
						op := ops[items-1-i]
						return ui.opRow(gtx, op)
					})
				}
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return ui.muted(gtx, "No operations yet")
				})
			}),
		)
	})
}

func (ui *UI) opRow(gtx layout.Context, op *actions.Op) layout.Dimensions {
	hdr, ok := ui.opHeader[op.ID]
	if !ok {
		hdr = &widget.Clickable{}
		ui.opHeader[op.ID] = hdr
	}
	if hdr.Clicked(gtx) {
		ui.opOpen[op.ID] = !ui.opOpen[op.ID]
	}

	var statusIcon *widget.Icon
	var statusColor color.NRGBA
	var statusText string
	switch op.Status {
	case actions.StatusRunning:
		statusIcon = icons.AvLoop
		statusColor = ui.th.Pal.Accent
		statusText = fmt.Sprintf("%.0fs", time.Since(op.Started).Seconds())
	case actions.StatusError:
		statusIcon = icons.AlertError
		statusColor = ui.th.Pal.Danger
		statusText = "failed"
	default:
		statusIcon = icons.ActionCheckCircle
		statusColor = ui.th.Pal.Success
		statusText = fmt.Sprintf("%.0fs", op.Finished.Sub(op.Started).Seconds())
	}

	return layout.Inset{Top: 2, Bottom: 2, Left: 14, Right: 14}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return material.Clickable(gtx, hdr, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: 6, Bottom: 6}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return statusIcon.Layout(gtx, statusColor)
							}),
							layout.Rigid(layout.Spacer{Width: 10}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								l := material.Label(ui.th, unit.Sp(13), op.Label)
								l.Color = ui.th.Pal.Fg
								return l.Layout(gtx)
							}),
							layout.Flexed(1, layout.Spacer{Width: 0}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								l := material.Label(ui.th, unit.Sp(11), statusText)
								l.Color = ui.th.Pal.Muted
								return l.Layout(gtx)
							}),
						)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if ui.opOpen[op.ID] {
							return ui.opLog(gtx, op)
						}
						if op.Status == actions.StatusError && op.Err != nil {
							return layout.Inset{Top: 4, Left: 30}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								l := material.Label(ui.th, unit.Sp(12), op.Err.Error())
								l.Color = ui.th.Pal.Danger
								return l.Layout(gtx)
							})
						}
						return layout.Dimensions{}
					}),
				)
			})
		})
	})
}

func (ui *UI) opLog(gtx layout.Context, op *actions.Op) layout.Dimensions {
	list, ok := ui.opLogs[op.ID]
	if !ok {
		list = &widget.List{}
		list.Axis = layout.Vertical
		ui.opLogs[op.ID] = list
	}
	logs := op.Log
	if len(logs) > 120 {
		logs = logs[len(logs)-120:]
	}
	return layout.Inset{Top: 6, Left: 30}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(110))
		return material.List(ui.th, list).Layout(gtx, len(logs), func(gtx layout.Context, i int) layout.Dimensions {
			l := material.Label(ui.th, unit.Sp(11), logs[i])
			l.Color = ui.th.Pal.Muted
			l.MaxLines = 0
			return l.Layout(gtx)
		})
	})
}

func (ui *UI) showDialog(title, message, confirmLabel string, danger bool, onConfirm func()) {
	ui.dialog.open = true
	ui.dialog.title = title
	ui.dialog.message = message
	ui.dialog.confirmLabel = confirmLabel
	ui.dialog.danger = danger
	ui.dialog.onConfirm = onConfirm
	ui.win.Invalidate()
}

func (ui *UI) closeDialog() {
	ui.dialog.open = false
	ui.dialog.onConfirm = nil
}

func (ui *UI) renderDialog(gtx layout.Context) layout.Dimensions {
	if ui.dialog.scrim.Clicked(gtx) {
		ui.closeDialog()
	}
	if ui.dialog.cancel.Clicked(gtx) {
		ui.closeDialog()
	}
	if ui.dialog.confirm.Clicked(gtx) {
		fn := ui.dialog.onConfirm
		ui.closeDialog()
		if fn != nil {
			fn()
		}
	}
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			material.Clickable(gtx, &ui.dialog.scrim, func(gtx layout.Context) layout.Dimensions {
				paint.FillShape(gtx.Ops, color.NRGBA{R: 0, G: 0, B: 0, A: 170}, clip.Rect{Max: gtx.Constraints.Max}.Op())
				return layout.Dimensions{Size: gtx.Constraints.Max}
			})
			return layout.Dimensions{Size: gtx.Constraints.Max}
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Max.X = gtx.Dp(unit.Dp(430))
				gtx.Constraints.Min.X = 0
				return ui.card(gtx, ui.th.Pal.Card, radiusCard, 22, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							l := material.Label(ui.th, unit.Sp(17), ui.dialog.title)
							l.Font.Weight = 700
							return l.Layout(gtx)
						}),
						layout.Rigid(layout.Spacer{Height: 10}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							l := material.Label(ui.th, unit.Sp(13), ui.dialog.message)
							l.Color = ui.th.Pal.Muted
							return l.Layout(gtx)
						}),
						layout.Rigid(layout.Spacer{Height: 22}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
								layout.Flexed(1, layout.Spacer{Width: 0}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return ui.ghostButton(gtx, &ui.dialog.cancel, "Cancel")
								}),
								layout.Rigid(layout.Spacer{Width: 10}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									if ui.dialog.danger {
										return ui.dangerButton(gtx, &ui.dialog.confirm, ui.dialog.confirmLabel)
									}
									return ui.primaryButton(gtx, &ui.dialog.confirm, ui.dialog.confirmLabel)
								}),
							)
						}),
					)
				})
			})
		}),
	)
}
