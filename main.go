package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	appService := NewApp()

	app := application.New(application.Options{
		Name:        "Remnawave Manager (Beta)",
		Description: "Remnawave Deployment & Manager Suite",
		Services: []application.Service{
			application.NewService(appService),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "org.remnawave.ui.beta",
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
		Title:            "Remnawave Manager (Wails v3 Beta)",
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
	})

	err := app.Run()
	if err != nil {
		log.Fatal(err)
	}
}
