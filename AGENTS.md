# Blinker

Well-being desktop app that reminds the user to blink and take breaks.

Some might be useful files to read:
- `README.md`
- `DESIGN.md` Frontend UI design guideline.
- `TASKS.md` the tasks the developer is currently working on.
- `ARCHITECTURE.md` for a simple architecture / how it worked  overview of this project.

## Commands

- `wails3 -help` to list all available wails subcommands.
- `wails3 dev` to run the app in development mode (watch for changes)
- `wails3 build` build the app
- `wails3 generate bindings` generate registered services type from Go to Typescript inside `./frontend/bindings`

To run `npm` command you must first enter the `frontend` directory. `cd frontend`.

- `npm run dev -- --port 8080` run astro server with port 8080 .
- `npm run format` format the `frontend/src/` directory using oxfmt.
- `npm run build` build the the frontend and result in `frontend/dist` directory.