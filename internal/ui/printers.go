package ui

import (
	"context"
	"image/color"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"tracepoint/internal/collectors/printers"
	"tracepoint/internal/state"
)

type printersState struct {
	list        widget.List
	testBtn     map[string]*widget.Clickable
	refresh     widget.Clickable
	defaultPill widget.Clickable
}

func (ui *UI) printersPage(gtx layout.Context) layout.Dimensions {
	ps := &ui.printers
	if ps.testBtn == nil {
		ps.testBtn = map[string]*widget.Clickable{}
	}

	if ps.refresh.Clicked(gtx) {
		ui.acts.RunErr("Refresh printers", func(ctx context.Context, log func(format string, args ...any)) error {
			if ui.col.Printers == nil {
				return nil
			}
			ui.col.Printers.Refresh(ctx)
			log("printers refreshed")
			return nil
		})
	}
	for name, click := range ps.testBtn {
		if click.Clicked(gtx) {
			ui.acts.RunErr("Print test page on "+name, func(ctx context.Context, log func(format string, args ...any)) error {
				log("printing test page on %s…", name)
				err := printers.TestPage(ctx, name)
				if err == nil {
					log("test page sent")
				}
				return err
			})
		}
	}

	var printersList []state.Printer
	ui.st.Read(func(s *state.Store) { printersList = s.Printers })

	return ui.pageInset(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.card(gtx, ui.th.Pal.Card, radiusCard, 14, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							l := material.Label(ui.th.Theme, unit.Sp(15), "Printers")
							l.Font.Weight = 600
							return l.Layout(gtx)
						}),
						layout.Rigid(layout.Spacer{Width: 14}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.caption(gtx, "Windows / CUPS")
						}),
						layout.Flexed(1, layout.Spacer{Width: 0}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.ghostButton(gtx, &ps.refresh, "Refresh")
						}),
					)
				})
			}),
			layout.Rigid(layout.Spacer{Height: 14}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.card(gtx, ui.th.Pal.Card, radiusCard, 16, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.sectionTitle(gtx, "Print Queues")
						}),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							if len(printersList) == 0 {
								return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return ui.muted(gtx, "No printers detected")
								})
							}
							gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(520))
							return material.List(ui.th.Theme, &ps.list).Layout(gtx, len(printersList), func(gtx layout.Context, i int) layout.Dimensions {
								return ui.printerRow(gtx, printersList[i])
							})
						}),
					)
				})
			}),
		)
	})
}

func (ui *UI) printerRow(gtx layout.Context, p state.Printer) layout.Dimensions {
	ps := &ui.printers
	test := clickFor(&ps.testBtn, p.Name)

	var statusColor color.NRGBA
	switch p.Status {
	case "idle":
		statusColor = ui.th.Pal.Success
	case "printing", "warmup":
		statusColor = ui.th.Pal.Accent
	case "stopped":
		statusColor = ui.th.Pal.Warn
	default:
		statusColor = ui.th.Pal.Danger
	}

	return layout.Inset{Top: 8, Bottom: 8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.statusDot(gtx, statusColor, 8)
			}),
			layout.Rigid(layout.Spacer{Width: 10}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								l := material.Label(ui.th.Theme, unit.Sp(13), p.Name)
								l.Font.Weight = 600
								l.Color = ui.th.Pal.Fg
								return l.Layout(gtx)
							}),
							layout.Rigid(layout.Spacer{Width: 8}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								if p.Default {
									return ui.pill(gtx, &ps.defaultPill, "default", ui.th.Pal.AccentDark, ui.th.Pal.TextOnAccent, 600)
								}
								return layout.Dimensions{}
							}),
						)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.caption(gtx, p.Status)
					}),
				)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if p.Driver != "" {
					return ui.caption(gtx, p.Driver)
				}
				return layout.Dimensions{}
			}),
			layout.Rigid(layout.Spacer{Width: 12}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.ghostButton(gtx, test, "Test Page")
			}),
		)
	})
}
