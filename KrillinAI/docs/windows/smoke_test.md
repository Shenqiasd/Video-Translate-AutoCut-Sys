# Windows Desktop Smoke Test (Portable Default)

Goal: a repeatable checklist that validates the release artifact is usable by a normal Windows user.

## Pre-conditions

- Use a clean Windows machine/user profile if possible.
- Unzip the release into an empty folder, for example:
  - `C:\Users\Alice\Desktop\KrillinAI Test\`

## 1. First Launch

1) Double click the executable.
2) Confirm the UI window shows up within 30s.
3) Confirm the following directories are created next to the exe:
   - `data\config\`
   - `data\logs\`
   - `data\output\`
   - `data\cache\`
4) Confirm config file exists:
   - `data\config\config.toml`

## 2. Diagnose Output

Run in a terminal (PowerShell) from the folder:

- `./KrillinAI.exe --diagnose`

Confirm it prints:
- executable path
- portable mode = true
- config/log/output/cache dirs
- dependency checks summary

## 3. Dependency Check UX

1) Start the app.
2) Navigate to the dependency section (or start a task to trigger it).
3) If ffmpeg is missing, confirm:
   - a clear message shows
   - there is a one-click fix/install action (if implemented)

## 4. Subtitle Task (Happy Path)

1) Configure LLM provider settings.
2) Input a known-working URL (or a small local file).
3) Start the subtitle task.
4) Confirm:
   - task is visible in a task list/center
   - stage/progress updates appear
5) On success:
   - output files appear under `data\output\...`
   - UI can open output directory

## 5. Failure Path

1) Disconnect network.
2) Start a task that needs download.
3) Confirm:
   - task fails gracefully
   - error is actionable (mentions network)
   - logs contain relevant info under `data\logs\`

## 6. Path Edge Cases

Repeat 1-4 with install folder names containing:
- spaces
- Chinese characters

Optionally test longer paths.

## 7. Uninstall / Cleanup

- Delete the folder. No data should be written outside of it in portable mode.
