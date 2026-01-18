package main

import (
	"embed"

	"wipedisk_enterprise/internal/app"
	"wipedisk_enterprise/internal/config"
	"wipedisk_enterprise/internal/logging"
	"wipedisk_enterprise/internal/maintenance"
	"wipedisk_enterprise/internal/wipe"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

// main is entry point for Wails application
func main() {
	// Initialize logger
	logger, err := logging.NewEnterpriseLogger(config.Default(), false)
	if err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}

	// Initialize wipe engine
	wipeEngine := wipe.NewWipeEngine(logger)

	// Initialize maintenance runner
	maintenanceRunner := maintenance.NewMaintenanceRunner(logger)

	// Create an instance of app structure with dependencies
	appInstance := app.NewAppWithDependencies(logger, wipeEngine, maintenanceRunner)

	// Create application with options
	err = wails.Run(&options.App{
		Title:  "WipeDisk Enterprise v1.3.0",
		Width:  1200,
		Height: 800,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour:  &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:         appInstance.Startup,
		OnDomReady:        appInstance.DomReady,
		OnBeforeClose:     appInstance.BeforeClose,
		OnShutdown:        appInstance.Shutdown,
		Frameless:         false,
		StartHidden:       false,
		HideWindowOnClose: false,
		RGBA:              &options.RGBA{R: 0, G: 0, B: 0, A: 0},
	})

	if err != nil {
		panic("Error when running application: " + err.Error())
	}
}
