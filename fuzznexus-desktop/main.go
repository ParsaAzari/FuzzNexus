package main

import (
	"embed"
	goruntime "runtime"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

func createApplicationMenu(app *App) *menu.Menu {
	appMenu := menu.NewMenu()

	if goruntime.GOOS == "darwin" {
		appMenu.Append(menu.AppMenu())
	}

	// Edit menu is vital for clipboard in WKWebView on macOS
	appMenu.Append(menu.EditMenu())

	if goruntime.GOOS == "darwin" {
		appMenu.Append(menu.WindowMenu())
	}

	helpMenu := appMenu.AddSubmenu("Help")
	helpMenu.AddText("Developer Website (parsaazari.com)", nil, func(_ *menu.CallbackData) {
		_ = app.OpenURL("https://parsaazari.com/")
	})
	helpMenu.AddText("GitHub Repository (FuzzNexus)", nil, func(_ *menu.CallbackData) {
		_ = app.OpenURL("https://github.com/ParsaAzari/FuzzNexus")
	})
	helpMenu.AddSeparator()
	helpMenu.AddText("About FuzzNexus", keys.CmdOrCtrl("i"), func(_ *menu.CallbackData) {
		if app.ctx != nil {
			runtime.EventsEmit(app.ctx, "open:about-modal")
		}
	})

	return appMenu
}

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:             "FuzzNexus",
		Width:             640,
		Height:            340,
		MinWidth:          640,
		MinHeight:         340,
		HideWindowOnClose: true,
		CSSDragProperty:   "--wails-draggable",
		CSSDragValue:      "drag",
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop:     true,
			DisableWebViewDrop: false,
		},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Menu:             createApplicationMenu(app),
		Mac: &mac.Options{
			TitleBar:                     mac.TitleBarHiddenInset(),
			Appearance:                   mac.DefaultAppearance,
			WebviewIsTransparent:         true,
			WindowIsTranslucent:          true,
			DisableEscapeExitsFullscreen: true,
			Preferences: &mac.Preferences{
				TabFocusesLinks:        mac.Enabled,
				TextInteractionEnabled: mac.Enabled,
				FullscreenEnabled:      mac.Enabled,
			},
			About: &mac.AboutInfo{
				Title:   "FuzzNexus",
				Message: "FuzzNexus\nAdvanced Web Reconnaissance & Fuzzing Hub",
			},
		},
		Windows: &windows.Options{
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			BackdropType:         windows.Mica,
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
