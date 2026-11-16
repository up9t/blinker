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

	breakpointService := NewBreakpointService(breakpointRepo)
	bpEvent, err := breakpointService.Start(ctx)

	if err != nil {
		logger.Error("Failed to start breakpoint service", "err", err)
		return
	}

	defer breakpointService.Stop()

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
		URL:              "/overlay",
		Hidden:           true,
	})

	mainWindow.OnWindowEvent(events.Common.WindowClosing, func(event *application.WindowEvent) {
		app.Quit()
	})
	mainWindow.OnWindowEvent(events.Common.WindowMinimise, func(event *application.WindowEvent) {
		mainWindow.Hide()
	})

	// Create system tray
	systray := app.SystemTray.New()
	// systray.SetIcon(icon)
	systray.SetLabel(appName)
	systray.AttachWindow(mainWindow)

	// Add menu
	menu := app.NewMenu()
	menu.Add("Show").OnClick(func(ctx *application.Context) {
		// Show main window
		mainWindow.Show()
		mainWindow.Focus()
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

	go func() {
		for ev := range bpEvent {
			if ctx.Err() != nil {
				return
			}

			bp := ev.data
			switch ev.typ {
			case BreakpointEventStartBreak:
				app.Event.Emit(BreakEventStart, bp)
				// overlayWindow.SetHTML()
				overlayWindow.SetURL("/break")
				overlayWindow.Show()
			case BreakpointEventFinishBreak:
				overlayWindow.SetURL("/overlay")
				overlayWindow.Hide()
				app.Event.Emit(BreakEventFinish, bp)
			}
		}
	}()

	// Run the application. This blocks until the application has been exited.
	// If an error occurred while running the application, log it and exit.
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
