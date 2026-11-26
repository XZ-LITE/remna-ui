//go:build !windows

package main

func ensurePlatformWebView() bool {
	// Linux uses WebKitGTK and macOS uses WKWebView provided by the OS/Wails runtime.
	return true
}
