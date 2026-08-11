package ui

import (
	"context"
	"fmt"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"tracepoint/internal/collectors/internet"
	"tracepoint/internal/state"
)

type internetState struct {
	retestBtn  widget.Clickable
	checksList widget.List
	busy       bool
}

func (ui *UI) internetPage(gtx layout.Context) layout.Dimensions {
	ins := &ui.internet
	if ins.retestBtn.Clicked(gtx) {
		ins.busy = true
		ui.acts.Run("Internet connectivity check", func(ctx context.Context, log func(format string, args ...any)) {
			ui.ic.Check(ctx)
			ins.busy = false
			ui.win.Invalidate()
		})
	}

	var internetData state.Internet
	ui.st.Read(func(s *state.Store) { internetData = s.Internet })

	overall := internet.Overall(internetData.Checks)
	overallColor := ui.th.statusColor(int(overall))

	return ui.pageInset(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.card(gtx, ui.th.Pal.Card, radiusCard, 16, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.statusDot(gtx, overallColor, 16)
						}),
						layout.Rigid(layout.Spacer{Width: 14}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									l := material.Label(ui.th.Theme, unit.Sp(20), "Internet " + overall.String())
									l.Font.Weight = 700
									l.Color = overallColor
									return l.Layout(gtx)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return ui.caption(gtx, fmt.Sprintf("%d/%d checks passing · updated %s", okCount(internetData.Checks), len(internetData.Checks), internetData.UpdatedAt.Format("15:04:05")))
								}),
							)
						}),
						layout.Flexed(1, layout.Spacer{Width: 0}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if ins.busy {
								return ui.ghostButton(gtx, &ins.retestBtn, "Testing…")
							}
							return ui.primaryButton(gtx, &ins.retestBtn, "Re-test")
						}),
					)
				})
			}),
			layout.Rigid(layout.Spacer{Height: 14}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.card(gtx, ui.th.Pal.Card, radiusCard, 16, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.sectionTitle(gtx, "Connectivity Checks")
						}),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							checks := internetData.Checks
							if len(checks) == 0 {
								return ui.muted(gtx, "No checks yet")
							}
							gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(340))
							return material.List(ui.th.Theme, &ins.checksList).Layout(gtx, len(checks), func(gtx layout.Context, i int) layout.Dimensions {
								return ui.checkRow(gtx, checks[i])
							})
						}),
					)
				})
			}),
		)
	})
}

func okCount(checks []state.CheckResult) int {
	n := 0
	for _, c := range checks {
		if c.Status == state.StatusOK {
			n++
		}
	}
	return n
}

func (ui *UI) checkRow(gtx layout.Context, c state.CheckResult) layout.Dimensions {
	col := ui.th.statusColor(int(c.Status))
	return layout.Inset{Top: 8, Bottom: 8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.statusDot(gtx, col, 10)
			}),
			layout.Rigid(layout.Spacer{Width: 12}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						l := material.Label(ui.th.Theme, unit.Sp(13), c.Name)
						l.Color = ui.th.Pal.Fg
						return l.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.caption(gtx, c.Detail)
					}),
				)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				l := material.Label(ui.th.Theme, unit.Sp(12), fmt.Sprintf("%.1f ms", c.Latency))
				l.Color = ui.th.Pal.Muted
				return l.Layout(gtx)
			}),
		)
	})
}
