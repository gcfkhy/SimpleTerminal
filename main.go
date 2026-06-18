package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:  "TerminalTree",
		Width:  1280,
		Height: 800,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 30, G: 30, B: 46, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			Theme: windows.Dark,
			CustomTheme: &windows.ThemeSettings{
				DarkModeTitleBar:   windows.RGB(30, 30, 46),
				DarkModeTitleText:  windows.RGB(205, 214, 244),
				DarkModeBorder:     windows.RGB(30, 30, 46),
				LightModeTitleBar:  windows.RGB(30, 30, 46),
				LightModeTitleText: windows.RGB(205, 214, 244),
				LightModeBorder:    windows.RGB(30, 30, 46),
			},
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
