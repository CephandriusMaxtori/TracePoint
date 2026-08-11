package ui

import (
	"context"
	"fmt"
	"image/color"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"tracepoint/internal/collectors/services"
	"tracepoint/internal/state"
)

type servicesState struct {
	list      widget.List
	start     map[string]*widget.Clickable
	stop      map[string]*widget.Clickable
	restart   map[string]*widget.Clickable
	stopName  string
	stopClick *widget.Clickable
	filter    widget.Editor
}

func (ui *UI) servicesPage(gtx layout.Context) layout.Dimensions {
	ss := &ui.services
	if ss.start == nil {
		ss.start = map[string]*widget.Clickable{}
		ss.stop = map[string]*widget.Clickable{}
		ss.restart = map[string]*widget.Clickable{}
	}

	for name, click := range ss.start {
		if click.Clicked(gtx) {
			ui.serviceAction("Start "+name, name, services.Start)
		}
	}
	for name, click := range ss.stop {
		if click.Clicked(gtx) {
			ss.stopName = name
			ss.stopClick = ss.stop[name]
			ui.showDialog("Stop Service",
				fmt.Sprintf("Stop the %q service?", name),
				"Stop", true, func() {
					ui.serviceAction("Stop "+name, name, services.Stop)
				})
		}
	}
	for name, click := range ss.restart {
		if click.Clicked(gtx) {
			ui.showDialog("Restart Service",
				fmt.Sprintf("Restart the %q service?", name),
				"Restart", true, func() {
					ui.serviceAction("Restart "+name, name, services.Restart)
				})
		}
	}

	var svcs []state.Service
	ui.st.Read(func(s *state.Store) { svcs = s.Services })

	filter := ss.filter.Text()
	var shown []state.Service
	for _, sv := range svcs {
		if filter == "" || containsFold(sv.Name, filter) || containsFold(sv.DisplayName, filter) {
			shown = append(shown, sv)
		}
	}

	return ui.pageInset(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.card(gtx, ui.th.Pal.Card, radiusCard, 14, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							l := material.Label(ui.th.Theme, unit.Sp(15), "Services")
							l.Font.Weight = 600
							return l.Layout(gtx)
						}),
						layout.Rigid(layout.Spacer{Width: 14}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.caption(gtx, fmt.Sprintf("%d total", len(svcs)))
						}),
						layout.Flexed(1, layout.Spacer{Width: 0}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(220))
							ed := material.Editor(ui.th.Theme, &ss.filter, "Filter…")
							ed.Color = ui.th.Pal.Fg
							ed.HintColor = ui.th.Pal.Muted
							return ed.Layout(gtx)
						}),
					)
				})
			}),
			layout.Rigid(layout.Spacer{Height: 14}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.card(gtx, ui.th.Pal.Card, radiusCard, 16, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.sectionTitle(gtx, "Windows Services")
						}),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							if len(shown) == 0 {
								return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return ui.muted(gtx, "No services found")
								})
							}
							gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(560))
							return material.List(ui.th.Theme, &ss.list).Layout(gtx, len(shown), func(gtx layout.Context, i int) layout.Dimensions {
								return ui.serviceRow(gtx, shown[i])
							})
						}),
					)
				})
			}),
		)
	})
}

func (ui *UI) serviceAction(label, name string, fn func(ctx context.Context, name string) error) {
	ui.acts.RunErr(label, func(ctx context.Context, log func(format string, args ...any)) error {
		log("running %s…", label)
		err := fn(ctx, name)
		if err == nil {
			log("done")
			if ui.col.Services != nil {
				ui.col.Services.Refresh(ctx)
			}
		}
		return err
	})
}

func (ui *UI) serviceRow(gtx layout.Context, sv state.Service) layout.Dimensions {
	ss := &ui.services
	start := clickFor(&ss.start, sv.Name)
	stop := clickFor(&ss.stop, sv.Name)
	restart := clickFor(&ss.restart, sv.Name)

	running := sv.State == "running" || sv.State == "starting"

	var stateColor color.NRGBA
	switch sv.State {
	case "running":
		stateColor = ui.th.Pal.Success
	case "stopped", "stopping":
		stateColor = ui.th.Pal.Muted
	case "starting", "paused":
		stateColor = ui.th.Pal.Warn
	default:
		stateColor = ui.th.Pal.Danger
	}

	return layout.Inset{Top: 5, Bottom: 5}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.statusDot(gtx, stateColor, 8)
			}),
			layout.Rigid(layout.Spacer{Width: 10}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						l := material.Label(ui.th.Theme, unit.Sp(13), sv.Name)
						l.Font.Weight = 600
						l.Color = ui.th.Pal.Fg
						return l.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.caption(gtx, sv.State)
					}),
				)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.pill(gtx, start, "Start", ui.enabledColor(!running), ui.th.Pal.Fg, 600)
			}),
			layout.Rigid(layout.Spacer{Width: 6}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.pill(gtx, stop, "Stop", ui.enabledColor(running), ui.th.Pal.Fg, 600)
			}),
			layout.Rigid(layout.Spacer{Width: 6}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.pill(gtx, restart, "Restart", ui.th.Pal.CardAlt, ui.th.Pal.Fg, 600)
			}),
		)
	})
}

// enabledColor returns the background for an action pill: dimmed when the
// action does not apply to the current state, accent when it does.
func (ui *UI) enabledColor(enabled bool) color.NRGBA {
	if !enabled {
		return ui.th.Pal.CardAlt
	}
	return ui.th.Pal.AccentDark
}

// clickFor returns the clickable for key in m, creating it if needed.
func clickFor(m *map[string]*widget.Clickable, key string) *widget.Clickable {
	if *m == nil {
		*m = map[string]*widget.Clickable{}
	}
	c, ok := (*m)[key]
	if !ok {
		c = &widget.Clickable{}
		(*m)[key] = c
	}
	return c
}

func containsFold(haystack, needle string) bool {
	return len(needle) == 0 || indexFold(haystack, needle) >= 0
}

func indexFold(haystack, needle string) int {
	if len(needle) == 0 {
		return 0
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if equalFold(haystack[i:i+len(needle)], needle) {
			return i
		}
	}
	return -1
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if 'A' <= ca && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if 'A' <= cb && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
