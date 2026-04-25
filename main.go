package main

import (
	"embed"
	"io/fs"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend
var assets embed.FS

func main() {
	subFS, err := fs.Sub(assets, "frontend")
	if err != nil {
		log.Fatal(err)
	}

	app := NewApp()
	err = wails.Run(&options.App{
		Title:  "Stationeers Modding Installer",
		Width:  1100,
		Height: 660,
		AssetServer: &assetserver.Options{
			Assets: subFS,
		},
		BackgroundColour: &options.RGBA{R: 18, G: 18, B: 22, A: 1},
		OnStartup:        app.startup,
		Bind:             []interface{}{app},
	})
	if err != nil {
		log.Fatal(err)
	}
}
