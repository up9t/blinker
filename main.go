package main

import (
	"context"
	"embed"
	"log/slog"
	"os"

	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"
	_ "modernc.org/sqlite"
)

// Wails uses Go's `embed` package to embed the frontend files into the binary.
// Any files in the frontend/dist folder will be embedded into the binary and
// made available to the frontend.
// See https://pkg.go.dev/embed for more information.

//go:embed all:frontend/dist
var assets embed.FS

const (
	NotificationEventSend = "notification:send"
)

const (
	BreakEventStart  = "break:start"
	BreakEventFinish = "break:finish"
)

const (
	overlayModeNone = iota
	overlayModeBlink
	overlayModeBreak
)

// far enough away that no monitor can show it
const offscreenPosition = -10000

// keep the window mapped but tiny, Wayland ignores requested positions
const (
	hiddenWidth  = 1
	hiddenHeight = 1
)

func init() {
	// Register a custom event whose associated data type is string.
	// This is not required, but the binding generator will pick up registered events
	// and provide a strongly typed JS/TS API for them.
	application.RegisterEvent[string](NotificationEventSend)
	application.RegisterEvent[string](BreakEventStart)
	application.RegisterEvent[string](BreakEventFinish)
}

// main function serves as the application's entry point. It initializes the application, creates a window,
// and starts a goroutine that emits a time-based event every second. It subsequently runs the application and
// logs any error that might occur.
func main() {
	appName := "Blinker"

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()

	notifier := notifications.New()

	dbPath := "./app.sqlite3"
	db, err := NewSqliteDB(dbPath)

	if err != nil {
		logger.Error("failed to setup sqlite",
			"path", dbPath,
			"err", err,
		)

		return
	}

	defer db.Close()

	breakpointRepo := NewBreakpointSqliteRepository("breakpoints", db)

	if err := breakpointRepo.Migrate(ctx); err != nil {
		logger.Error("Failed to do migration", "err", err)
		return
	}

	cameraRepo := NewCameraSqliteRepository(db)

	if err := cameraRepo.Migrate(ctx); err != nil {
		logger.Error("Failed to migrate camera tables", "err", err)
		return
	}

	breakpointService := NewBreakpointService(breakpointRepo)
	bpEvent, err := breakpointService.Start(ctx)

	if err != nil {
		logger.Error("Failed to start breakpoint service", "err", err)
		return
	}

	defer breakpointService.Stop()

	blinkSettingsRepo := NewBlinkSettingsSqliteRepository(db)

	if err := blinkSettingsRepo.Migrate(ctx); err != nil {
		logger.Error("Failed to migrate blink settings table", "err", err)
		return
	}

	blinkSettingsService := NewBlinkSettingsService(blinkSettingsRepo)

	if err := blinkSettingsService.Start(ctx); err != nil {
		logger.Error("Failed to load blink settings", "err", err)
		return
	}

	cameraService := NewCameraService(cameraRepo, blinkSettingsService.blinkTimeout)
	cameraEvent := cameraService.Start()

	defer cameraService.Stop()

	// Create a new Wails application by providing the necessary options.
	// Variables 'Name' and 'Description' are for application metadata.
	// 'Assets' configures the asset server with the 'FS' variable pointing to the frontend files.
	// 'Bind' is a list of Go struct instances. The frontend has access to the methods of these instances.
	// 'Mac' options tailor the application when running an macOS.
	app := application.New(application.Options{
		Name:        appName,
		Description: "Helps lubricate eyes by forcing user to blink.",
		Services: []application.Service{
			application.NewService(breakpointService),
			application.NewService(cameraService),
			application.NewService(blinkSettingsService),
			application.NewService(notifier),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	// Create a new window with the necessary options.
	// 'Title' is the title of the window.
	// 'Mac' options tailor the window when running on macOS.
	// 'BackgroundColour' is the background colour of the window.
	// 'URL' is the URL that will be loaded into the webview.
	mainWindow := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "Blinker",
		Name:  "main",
		// Window sized to the golden ratio (1000 / 618 ≈ 1.618).
		Width:  1000,
		Height: 618,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(6, 7, 15),
		URL:              "/",
		Permissions: map[application.PermissionType]application.Permission{
			application.PermissionCamera:     application.PermissionAllow,
			application.PermissionMicrophone: application.PermissionAllow,
		},
	})

	// overlay window
	overlayWindow := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "Overlay",
		Name:  "overlay",
		// Window sized to the golden ratio (1000 / 618 ≈ 1.618).
		Width:  1000,
		Height: 618,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(6, 7, 15),
		URL:              "/overlay/",
		Hidden:           true,
	})

	mainWindow.OnWindowEvent(events.Common.WindowClosing, func(event *application.WindowEvent) {
		app.Quit()
	})

	// A hidden or minimised window stops the webview rendering, which stops the camera
	// loop. So "hiding" keeps the window mapped and makes it as invisible as the platform
	// allows: off screen (X11) and 1x1 (Wayland, which ignores requested positions).
	var savedX, savedY, savedWidth, savedHeight int
	hidden := false

	hideMainWindow := func() {
		if hidden {
			return
		}

		savedX, savedY = mainWindow.Position()
		savedWidth, savedHeight = mainWindow.Size()
		hidden = true

		// a tiny window still shows its decorations, so drop the frame while hidden
		// GTK4 ignores the default size on a mapped resizable window, a non-resizable
		// one has to follow it
		mainWindow.SetResizable(false)
		mainWindow.SetSize(hiddenWidth, hiddenHeight)
		mainWindow.SetFrameless(true)
		mainWindow.SetPosition(offscreenPosition, offscreenPosition)
	}

	showMainWindow := func() {
		if hidden {
			// resize while still non-resizable, then make it resizable again
			mainWindow.SetSize(savedWidth, savedHeight)
			mainWindow.SetResizable(true)
			mainWindow.SetFrameless(false)
			mainWindow.SetPosition(savedX, savedY)
			hidden = false
		}

		mainWindow.Show()
		mainWindow.Focus()
	}

	mainWindow.OnWindowEvent(events.Common.WindowMinimise, func(event *application.WindowEvent) {
		mainWindow.UnMinimise()
		hideMainWindow()
	})

	// Create system tray
	systray := app.SystemTray.New()
	// systray.SetIcon(icon)
	systray.SetLabel(appName)
	// AttachWindow's default toggle calls Hide(), which stops the webview rendering,
	// so the click has to go through the off screen hide/show instead.
	systray.OnClick(func() {
		if hidden {
			showMainWindow()
			return
		}

		hideMainWindow()
	})

	// Add menu
	menu := app.NewMenu()
	menu.Add("Show").OnClick(func(ctx *application.Context) {
		showMainWindow()
	})
	menu.Add("Quit").OnClick(func(ctx *application.Context) {
		app.Quit()
	})
	systray.SetMenu(menu)

	app.Event.On(NotificationEventSend, func(event *application.CustomEvent) {
		data, ok := event.Data.(struct {
			Title string `json:"title"`
			Body  string `json:"body"`
		})

		if !ok {
			logger.Error("failed to parse notification data", "data", data)
			return
		}

		if err := notifier.SendNotification(notifications.NotificationOptions{
			ID:    "blinker_notification_id",
			Title: data.Title,
			Body:  data.Body,
		}); err != nil {
			logger.Error("failed to send notification", "err", err)
		}
	})

	overlayMode := overlayModeNone

	go func() {
		for {
			select {
			case ev, ok := <-bpEvent:
				if !ok {
					bpEvent = nil
					continue
				}

				bp := ev.data
				switch ev.typ {
				case BreakpointEventStartBreak:
					overlayMode = overlayModeBreak
					app.Event.Emit(BreakEventStart, bp)
					// overlayWindow.SetHTML()
					overlayWindow.SetURL("/break/")
					overlayWindow.Show()
				case BreakpointEventFinishBreak:
					if overlayMode == overlayModeBreak {
						overlayMode = overlayModeNone
						overlayWindow.Hide()
					}
					app.Event.Emit(BreakEventFinish, bp)
				}
			case ev := <-cameraEvent:
				switch ev.typ {
				case CameraEventBlinkReminder:
					// a break is more important than a blink reminder
					if overlayMode != overlayModeNone {
						continue
					}

					overlayMode = overlayModeBlink
					overlayWindow.SetURL("/overlay/")
					overlayWindow.Show()
				case CameraEventBlinkDismiss:
					if overlayMode != overlayModeBlink {
						continue
					}

					overlayMode = overlayModeNone
					overlayWindow.Hide()
				}
			}
		}
	}()

	// Run the application. This blocks until the application has been exited.
	// If an error occurred while running the application, log it and exit.
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
