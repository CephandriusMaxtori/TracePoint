package main

import (
	"context"
	"log"
	"os"

	"gioui.org/app"
	"gioui.org/op"
	"gioui.org/unit"

	"tracepoint/internal/actions"
	"tracepoint/internal/collectors/apps"
	"tracepoint/internal/collectors/docker"
	"tracepoint/internal/collectors/internet"
	"tracepoint/internal/collectors/network"
	"tracepoint/internal/collectors/printers"
	"tracepoint/internal/collectors/services"
	"tracepoint/internal/collectors/system"
	"tracepoint/internal/state"
	"tracepoint/internal/ui"
)

func main() {
	go func() {
		w := new(app.Window)
		w.Option(app.Title("TracePoint"), app.Size(unit.Dp(1200), unit.Dp(800)))
		if err := loop(w); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

func loop(w *app.Window) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	st := state.New()
	acts := actions.NewManager(func() { w.Invalidate() })

	sysCol := system.New(st)
	netCol := network.New(st)
	ic := internet.New(st)
	svcCol := services.New(st)
	dc := docker.New(st)
	prtCol := printers.New(st)
	appsCol := apps.New(st)

	go sysCol.Run(ctx)
	go netCol.Run(ctx)
	go ic.Run(ctx)
	go svcCol.Run(ctx)
	go dc.Run(ctx)
	go prtCol.Run(ctx)
	go appsCol.Run(ctx)

	col := ui.Collectors{
		Internet: ic,
		Docker:   dc,
		Services: svcCol,
		Apps:     appsCol,
		Printers: prtCol,
	}

	th := ui.NewTheme()
	u := ui.New(th, st, acts, col, ctx)
	u.SetWindow(w)

	var ops op.Ops
	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			cancel()
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			u.Layout(gtx)
			e.Frame(gtx.Ops)
		}
	}
}
