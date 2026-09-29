package main

import (
	"embed"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"bomix-app/backend"
	"bomix-app/backend/config"
	"bomix-app/backend/logger"
	"bomix-app/backend/platform"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// Wails uses Go's `embed` package to embed the frontend files into the binary.
// Any files in the frontend/dist folder will be embedded into the binary and
// made available to the frontend.
// See https://pkg.go.dev/embed for more information.

//go:embed all:frontend/dist
var assets embed.FS

func init() {
	// Register custom events
	application.RegisterEvent[string]("time")
	application.RegisterEvent[[]string]("files:dropped")
	application.RegisterEvent[[]string]("files-dropped")
	application.RegisterEvent[string]("file:open")
}

// main function serves as the application's entry point. It initializes the application, creates a window,
// and starts a goroutine that emits a time-based event every second. It subsequently runs the application and
// logs any error that might occur.
func main() {
	// Initialize logger
	logLogger := logger.NewLogger(1000)

	// 檢查啟動時命令列引數中是否包含 .bomx 檔案路徑（例如透過檔案總管雙擊檔案啟動）
	var initialBOMXPath string
	for _, arg := range os.Args[1:] {
		if strings.HasSuffix(strings.ToLower(arg), ".bomx") {
			if abs, err := filepath.Abs(arg); err == nil {
				if _, statErr := os.Stat(abs); statErr == nil {
					initialBOMXPath = abs
					break
				}
			}
		}
	}

	// Initialize config
	cfg, err := config.Load(config.GetConfigPath())
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	var mainWindow *application.WebviewWindow
	var wailsApp *application.App

	// Create a new Wails application by providing the necessary options.
	// Variables 'Name' and 'Description' are for application metadata.
	// 'Assets' configures the asset server with the 'FS' variable pointing to the frontend files.
	// 'Bind' is a list of Go struct instances. The frontend has access to the methods of these instances.
	// 'Mac' options tailor the application when running an macOS.
	wailsApp = application.New(application.Options{
		Name:        "bomix-app",
		Description: "BOMix - BOM Management Application",
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "com.angeloeyez.bomix",
			OnSecondInstanceLaunch: func(data application.SecondInstanceData) {
				if mainWindow != nil {
					mainWindow.UnMinimise()
					mainWindow.Focus()
				}
				for _, arg := range data.Args {
					if strings.HasSuffix(strings.ToLower(arg), ".bomx") {
						if abs, err := filepath.Abs(arg); err == nil {
							if _, statErr := os.Stat(abs); statErr == nil {
								logLogger.Info("第二實例被啟動並請求開啟 .bomx 檔案", "path", abs)
								wailsApp.Event.Emit("file:open", abs)
								break
							}
						}
					}
				}
			},
		},
	})

	// Initialize the backend App and register it as a service
	backendApp := backend.NewApp(wailsApp, logLogger, cfg)
	if initialBOMXPath != "" {
		backendApp.SetStartupFile(initialBOMXPath)
		logLogger.Info("應用程式由 .bomx 檔案關聯啟動", "path", initialBOMXPath)
	}
	wailsApp.RegisterService(application.NewService(backendApp))

	// Create a new window with the necessary options.
	// 'Title' is the title of the window.
	// 'Mac' options tailor the window when running on macOS.
	// 'BackgroundColour' is the background colour of the window.
	// 'URL' is the URL that will be loaded into the webview.
	mainWindow = wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "BOMIX",
		// Window sized to the golden ratio (1000 / 618 ≈ 1.618).
		Width:          1000,
		Height:         618,
		EnableFileDrop: true,
		Frameless:      true,
		Windows: application.WindowsWindow{
			DisableFramelessWindowDecorations: false,
			NonClientRegionSupport:            true,
			WebView2CompositionHosting:        false,
		},
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(6, 7, 15),
		URL:              "/",
		DevToolsEnabled:  isDevMode,
		KeyBindings:      setupKeyBindings(),
	})

	// 在背景執行檔案關聯與圖標檢查（完全非同步、無卡頓、具備中斷保護）
	go func() {
		platform.CheckAndSetupFileAssociation(logLogger)
	}()

	// 監聽視窗原生檔案拖放事件，取得真正的本機絕對路徑並推送至前端
	mainWindow.OnWindowEvent(events.Common.WindowFilesDropped, func(event *application.WindowEvent) {
		droppedFiles := event.Context().DroppedFiles()
		logLogger.Info("WindowFilesDropped received", "count", len(droppedFiles), "files", droppedFiles)
		if len(droppedFiles) > 0 {
			wailsApp.Event.Emit("files:dropped", droppedFiles)
			wailsApp.Event.Emit("files-dropped", droppedFiles)
		}
	})

	// Create a goroutine that emits an event containing the current time every second.
	// The frontend can listen to this event and update the UI accordingly.
	go func() {
		for {
			now := time.Now().Format(time.RFC1123)
			wailsApp.Event.Emit("time", now)
			time.Sleep(time.Second)
		}
	}()

	// 在開發模式下若有指定 FRONTEND_DEVSERVER_URL，先等待前端 Vite 伺服器啟動就緒
	waitForDevServer()

	// Run the application. This blocks until the application has been exited.
	if err := wailsApp.Run(); err != nil {
		log.Fatal(err)
	}
}
