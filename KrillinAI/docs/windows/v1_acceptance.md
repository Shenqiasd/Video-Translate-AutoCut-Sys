# Windows Desktop v1 Acceptance Criteria

This document defines the "done" criteria for the Windows desktop (portable-by-default) build.

## Release Mode

- Default mode: portable
- Portable mode root: alongside the executable
  - `data/config/config.toml`
  - `data/logs/`
  - `data/output/`
  - `data/cache/`

## A. Install / First Run

- User can unzip the release archive into an empty folder.
- User can double-click the executable and the app starts.
- On first run, required directories under `data/` are created automatically.
- On first run, a default config is created automatically if missing.
- App shows a clear error dialog if startup fails.

## B. Core Workflow (Subtitle)

- User can input a video URL or select a local file (at least one must work in v1).
- User can configure LLM provider settings (base URL, model, API key if required).
- User can start a subtitle task from the UI.
- UI shows task lifecycle:
  - queued
  - preparing
  - processing
  - finalizing
  - succeeded / failed / canceled
- UI shows progress updates or at least a stage + log stream.
- On success, user can open the output directory from the UI.

## C. Error Handling & Recovery

- Missing dependencies are detected before starting a task.
- For missing required dependencies (e.g. ffmpeg), UI shows:
  - what is missing
  - why it is needed
  - a one-click "Fix/Install" action if supported
- Failures record a human-readable reason.
- User can retry a failed task.
- User can cancel a queued task; running task cancellation is best-effort.

## D. Observability / Diagnose

- UI offers one-click actions:
  - open config file
  - open log directory
  - open output directory
- CLI `--diagnose` prints:
  - app version
  - executable path
  - portable mode
  - effective directories (config/log/output/cache)
  - dependency checks summary
- App writes logs to `data/logs/` in portable mode.

## E. Windows Compatibility Matrix (must pass before release)

- Paths:
  - spaces in path
  - Chinese characters in path
  - long-ish paths (best-effort; document limitations if any)
- Permissions:
  - non-admin user
  - folder is not writable (shows clear error)
- Network:
  - offline start
  - download timeout (shows clear error)

## Non-Goals for v1

- Code signing / SmartScreen reputation improvements.
- Full offline model bundles.
- Perfect cancellation of all underlying subprocesses.
