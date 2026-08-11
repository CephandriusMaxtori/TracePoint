package ui

import (
	"image"
	"math"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"image/color"
)

const (
	radiusCard    = 12
	radiusPill    = 20
	radiusControl = 8
)

func (ui *UI) card(gtx layout.Context, bg color.NRGBA, radius int, pad unit.Dp, w layout.Widget) layout.Dimensions {
	return layout.Background{}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			r := image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)
			rr := clip.UniformRRect(r, radius)
			paint.FillShape(gtx.Ops, bg, rr.Op(gtx.Ops))
			border := clip.Stroke{Path: rr.Path(gtx.Ops), Width: 1}
			paint.FillShape(gtx.Ops, ui.th.Pal.Border, border.Op())
			return layout.Dimensions{Size: gtx.Constraints.Max}
		},
		func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: pad, Bottom: pad, Left: pad, Right: pad}.Layout(gtx, w)
		},
	)
}

func (ui *UI) sectionTitle(gtx layout.Context, title string) layout.Dimensions {
	return layout.Inset{Top: 6, Bottom: 10}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return material.Label(ui.th, unit.Sp(15), title).Layout(gtx)
	})
}

func (ui *UI) muted(gtx layout.Context, text string) layout.Dimensions {
	l := material.Label(ui.th, unit.Sp(12), text)
	l.Color = ui.th.Pal.Muted
	return l.Layout(gtx)
}

func (ui *UI) caption(gtx layout.Context, text string) layout.Dimensions {
	l := material.Label(ui.th, unit.Sp(11), text)
	l.Color = ui.th.Pal.Muted
	return l.Layout(gtx)
}

func (ui *UI) statusDot(gtx layout.Context, c color.NRGBA, size int) layout.Dimensions {
	circle := clip.Circle{Center: f32.Pt(float32(size)/2, float32(size)/2), Radius: float32(size) / 2}
	paint.FillShape(gtx.Ops, c, circle.Op(gtx.Ops))
	return layout.Dimensions{Size: image.Pt(size, size)}
}

// ring draws a circular progress gauge.
func (ui *UI) ring(gtx layout.Context, frac float32, c color.NRGBA, size int) layout.Dimensions {
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	r := float32(size)/2 - 6
	center := f32.Pt(float32(size)/2, float32(size)/2)
	track := ui.th.Pal.CardAlt
	stroke := clip.Stroke{Path: circlePath(gtx.Ops, center.X, center.Y, r), Width: 7}
	paint.FillShape(gtx.Ops, track, stroke.Op())
	if frac > 0.01 {
		arc := arcPath(gtx.Ops, center.X, center.Y, r, -90, -90+360*frac, 72)
		paint.FillShape(gtx.Ops, c, clip.Stroke{Path: arc, Width: 7}.Op())
	}
	return layout.Dimensions{Size: image.Pt(size, size)}
}

// sparkline draws a polyline of history samples.
func sparkline(ops *op.Ops, data []float64, c color.NRGBA, w, h int) {
	if len(data) < 2 {
		return
	}
	max := 0.0
	for _, v := range data {
		if v > max {
			max = v
		}
	}
	if max <= 0 {
		max = 1
	}
	var p clip.Path
	p.Begin(ops)
	for i, v := range data {
		x := float32(i) / float32(len(data)-1) * float32(w-2)
		y := float32(h-2) - float32(v)/float32(max)*float32(h-4)
		if i == 0 {
			p.MoveTo(f32.Pt(x+1, y+1))
		} else {
			p.LineTo(f32.Pt(x+1, y+1))
		}
	}
	paint.FillShape(ops, c, clip.Stroke{Path: p.End(), Width: 2}.Op())
}

func hbar(gtx layout.Context, frac float32, fill, track color.NRGBA, width, height int) layout.Dimensions {
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	rr := clip.UniformRRect(image.Rect(0, 0, width, height), height/2)
	paint.FillShape(gtx.Ops, track, rr.Op(gtx.Ops))
	if frac > 0.005 {
		fw := int(float32(width) * frac)
		if fw < height {
			fw = height
		}
		r2 := clip.UniformRRect(image.Rect(0, 0, fw, height), height/2)
		paint.FillShape(gtx.Ops, fill, r2.Op(gtx.Ops))
	}
	return layout.Dimensions{Size: image.Pt(width, height)}
}

func (ui *UI) button(gtx layout.Context, clickable *widget.Clickable, label string, bg color.NRGBA, fg color.NRGBA) layout.Dimensions {
	btn := material.Button(ui.th, clickable, label)
	btn.Background = bg
	btn.Color = fg
	btn.CornerRadius = unit.Dp(radiusControl)
	btn.Inset = layout.Inset{Top: 7, Bottom: 7, Left: 12, Right: 12}
	return btn.Layout(gtx)
}

func (ui *UI) primaryButton(gtx layout.Context, clickable *widget.Clickable, label string) layout.Dimensions {
	return ui.button(gtx, clickable, label, ui.th.Pal.Accent, ui.th.Pal.TextOnAccent)
}

func (ui *UI) ghostButton(gtx layout.Context, clickable *widget.Clickable, label string) layout.Dimensions {
	return ui.button(gtx, clickable, label, ui.th.Pal.CardAlt, ui.th.Pal.Fg)
}

func (ui *UI) dangerButton(gtx layout.Context, clickable *widget.Clickable, label string) layout.Dimensions {
	return ui.button(gtx, clickable, label, ui.th.Pal.Danger, ui.th.Pal.TextOnAccent)
}

func (ui *UI) successButton(gtx layout.Context, clickable *widget.Clickable, label string) layout.Dimensions {
	return ui.button(gtx, clickable, label, ui.th.Pal.Success, ui.th.Pal.TextOnAccent)
}

func (ui *UI) iconButton(gtx layout.Context, clickable *widget.Clickable, icon *widget.Icon, tooltip string) layout.Dimensions {
	ib := material.IconButton(ui.th, clickable, icon, tooltip)
	ib.Background = ui.th.Pal.CardAlt
	ib.Color = ui.th.Pal.Fg
	ib.Size = unit.Dp(18)
	return ib.Layout(gtx)
}

func (ui *UI) pill(gtx layout.Context, clickable *widget.Clickable, label string, bg color.NRGBA, fg color.NRGBA, weight float32) layout.Dimensions {
	btn := material.Button(ui.th, clickable, label)
	btn.Background = bg
	btn.Color = fg
	btn.CornerRadius = unit.Dp(radiusPill)
	btn.Inset = layout.Inset{Top: 5, Bottom: 5, Left: 14, Right: 14}
	btn.Font.Weight = 600
	return btn.Layout(gtx)
}

func circlePath(ops *op.Ops, cx, cy, r float32) clip.PathSpec {
	var p clip.Path
	p.Begin(ops)
	const n = 64
	for i := 0; i < n; i++ {
		a := float32(i) / n * 2 * math.Pi
		x := cx + r*float32(math.Cos(float64(a)))
		y := cy + r*float32(math.Sin(float64(a)))
		if i == 0 {
			p.MoveTo(f32.Pt(x, y))
		} else {
			p.LineTo(f32.Pt(x, y))
		}
	}
	p.Close()
	return p.End()
}

func arcPath(ops *op.Ops, cx, cy, r, a0, a1 float32, segs int) clip.PathSpec {
	var p clip.Path
	p.Begin(ops)
	for i := 0; i <= segs; i++ {
		a := (a0 + (a1-a0)*float32(i)/float32(segs)) * math.Pi / 180
		x := cx + r*float32(math.Cos(float64(a)))
		y := cy + r*float32(math.Sin(float64(a)))
		if i == 0 {
			p.MoveTo(f32.Pt(x, y))
		} else {
			p.LineTo(f32.Pt(x, y))
		}
	}
	return p.End()
}
