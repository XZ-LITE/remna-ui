//go:build windows

package main

import (
	"github.com/wailsapp/wails/v3/pkg/w32"
	"golang.org/x/sys/windows/registry"
)

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

func ensurePlatformWebView() bool {
	if !isWebView2Installed() {
		w32.MessageBox(
			0,
			"Для запуска Remnawave UI требуется компонент Microsoft Edge WebView2 Runtime.\nПожалуйста, установите его с официального сайта Microsoft: https://developer.microsoft.com/microsoft-edge/webview2/",
			"Remnawave UI — Ошибка запуска",
			w32.MB_ICONERROR|w32.MB_OK,
		)
		return false
	}
	return true
}
