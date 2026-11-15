package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/w32"
	"golang.org/x/sys/windows/registry"
)

//go:embed all:frontend/dist
var assets embed.FS

func isWebView2Installed() bool {
	keys := []string{
		`SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}`,
		`SOFTWARE\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}`,
	}
	for _, root := range []registry.Key{registry.LOCAL_MACHINE, registry.CURRENT_USER} {
		for _, k := range keys {
			key, err := registry.OpenKey(root, k, registry.QUERY_VALUE)
			if err == nil {
				val, _, err := key.GetStringValue("pv")
				key.Close()
				if err == nil && val != "" && val != "0.0.0.0" {
					return true
				}
			}
		}
	}
	return false
}

func main() {
	if !isWebView2Installed() {
		w32.MessageBox(
			0,
			"Для запуска Remnawave UI требуется компонент Microsoft Edge WebView2 Runtime.\nПожалуйста, установите его с официального сайта Microsoft: https://developer.microsoft.com/microsoft-edge/webview2/",
			"Remnawave UI — Ошибка запуска",
			w32.MB_ICONERROR|w32.MB_OK,
		)
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
	})

	err := app.Run()
	if err != nil {
		log.Fatal(err)
	}
}
