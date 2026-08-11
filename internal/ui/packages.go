package ui

import (
	"context"
	"fmt"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"tracepoint/internal/collectors/apps"
	"tracepoint/internal/state"
)

type appsState struct {
	searchEditor widget.Editor
	searchBtn    widget.Clickable
	searchBusy   bool
	searchErr    string
	searchList   widget.List

	installedList widget.List
	installBtn    map[string]*widget.Clickable
	upgradeBtn    map[string]*widget.Clickable
	uninstallBtn  map[string]*widget.Clickable
	upgradeAll    widget.Clickable
	refresh       widget.Clickable
}

func (ui *UI) appsPage(gtx layout.Context) layout.Dimensions {
	as := &ui.apps
	if as.installBtn == nil {
		as.installBtn = map[string]*widget.Clickable{}
		as.upgradeBtn = map[string]*widget.Clickable{}
		as.uninstallBtn = map[string]*widget.Clickable{}
	}

	ui.handleAppsActions(gtx)

	var pk state.Packages
	ui.st.Read(func(s *state.Store) { pk = s.Packages })

	return ui.pageInset(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.appsHeader(gtx, pk)
			}),
			layout.Rigid(layout.Spacer{Height: 14}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.card(gtx, ui.th.Pal.Card, radiusCard, 16, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.sectionTitle(gtx, "Installed Packages")
						}),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							if len(pk.Apps) == 0 {
								return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return ui.muted(gtx, "No installed packages")
								})
							}
							gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(520))
							return material.List(ui.th.Theme, &as.installedList).Layout(gtx, len(pk.Apps), func(gtx layout.Context, i int) layout.Dimensions {
								return ui.installedRow(gtx, pk, pk.Apps[i])
							})
						}),
					)
				})
			}),
			layout.Rigid(layout.Spacer{Height: 14}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.searchResultsCard(gtx, pk)
			}),
		)
	})
}

func (ui *UI) appsHeader(gtx layout.Context, pk state.Packages) layout.Dimensions {
	as := &ui.apps
	return ui.card(gtx, ui.th.Pal.Card, radiusCard, 16, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						c := ui.th.Pal.Danger
						name := "none"
						if pk.Available {
							c = ui.th.Pal.Success
							name = pk.Backend
						}
						return ui.statusDot(gtx, c, 12)
					}),
					layout.Rigid(layout.Spacer{Width: 10}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								l := material.Label(ui.th.Theme, unit.Sp(17), "Package Manager: "+name)
								l.Font.Weight = 700
								return l.Layout(gtx)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								if !pk.Available {
									l := material.Label(ui.th.Theme, unit.Sp(11), apps.PlatformHint())
									l.Color = ui.th.Pal.Warn
									return l.Layout(gtx)
								}
								return ui.caption(gtx, fmt.Sprintf("%s · %d installed · %d outdated", pk.Version, pk.Installed, pk.Outdated))
							}),
						)
					}),
					layout.Flexed(1, layout.Spacer{Width: 0}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.ghostButton(gtx, &as.refresh, "Refresh")
					}),
					layout.Rigid(layout.Spacer{Width: 8}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.primaryButton(gtx, &as.upgradeAll, "Upgrade All")
					}),
				)
			}),
			layout.Rigid(layout.Spacer{Height: 14}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						l := material.Label(ui.th.Theme, unit.Sp(13), "Search")
						l.Color = ui.th.Pal.Muted
						return l.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Width: 10}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						ed := material.Editor(ui.th.Theme, &as.searchEditor, "package name, e.g. firefox")
						ed.Color = ui.th.Pal.Fg
						ed.HintColor = ui.th.Pal.Muted
						return ed.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Width: 10}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if as.searchBusy {
							return ui.ghostButton(gtx, &as.searchBtn, "Searching…")
						}
						return ui.primaryButton(gtx, &as.searchBtn, "Search")
					}),
				)
			}),
		)
	})
}

func (ui *UI) searchResultsCard(gtx layout.Context, pk state.Packages) layout.Dimensions {
	as := &ui.apps
	return ui.card(gtx, ui.th.Pal.Card, radiusCard, 16, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.sectionTitle(gtx, "Search Results")
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if as.searchErr != "" {
					l := material.Label(ui.th.Theme, unit.Sp(12), as.searchErr)
					l.Color = ui.th.Pal.Danger
					return l.Layout(gtx)
				}
				return layout.Dimensions{}
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				if len(as.searchResults) == 0 {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return ui.muted(gtx, "Run a search to find packages")
					})
				}
				gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(360))
				return material.List(ui.th.Theme, &as.searchList).Layout(gtx, len(as.searchResults), func(gtx layout.Context, i int) layout.Dimensions {
					return ui.searchResultRow(gtx, pk, as.searchResults[i])
				})
			}),
		)
	})
}

func (ui *UI) handleAppsActions(gtx layout.Context) {
	as := &ui.apps
	if as.searchBtn.Clicked(gtx) {
		query := as.searchEditor.Text()
		if query == "" {
			return
		}
		var backend string
		ui.st.Read(func(s *state.Store) { backend = s.Packages.Backend })
		as.searchBusy = true
		ui.acts.RunErr("Search packages for "+query, func(ctx context.Context, log func(format string, args ...any)) error {
			res, err := apps.Search(ctx, backend, query, log)
			as.searchBusy = false
			as.searchErr = ""
			ui.win.Invalidate()
			if err != nil {
				as.searchErr = err.Error()
				return err
			}
			as.searchResults = res
			return nil
		})
	}
	if as.refresh.Clicked(gtx) {
		ui.acts.RunErr("Refresh packages", func(ctx context.Context, log func(format string, args ...any)) error {
			if ui.col.Apps != nil {
				ui.col.Apps.Refresh(ctx)
			}
			log("packages refreshed")
			return nil
		})
	}
	if as.upgradeAll.Clicked(gtx) {
		ui.upgradeAll(gtx)
	}
	for name, click := range as.installBtn {
		if click.Clicked(gtx) {
			ui.installPkg(gtx, name)
		}
	}
	for name, click := range as.upgradeBtn {
		if click.Clicked(gtx) {
			ui.upgradePkg(gtx, name)
		}
	}
	for name, click := range as.uninstallBtn {
		if click.Clicked(gtx) {
			ui.uninstallPkg(gtx, name)
		}
	}
}

func (ui *UI) backend() string {
	var b string
	ui.st.Read(func(s *state.Store) { b = s.Packages.Backend })
	return b
}

func (ui *UI) installPkg(gtx layout.Context, pkg string) {
	backend := ui.backend()
	ui.acts.RunErr("Install "+pkg, func(ctx context.Context, log func(format string, args ...any)) error {
		err := apps.Install(ctx, backend, pkg, log)
		if err == nil && ui.col.Apps != nil {
			ui.col.Apps.Refresh(ctx)
		}
		return err
	})
}

func (ui *UI) upgradePkg(gtx layout.Context, pkg string) {
	backend := ui.backend()
	ui.acts.RunErr("Upgrade "+pkg, func(ctx context.Context, log func(format string, args ...any)) error {
		err := apps.Upgrade(ctx, backend, pkg, log)
		if err == nil && ui.col.Apps != nil {
			ui.col.Apps.Refresh(ctx)
		}
		return err
	})
}

func (ui *UI) uninstallPkg(gtx layout.Context, pkg string) {
	backend := ui.backend()
	ui.acts.RunErr("Uninstall "+pkg, func(ctx context.Context, log func(format string, args ...any)) error {
		err := apps.Uninstall(ctx, backend, pkg, log)
		if err == nil && ui.col.Apps != nil {
			ui.col.Apps.Refresh(ctx)
		}
		return err
	})
}

func (ui *UI) upgradeAll(gtx layout.Context) {
	backend := ui.backend()
	ui.acts.RunErr("Upgrade all packages", func(ctx context.Context, log func(format string, args ...any)) error {
		err := apps.UpgradeAll(ctx, backend, log)
		if err == nil && ui.col.Apps != nil {
			ui.col.Apps.Refresh(ctx)
		}
		return err
	})
}

func (ui *UI) installedRow(gtx layout.Context, pk state.Packages, a state.App) layout.Dimensions {
	as := &ui.apps
	upgrade := clickFor(&as.upgradeBtn, a.Name)
	uninstall := clickFor(&as.uninstallBtn, a.Name)

	return layout.Inset{Top: 6, Bottom: 6}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				c := ui.th.Pal.Muted
				if a.Outdated {
					c = ui.th.Pal.Warn
				}
				return ui.statusDot(gtx, c, 8)
			}),
			layout.Rigid(layout.Spacer{Width: 10}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						l := material.Label(ui.th.Theme, unit.Sp(13), a.Name)
						l.Color = ui.th.Pal.Fg
						return l.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if a.Outdated {
							return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return ui.caption(gtx, fmt.Sprintf("%s → %s", a.Version, a.Latest))
								}),
							)
						}
						return ui.caption(gtx, a.Version)
					}),
				)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if a.Outdated {
					return ui.pill(gtx, upgrade, "Upgrade", ui.th.Pal.AccentDark, ui.th.Pal.TextOnAccent, 600)
				}
				return layout.Dimensions{}
			}),
			layout.Rigid(layout.Spacer{Width: 6}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.pill(gtx, uninstall, "Uninstall", ui.th.Pal.CardAlt, ui.th.Pal.Muted, 600)
			}),
		)
	})
}

func (ui *UI) searchResultRow(gtx layout.Context, pk state.Packages, a state.App) layout.Dimensions {
	as := &ui.apps
	install := clickFor(&as.installBtn, a.Name)

	return layout.Inset{Top: 6, Bottom: 6}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						l := material.Label(ui.th.Theme, unit.Sp(13), a.Name)
						l.Color = ui.th.Pal.Fg
						return l.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if a.Version != "" {
							return ui.caption(gtx, a.Version)
						}
						return layout.Dimensions{}
					}),
				)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.primaryButton(gtx, install, "Install")
			}),
		)
	})
}
