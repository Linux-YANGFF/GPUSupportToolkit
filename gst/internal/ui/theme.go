package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

var DarkTheme = &myTheme{}

type myTheme struct{}

func (t *myTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if name == theme.ColorBackground {
		if variant == theme.VariantDark {
			return color.RGBA{R: 30, G: 30, B: 30, A: 255}
		}
	}
	return theme.DefaultTheme().Color(name, variant)
}

func (t *myTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t *myTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *myTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}

// CurrentTheme 返回当前主题
func CurrentTheme() fyne.Theme {
	return DarkTheme
}
