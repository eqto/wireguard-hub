package main

import (
	"embed"
	"log"

	"wireguardhub/internal/server"
	"wireguardhub/internal/wireguard"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	serverSvc := server.NewService()
	wgSvc := wireguard.NewService(serverSvc)

	app := application.New(application.Options{
		Name:        "WireguardHub",
		Description: "Manage WireGuard servers over SSH",
		Services: []application.Service{
			application.NewService(serverSvc),
			application.NewService(wgSvc),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "WireguardHub",
		Width:            1200,
		Height:           800,
		BackgroundColour: application.NewRGB(15, 17, 28),
		URL:              "/",
	})

	err := app.Run()
	if err != nil {
		log.Fatal(err)
	}
}
