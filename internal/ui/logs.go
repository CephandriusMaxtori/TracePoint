package ui

import (
	"context"
	"time"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"tracepoint/internal/collectors/logs"
)

type logsState struct {
	pathEditor   widget.Editor
	filterEditor widget.Editor
	followBtn    widget.Clickable
	stopBtn      widget.Clickable
	clearBtn     widget.Clickable
	list         widget.List

	tailer     *logs.Tailer
	tailCancel context.CancelFunc
	follow     bool
}

func (ui *UI) logsPage(gtx layout.Context) layout.Dimensions {
	ls := &ui.logs

	if ls.followBtn.Clicked(gtx) {
		ui.startTail(ls)
	}
	if ls.stopBtn.Clicked(gtx) {
		ui.stopTail(ls)
	}
	if ls.clearBtn.Clicked(gtx) {
		if ls.tailer != nil {
			ls.tailer.Clear()
		}
	}

	var lines []string
	var tailErr string
	following := false
	if ls.tailer != nil {
		lines = ls.tailer.Lines()
		tailErr = ls.tailer.Err()
		following = ls.tailer.Active()
	}

	if following {
		ls.list.ScrollToEnd()
	}

	return ui.pageInset(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.card(gtx, ui.th.Pal.Card, radiusCard, 14, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									l := material.Label(ui.th.Theme, unit.Sp(15), "Log Viewer")
									l.Font.Weight = 600
									return l.Layout(gtx)
								}),
								layout.Rigid(layout.Spacer{Width: 12}.Layout),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									c := ui.th.Pal.Muted
									label := "stopped"
									if following {
										c = ui.th.Pal.Success
										label = "following"
									}
									return ui.statusDot(gtx, c, 8)
								}),
								layout.Rigid(layout.Spacer{Width: 6}.Layout),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return ui.caption(gtx, label)
								}),
							)
						}),
						layout.Rigid(layout.Spacer{Height: 12}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
								layout.Flexed(2, func(gtx layout.Context) layout.Dimensions {
									ed := material.Editor(ui.th.Theme, &ls.pathEditor, "log file path, e.g. C:\\Windows\\WindowsUpdate.log")
									ed.Color = ui.th.Pal.Fg
									ed.HintColor = ui.th.Pal.Muted
									return ed.Layout(gtx)
								}),
								layout.Rigid(layout.Spacer{Width: 10}.Layout),
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									ed := material.Editor(ui.th.Theme, &ls.filterEditor, "filter (optional)")
									ed.Color = ui.th.Pal.Fg
									ed.HintColor = ui.th.Pal.Muted
									return ed.Layout(gtx)
								}),
								layout.Rigid(layout.Spacer{Width: 10}.Layout),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									if following {
										return ui.ghostButton(gtx, &ls.followBtn, "Re-tail")
									}
									return ui.primaryButton(gtx, &ls.followBtn, "Tail")
								}),
								layout.Rigid(layout.Spacer{Width: 6}.Layout),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return ui.ghostButton(gtx, &ls.stopBtn, "Stop")
								}),
								layout.Rigid(layout.Spacer{Width: 6}.Layout),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return ui.ghostButton(gtx, &ls.clearBtn, "Clear")
								}),
							)
						}),
					)
				})
			}),
			layout.Rigid(layout.Spacer{Height: 14}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.card(gtx, ui.th.Pal.Card, radiusCard, 14, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return ui.caption(gtx, "Output")
								}),
								layout.Flexed(1, layout.Spacer{Width: 0}.Layout),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return ui.caption(gtx, time.Now().Format("15:04:05"))
								}),
							)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if tailErr != "" {
								l := material.Label(ui.th.Theme, unit.Sp(12), tailErr)
								l.Color = ui.th.Pal.Danger
								return l.Layout(gtx)
							}
							return layout.Dimensions{}
						}),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(560))
							if len(lines) == 0 {
								return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return ui.muted(gtx, "No log lines yet")
								})
							}
							return material.List(ui.th.Theme, &ls.list).Layout(gtx, len(lines), func(gtx layout.Context, i int) layout.Dimensions {
								l := material.Label(ui.th.Theme, unit.Sp(11), lines[i])
								l.Color = ui.th.Pal.Fg
								return l.Layout(gtx)
							})
						}),
					)
				})
			}),
		)
	})
}

func (ui *UI) startTail(ls *logsState) {
	ui.stopTail(ls)
	path := ls.pathEditor.Text()
	if path == "" {
		return
	}
	ctx, cancel := context.WithCancel(ui.ctx)
	ls.tailCancel = cancel
	t := logs.NewTailer(path, ls.filterEditor.Text())
	ls.tailer = t
	ls.follow = true
	t.Start(ctx)

	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				ui.win.Invalidate()
			}
		}
	}()
}

func (ui *UI) stopTail(ls *logsState) {
	if ls.tailer != nil {
		ls.tailer.Stop()
	}
	if ls.tailCancel != nil {
		ls.tailCancel()
		ls.tailCancel = nil
	}
	ls.follow = false
}
