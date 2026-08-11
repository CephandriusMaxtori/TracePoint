package ui

import (
	"context"
	"fmt"
	"strings"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"tracepoint/internal/collectors/docker"
	"tracepoint/internal/state"
)

type dockerLogState struct {
	open  bool
	click widget.Clickable
	list  widget.List
	text  string
}

type dockerState struct {
	list     widget.List
	startBtn map[string]*widget.Clickable
	stopBtn  map[string]*widget.Clickable
	restart  map[string]*widget.Clickable
	logs     map[string]*dockerLogState
	showAll  widget.Bool
}

func (ui *UI) dockerPage(gtx layout.Context) layout.Dimensions {
	ds := &ui.docker
	if ds.startBtn == nil {
		ds.startBtn = map[string]*widget.Clickable{}
		ds.stopBtn = map[string]*widget.Clickable{}
		ds.restart = map[string]*widget.Clickable{}
		ds.logs = map[string]*dockerLogState{}
	}

	for id, click := range ds.startBtn {
		if click.Clicked(gtx) {
			ui.dockerAction("Start container "+id, id, ui.col.Docker.Start)
		}
	}
	for id, click := range ds.stopBtn {
		if click.Clicked(gtx) {
			ui.dockerAction("Stop container "+id, id, ui.col.Docker.Stop)
		}
	}
	for id, click := range ds.restart {
		if click.Clicked(gtx) {
			ui.dockerAction("Restart container "+id, id, ui.col.Docker.Restart)
		}
	}
	for id, lg := range ds.logs {
		if lg.click.Clicked(gtx) {
			lg.open = !lg.open
			if lg.open {
				ui.fetchDockerLogs(id, lg)
			}
		}
	}

	var dock state.Docker
	ui.st.Read(func(s *state.Store) { dock = s.Docker })

	return ui.pageInset(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.dockerHeader(gtx, dock)
			}),
			layout.Rigid(layout.Spacer{Height: 14}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.card(gtx, ui.th.Pal.Card, radiusCard, 16, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return ui.sectionTitle(gtx, "Containers")
								}),
								layout.Flexed(1, layout.Spacer{Width: 0}.Layout),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return ui.caption(gtx, "Show all")
										}),
										layout.Rigid(layout.Spacer{Width: 6}.Layout),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return material.Switch(ui.th.Theme, &ds.showAll, "show all").Layout(gtx)
										}),
									)
								}),
							)
						}),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							containers := dock.Containers
							if !ds.showAll.Value {
								var running []state.DockerContainer
								for _, c := range containers {
									if c.Running {
										running = append(running, c)
									}
								}
								containers = running
							}
							if len(containers) == 0 {
								return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return ui.muted(gtx, "No containers")
								})
							}
							gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(620))
							return material.List(ui.th.Theme, &ds.list).Layout(gtx, len(containers), func(gtx layout.Context, i int) layout.Dimensions {
								return ui.containerRow(gtx, containers[i])
							})
						}),
					)
				})
			}),
		)
	})
}

func (ui *UI) dockerHeader(gtx layout.Context, dock state.Docker) layout.Dimensions {
	return ui.card(gtx, ui.th.Pal.Card, radiusCard, 16, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				c := ui.th.Pal.Danger
				label := "not connected"
				if dock.Connected {
					c = ui.th.Pal.Success
					label = "connected"
				}
				return ui.statusDot(gtx, c, 14)
			}),
			layout.Rigid(layout.Spacer{Width: 12}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						l := material.Label(ui.th.Theme, unit.Sp(18), "Docker Engine")
						l.Font.Weight = 700
						return l.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if dock.Err != "" {
							l := material.Label(ui.th.Theme, unit.Sp(11), dock.Err)
							l.Color = ui.th.Pal.Danger
							return l.Layout(gtx)
						}
						return ui.caption(gtx, dock.Version)
					}),
				)
			}),
			layout.Flexed(1, layout.Spacer{Width: 0}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.caption(gtx, fmt.Sprintf("%d container(s) · updated %s", len(dock.Containers), dock.UpdatedAt.Format("15:04:05")))
			}),
		)
	})
}

func (ui *UI) dockerAction(label, id string, fn func(ctx context.Context, id string) error) {
	ui.acts.RunErr(label, func(ctx context.Context, log func(format string, args ...any)) error {
		log("running %s…", label)
		err := fn(ctx, id)
		if err == nil {
			log("done")
			if ui.col.Docker != nil {
				ui.col.Docker.Refresh(ctx)
			}
		}
		return err
	})
}

func (ui *UI) fetchDockerLogs(id string, lg *dockerLogState) {
	ui.acts.RunErr("Fetch logs "+id, func(ctx context.Context, log func(format string, args ...any)) error {
		log("fetching logs for %s…", id)
		text, err := ui.col.Docker.Logs(ctx, id, 200)
		lg.text = text
		ui.win.Invalidate()
		return err
	})
}

func (ui *UI) containerRow(gtx layout.Context, c state.DockerContainer) layout.Dimensions {
	ds := &ui.docker
	start := clickFor(&ds.startBtn, c.ID)
	stop := clickFor(&ds.stopBtn, c.ID)
	restart := clickFor(&ds.restart, c.ID)
	lg, ok := ds.logs[c.ID]
	if !ok {
		lg = &dockerLogState{}
		ds.logs[c.ID] = lg
	}

	stateColor := ui.th.Pal.Muted
	if c.Running {
		stateColor = ui.th.Pal.Success
	}

	return layout.Inset{Top: 6, Bottom: 6}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.statusDot(gtx, stateColor, 8)
					}),
					layout.Rigid(layout.Spacer{Width: 10}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								l := material.Label(ui.th.Theme, unit.Sp(13), c.Name)
								l.Font.Weight = 600
								l.Color = ui.th.Pal.Fg
								return l.Layout(gtx)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return ui.caption(gtx, fmt.Sprintf("%s · %s", c.Image, c.Status))
							}),
						)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if c.Running {
							return ui.caption(gtx, fmt.Sprintf("CPU %.0f%%  MEM %.0f%%", c.CPU, c.MemPct))
						}
						return ui.caption(gtx, "stopped")
					}),
					layout.Rigid(layout.Spacer{Width: 10}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.pill(gtx, start, "Start", ui.enabledColor(!c.Running), ui.th.Pal.Fg, 600)
					}),
					layout.Rigid(layout.Spacer{Width: 6}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.pill(gtx, stop, "Stop", ui.enabledColor(c.Running), ui.th.Pal.Fg, 600)
					}),
					layout.Rigid(layout.Spacer{Width: 6}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.pill(gtx, restart, "Restart", ui.th.Pal.CardAlt, ui.th.Pal.Fg, 600)
					}),
					layout.Rigid(layout.Spacer{Width: 6}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.ghostButton(gtx, &lg.click, "Logs")
					}),
				)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if !lg.open {
					return layout.Dimensions{}
				}
				return layout.Inset{Top: 8, Left: 18}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(220))
					if lg.text == "" {
						return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return ui.muted(gtx, "No logs")
						})
					}
					lines := splitLines(lg.text)
					return material.List(ui.th.Theme, &lg.list).Layout(gtx, len(lines), func(gtx layout.Context, i int) layout.Dimensions {
						l := material.Label(ui.th.Theme, unit.Sp(11), lines[i])
						l.Color = ui.th.Pal.Muted
						return l.Layout(gtx)
					})
				})
			}),
		)
	})
}

func splitLines(s string) []string {
	return strings.Split(s, "\n")
}
