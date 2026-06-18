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
	// 创建 App 实例
	app := NewApp()

	// 创建应用并配置选项
	err := wails.Run(&options.App{
		Title:     "TerminalTree",
		Width:     1100,
		Height:    720,
		MinWidth:  720,
		MinHeight: 480,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		// Catppuccin Mocha 基础色 #1e1e2e
		BackgroundColour: &options.RGBA{R: 30, G: 30, B: 46, A: 1},
		// 配合 startup 中的延迟 WindowShow，规避 WebView2Loader 不触发 show 的问题
		StartHidden: true,
		OnStartup:   app.startup,
		OnShutdown:  app.shutdown,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			// 关闭 WebView GPU 加速，避免与 AI-CLI 长时间交互后的渲染花屏
			WebviewGpuIsDisabled: true,
			Theme:                windows.Dark,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
