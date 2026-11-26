package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	if !ensurePlatformWebView() {
		return
	}

	appService := NewApp()

	app := application.New(application.Options{
		Name:        "Remnawave Manager",
		Description: "Remnawave Deployment & Manager Suite",
		Services: []application.Service{
			application.NewService(appService),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "org.remnawave.ui",
			OnSecondInstanceLaunch: func(data application.SecondInstanceData) {
				if a := application.Get(); a != nil {
					if win := a.Window.Current(); win != nil {
						win.Show()
						win.Focus()
					}
				}
			},
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Remnawave Manager",
		Width:            1120,
		Height:           760,
		MinWidth:         960,
		MinHeight:        640,
		BackgroundColour: application.NewRGB(16, 20, 28),
		BackgroundType:   application.BackgroundTypeTranslucent,
		URL:              "/",
		Windows: application.WindowsWindow{
			BackdropType: application.Mica,
			Theme:        application.Dark,
		},
		Mac: application.MacWindow{
			Backdrop: application.MacBackdropTranslucent,
			TitleBar: application.MacTitleBarHiddenInset,
		},
	})

	err := app.Run()
	if err != nil {
		log.Fatal(err)
	}
}
