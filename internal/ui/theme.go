package ui

import (
	"image/color"

	"gioui.org/widget/material"
)

type Palette struct {
	Sidebar      color.NRGBA
	Card         color.NRGBA
	CardAlt      color.NRGBA
	Border       color.NRGBA
	Muted        color.NRGBA
	Success      color.NRGBA
	Warn         color.NRGBA
	Danger       color.NRGBA
	Accent       color.NRGBA
	AccentDark   color.NRGBA
	TextOnAccent color.NRGBA
	Fg           color.NRGBA
	Bg           color.NRGBA
}

func nrgb(r, g, b uint8) color.NRGBA {
	return color.NRGBA{R: r, G: g, B: b, A: 255}
}

func NewPalette() Palette {
	return Palette{
		Sidebar:      nrgb(0x11, 0x15, 0x1c),
		Card:         nrgb(0x18, 0x1d, 0x26),
		CardAlt:      nrgb(0x1f, 0x25, 0x30),
		Border:       nrgb(0x2a, 0x32, 0x40),
		Muted:        nrgb(0x8a, 0x95, 0xa8),
		Success:      nrgb(0x34, 0xd0, 0x92),
		Warn:         nrgb(0xe8, 0xa3, 0x3d),
		Danger:       nrgb(0xf0, 0x5a, 0x5f),
		Accent:       nrgb(0x4f, 0x8c, 0xff),
		AccentDark:   nrgb(0x2f, 0x6b, 0xd0),
		TextOnAccent: nrgb(0xf5, 0xf8, 0xff),
		Fg:           nrgb(0xe8, 0xec, 0xf4),
		Bg:           nrgb(0x0e, 0x11, 0x16),
	}
}

type Theme struct {
	*material.Theme
	Pal Palette
}

func NewTheme() *Theme {
	mt := material.NewTheme()
	mt.Palette = material.Palette{
		Bg:         nrgb(0x0e, 0x11, 0x16),
		Fg:         nrgb(0xe8, 0xec, 0xf4),
		ContrastBg: nrgb(0x4f, 0x8c, 0xff),
		ContrastFg: nrgb(0xf5, 0xf8, 0xff),
	}
	mt.TextSize = 14
	return &Theme{Theme: mt, Pal: NewPalette()}
}

func (t *Theme) statusColor(s int) color.NRGBA {
	switch s {
	case 1:
		return t.Pal.Success
	case 2:
		return t.Pal.Warn
	case 3:
		return t.Pal.Danger
	default:
		return t.Pal.Muted
	}
}

func (t *Theme) valueColor(v float64) color.NRGBA {
	switch {
	case v >= 90:
		return t.Pal.Danger
	case v >= 70:
		return t.Pal.Warn
	default:
		return t.Pal.Success
	}
}
