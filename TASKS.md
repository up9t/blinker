# Tasks

- [x] Camera
  - [x] Blink Detector
  - [x] Camera Selection
  - [x] Screen time (face detect -> send event -> timer run on backend, face undetect -> send event -> timer stop), store data to sqlite, with buffered/timeout/aggregated/debounce.
  - [x] Enter efficiency mode when the user isn't on screen for a few seconds, simple timeout to detect if the user is back on screen.
  - [x] Simple camera indicator (badge or something) (active/paused)
  - [x] Tweak/Config/Settings for the blink parameter such as eye close/open threshold and timeout for blink. This one needs to persist/store the data to the backend, and automatically hydrate/load/sync the settings when relaunch/mount/reload the app.
  - [x] Show Today's Screen Time on CameraWidget.tsx.
  - [x] Show Today's Blink Count on CameraWidget.tsx. (reads today's stored total on mount, so it survives navigation)
  - [x] Each hour of screen time, send a reminder/notification. (hour of today's total, survives restarts by seeding from the stored day total)
  <!-- detection keeps running while "hidden": the window is moved off screen (x11) or shrunk to 1x1 frameless (wayland), since a hidden/minimised window stops rAF -->

- [x] Breakpoint
  - [x] Breakpoint page (breakpoints.astro)
  - [x] Breakpoint Widget/Panel/both to create/modify/delete breakpoints (settings like Break Interval then Break Duration, disable/enable, title/name)
  - [x] Add test for the backend including the breakpoint service.
  - [x] Tweak the existing breakpoint service if necessary.
  - [x] Support multiple breakpoints.
  - [x] Postpone feature on break (button on the overlay window)
  - [x] Queue break duration if overlap instead of replace.
  - [x] Save settings data to the backend and automatically hydrate/sync on launch/relaunch/mount/reload the app. This settings is in its own boundary/domain, don't mess with other settings like blink_settings or general/global settings.
  - [x] Add test to the frontend.
  - [x] Send Notification 40 seconds before break, if the break interval is >= 40 seconds. Store the number as a constant variable for now. With data of title=appname,body="Break is coming! 40s (name-of-the-break-here)"
  - [x] Postponed break can't be postponed twice, if the postponed break duration is finished, postpone feature back to be available, enforce the rule on the backend and hide the postpone button on the frontend if it's a postponed break.
  - [x] Add "message" field in the breakpoint struct, add input field on the frontend.
  - [x] Add 2 selection dropdowns to choose what audio/sound and layout to use based on what available in the assets manager.
  - [ ] (later need library) Play a sound when break ends, based on the selected audio (backend).
  - [ ] (later needs assets manager) Load the selectect ui layout when the break start.
  - [ ] (later needs overlay editor) Customize break UI, through Overlay Editor (see below)

- [ ] Overlay/Break Editor (need Asset Manager)
  - [ ] Show/preview using the base layout + the customized user content as the child component
  - [ ] Select file from assets manager (dropdown options + open in editor button)
  - [ ] Live preview, when there is change in the file in the selected option, automatically mirror the change in the browser.

- [x] Statistics
  - [x] Stored on the backend using sqlite. Buffered/aggregated/debounce/timeout, before sending the count data to the backend.
  - [x] Statistics page (statistics.astro)
  - [x] Show Total Blink Count: today, this week, this month
  - [x] Show Blink Count Graph: Today (hourly), This week (Per day), This month (Per day)
  - [x] Screen time this day, this week (per day), this month (per day).

- [ ] Settings (Global settings)
  - [ ] Editor path and argument

- [x] Tray
  - [x] Active as system tray when hide.
  - [x] Reset next break

- [ ] Assets Manager
  - [x] Add/include builtin assets (audio/layout) from `assets/` directory embedded in the Go binary using go:embed.
  - [x] AssetsService on the backend, and Frontend can retrieve data from the backend, method such as list audio, list layout, etc from the go:embed variable, with data includes asset name, asset type (audio/layout).
  - [ ] Allow user upload/import custom assets such as audio for custom notif sound, html layout for breakpoints ui. 
  - [ ] Assets Type: Audio/Overlay.
  - [ ] Import by Drag & drop.
  - [ ] If the user drop audio files only (mp3, ogg, flac, opus), automatically create a new assets (a directory with a package.json, maybe need to create template/package.json file first) with type audio. If they drop audio files with html files, make it assets type overlay.
  - [ ] (Later) Import by Install asset through a registry.
  - [ ] Uploaded content should be copied somewhere in a directory such as appdir/assets/.
  - [ ] Button to open assets dir in file manager. Frontend -> Backend. Does wails have this capability? I need to know it first.
  - [ ] Asset manifest defined in package.json. And "displayName" would be the data user see in the dropdown option in the breakpoints page.
  - [ ] Scan assets only 1-level directly inside the asset directory.
  - [ ] Enable/Disable assets. Disabled asset won't be shown in the dropdown selection.
  - [ ] Create Test AssetsManager/Service.

- [ ] Testing
  - Write different kinds of tests. Ensuring quality of the code and performance.

- [ ] CI/CD
  - Github action auto build/release.

- [ ] Extension/Plugin
  - Idk how it worked

- [ ] Logo (developer job, DO NOT INTERFERE!)
  - Update logo 

- [ ] Web (developer job, DO NOT INTERFERE!)
  - Make a website or landing page about this project 

- [x] Flatpak
  - [x] Store data persistently

- [ ] Docs
  - Update the showcase images
  - Update README.md

- [ ] UI (developer job, DO NOT INTERFERE!)
  - Design a beautiful/playful UI (lowest priority right now, consider using Shadcn components).