package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"uuid"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"

	"github.com/adrg/xdg"
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

const BreakEventStart = "break:start"

const (
	overlayModeNone = iota
	overlayModeBlink
	overlayModeBreak
)

// far enough away that no monitor can show it
const offscreenPosition = -10000

func init() {
	// Register a custom event whose associated data type is string.
	// This is not required, but the binding generator will pick up registered events
	// and provide a strongly typed JS/TS API for them.
	application.RegisterEvent[string](NotificationEventSend)
	application.RegisterEvent[BreakInfo](BreakEventStart)
}

const AppName = "Blinker"

func getAppDataDir() (string, error) {
	appDataDir := filepath.Join(xdg.DataHome, AppName)

	if err := os.MkdirAll(appDataDir, 0755); err != nil {
		return "", err
	}

	return appDataDir, nil
}

// main function serves as the application's entry point. It initializes the application, creates a window,
// and starts a goroutine that emits a time-based event every second. It subsequently runs the application and
// logs any error that might occur.
func main() {

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()

	notifier := notifications.New()

	appDataDir, err := getAppDataDir()

	if err != nil {
		logger.Error("failed to get app data directory",
			"path", appDataDir,
			"err", err,
		)

		return
	}

	dbPath := filepath.Join(appDataDir, "app.sqlite3")
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
		Name:        AppName,
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
		Linux: application.LinuxOptions{
			ApplicationID: "org.blinker.Blinker",
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
		Frameless:        true,
		StartState:       application.WindowStateFullscreen,
	})

	mainWindow.OnWindowEvent(events.Common.WindowClosing, func(event *application.WindowEvent) {
		app.Quit()
	})

	// A hidden or minimised window stops the webview rendering, which stops the camera
	// loop. So "hiding" keeps the window mapped and makes it as invisible as the platform
	// allows: off screen (X11) and 1x1 (Wayland, which ignores requested positions).
	var savedX, savedY int
	hidden := false

	hideMainWindow := func() {
		if hidden {
			return
		}

		savedX, savedY = mainWindow.Position()
		hidden = true

		// a tiny window still shows its decorations, so drop the frame while hidden
		// GTK4 ignores the default size on a mapped resizable window, a non-resizable
		// one has to follow it
		mainWindow.SetPosition(offscreenPosition, offscreenPosition)
	}

	showMainWindow := func() {
		if hidden {
			// resize while still non-resizable, then make it resizable again
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
	systray.SetLabel(AppName)
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
	menu.Add("Reset next break").OnClick(func(ctx *application.Context) {
		breakpointService.ResetNextBreak()
	})
	menu.Add("Quit").OnClick(func(ctx *application.Context) {
		app.Quit()
	})
	systray.SetMenu(menu)

	sendNotification := func(title, body string) {
		if err := notifier.SendNotification(notifications.NotificationOptions{
			ID:    uuid.New().String(),
			Title: title,
			Body:  body,
		}); err != nil {
			logger.Error("failed to send notification", "err", err)
		}
	}

	overlayMode := overlayModeNone

	go func() {
		for {
			select {
			case ev := <-bpEvent:
				switch ev.typ {
				case BreakpointEventBreakSoon:
					sendNotification(AppName, fmt.Sprintf("Break is coming! %ds (%s)", ev.data.RemainingMs/1000, ev.data.Name))
				case BreakpointEventStartBreak:
					// a break is more important than a blink reminder, it takes the overlay
					overlayMode = overlayModeBreak
					// the break page is already mounted when a queued break starts, so it
					// gets the info from the event and reads ActiveBreak() only on mount
					app.Event.Emit(BreakEventStart, ev.data)
					overlayWindow.SetURL("/break/")
					overlayWindow.Show()
				case BreakpointEventFinishBreak:
					if overlayMode == overlayModeBreak {
						overlayMode = overlayModeNone
						overlayWindow.Hide()
					}
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
				case CameraEventScreenTimeReminder:
					unit := "hours"
					if ev.hours == 1 {
						unit = "hour"
					}

					sendNotification(AppName, fmt.Sprintf("You've been on screen for %d %s today. Take a short break.", ev.hours, unit))
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
