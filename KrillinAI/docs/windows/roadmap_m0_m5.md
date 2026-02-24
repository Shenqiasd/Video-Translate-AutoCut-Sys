# Windows M0-M5 Roadmap (towards M5)

This checklist turns milestones into verifiable tasks with:
- Acceptance criteria (what to verify)
- Deliverables (file paths / artifacts)
- Dependencies (what must be done first)

Evidence rule: a task is only "done" when its deliverable exists AND the acceptance criteria can be checked.

---

## M0: Acceptance + Baseline

### T0.1 Define Windows v1 acceptance checklist

- Acceptance criteria:
  - Document explicitly defines "v1 done" for Windows desktop build.
  - Includes portable default layout and core workflow requirements.
- Deliverables:
  - `docs/windows/v1_acceptance.md`
- Dependencies:
  - None

### T0.2 Create repeatable Windows smoke test steps

- Acceptance criteria:
  - Steps can be followed by a Windows user on a clean machine.
  - Covers first launch, diagnose output, deps UX, happy path, failure path, path edge cases.
- Deliverables:
  - `docs/windows/smoke_test.md`
- Dependencies:
  - T0.1

---

## M1: Unified runtime paths (Windows + Portable)

Goal: all runtime directories are resolved through a single provider and all callers use it.

### T1.1 Implement global runtime directory provider (`internal/appdirs`)

- Acceptance criteria:
  - `internal/appdirs` provides a single entry to resolve:
    - `ConfigFile`
    - `LogDir`
    - `OutputDir`
    - `CacheDir`
  - Windows defaults to portable layout; env override supported.
  - Unit tests cover portable + non-windows behavior.
- Deliverables:
  - `internal/appdirs/appdirs.go`
  - `internal/appdirs/runtime_paths.go`
  - `internal/appdirs/*_test.go`
- Dependencies:
  - None

### T1.2 Refactor config load/save to use `ConfigFile`

- Acceptance criteria:
  - `config.LoadConfig` reads from `appdirs.Paths.ConfigFile`.
  - `config.SaveConfig` writes to `appdirs.Paths.ConfigFile`.
  - Parent directories are created automatically when saving.
  - Works in portable mode: creates `data/config/config.toml`.
- Deliverables:
  - Updated code in `config/` (exact files depend on current structure)
  - Tests covering:
    - missing config file => default generated
    - save creates directories
- Dependencies:
  - T1.1

### T1.3 Refactor logging to write to `LogDir` (keep console policy)

- Acceptance criteria:
  - In portable mode logs are written under `data/logs/`.
  - Console output remains available (same as current behavior).
  - Log directory created automatically.
- Deliverables:
  - Updated code in `log/` and any app init code
  - Test(s) verifying log dir path and writability checks (if feasible)
- Dependencies:
  - T1.1

### T1.4 Refactor tasks/output/cache/storage/download/ffmpeg path usage

- Acceptance criteria:
  - Any code that writes:
    - temporary files
    - downloaded deps
    - task outputs
    - caches
    uses `appdirs` resolved directories.
  - No hardcoded relative directories for windows portable mode.
- Deliverables:
  - Changes across:
    - `internal/service/`
    - `internal/storage/`
    - `internal/deps/`
    - ffmpeg invocation code paths
- Dependencies:
  - T1.1

### T1.5 Enhance `--diagnose` output

- Acceptance criteria:
  - `--diagnose` prints:
    - effective paths (config/log/output/cache)
    - writable checks
    - example file create results (for each dir)
  - Output is copy-paste friendly for remote debugging.
- Deliverables:
  - Updated diagnose implementation (likely under `cmd/*` or `internal/*`)
  - Optional doc snippet for expected output
- Dependencies:
  - T1.1, T1.2, T1.3

### T1.6 Add tests (pure Go) for windows/portable path resolution + key dirs create

- Acceptance criteria:
  - Unit tests cover portable and non-portable resolution.
  - Integration-ish tests ensure required dirs can be created under a temp root.
- Deliverables:
  - New/updated tests under `internal/appdirs/` and any path-consuming packages
- Dependencies:
  - T1.1-T1.5

---

## M2: Dependency self-check + auto-prepare (Windows first)

### T2.1 Define dependency list and levels

- Acceptance criteria:
  - Dependencies are categorized:
    - Must (e.g. ffmpeg)
    - Should (fonts)
    - Optional (advanced models)
  - Each dependency has:
    - purpose
    - detection method
    - installation strategy (auto/manual)
- Deliverables:
  - New doc: `docs/windows/deps_matrix.md`
  - Or code-level list + comments in `internal/deps/`
- Dependencies:
  - T1.1 (paths for installs)

### T2.2 Implement TODOs in `internal/deps/checker.go` (download/unzip/verify)

- Acceptance criteria:
  - At least ffmpeg is:
    - downloaded
    - extracted
    - verified (checksum or version probe)
    - stored under `CacheDir`
  - Re-running check is idempotent.
- Deliverables:
  - `internal/deps/checker.go` (+ any helpers)
  - Tests for downloader/extractor (can be mocked)
- Dependencies:
  - T1.1

### T2.3 Improve error classification and UX copy (offline/proxy/etc.)

- Acceptance criteria:
  - Errors are mapped to user-facing actionable messages.
  - Covers offline/timeout/cert/proxy/permission.
- Deliverables:
  - Updated code (deps checker + UI)
  - `docs/windows/troubleshooting_deps.md` (optional but recommended)
- Dependencies:
  - T2.2

### T2.4 UI runs deps check on startup with one-click fix + visible progress

- Acceptance criteria:
  - On startup, deps check runs.
  - Missing Must deps triggers dialog/panel.
  - Fix action shows progress (download/extract).
- Deliverables:
  - UI changes under `cmd/desktop` and/or `internal/desktop`
- Dependencies:
  - T2.2, T2.3

### T2.5 Deps check results exportable into diagnose output

- Acceptance criteria:
  - `--diagnose` includes deps summary (installed/missing/versions/path).
- Deliverables:
  - Diagnose code + deps checker export
- Dependencies:
  - T1.5, T2.2

---

## M3: Job system (UI controllable / observable / cancelable)

### T3.1 Adapt `internal/taskrunner` to `internal/appcore.Runner`

- Acceptance criteria:
  - Single Runner interface drives Job lifecycle and event stream.
  - Existing taskrunner is wired without losing functionality.
- Deliverables:
  - Code changes in `internal/taskrunner/` and `internal/appcore/`
  - Tests for runner event stream
- Dependencies:
  - M1 stable enough for output paths

### T3.2 Job <-> Task mapping with persistent records and consistent state machine

- Acceptance criteria:
  - Creating a job persists a record.
  - States: Queued/Running/Succeeded/Failed/Canceled.
  - UI state matches backend state.
- Deliverables:
  - Code changes in storage/db layer
  - Migration if needed
- Dependencies:
  - T3.1

### T3.3 UI "Task Center" page

- Acceptance criteria:
  - Task list + task detail view exist.
  - Detail shows progress/logs/errors/artifacts.
- Deliverables:
  - UI components
- Dependencies:
  - T3.1, T3.2

### T3.4 Cancel support

- Acceptance criteria:
  - Can cancel queued jobs.
  - Best-effort cancel running jobs:
    - stop downloads
    - kill subprocess
    - mark canceled
- Deliverables:
  - Runner/taskrunner cancel APIs
  - UI cancel button
- Dependencies:
  - T3.1

### T3.5 One-click open dirs/files (platform aware)

- Acceptance criteria:
  - Windows uses `explorer`.
  - macOS uses `open`.
  - Linux uses `xdg-open`.
  - UI offers open config/log/output.
- Deliverables:
  - New helper utility + UI actions
- Dependencies:
  - T1.1

---

## M4: Windows packaging / release loop

### T4.1 Define build tags strategy (`windows` vs `desktop`)

- Acceptance criteria:
  - Local build and goreleaser build behave consistently.
  - Tags are documented.
- Deliverables:
  - `docs/windows/build_tags.md`
  - Updates to build scripts if needed
- Dependencies:
  - M3 baseline

### T4.2 Add goreleaser artifact `windows_amd64_desktop` (zip)

- Acceptance criteria:
  - goreleaser produces zip containing exe + required assets.
  - Layout matches portable expectations.
- Deliverables:
  - `.goreleaser.yaml` changes
- Dependencies:
  - T4.1

### T4.3 Portable default strategy in release artifacts

- Acceptance criteria:
  - Windows release defaults to portable.
  - Either a single package or separate desktop_portable package is produced.
- Deliverables:
  - goreleaser config + release notes snippet
- Dependencies:
  - T4.2

### T4.4 Update README and Chinese docs for Windows install/FAQ/AV warnings

- Acceptance criteria:
  - Docs cover install steps, common issues, antivirus notes.
- Deliverables:
  - `README.md` and/or `docs/windows/*.md`
- Dependencies:
  - T4.2

---

## M5: Stability & UX hardening (special topics)

### T5.1 Path edge cases: spaces/Chinese/long paths

- Acceptance criteria:
  - CLI + UI flows work with spaces/Chinese paths.
  - Long path limitations are documented if unavoidable.
- Deliverables:
  - Tests + doc: `docs/windows/path_edge_cases.md`
- Dependencies:
  - M1

### T5.2 Permissions: no admin / controlled folder access

- Acceptance criteria:
  - If target dir not writable:
    - fallback path used OR
    - clear error shown with instructions
- Deliverables:
  - Code changes + doc snippet
- Dependencies:
  - M1

### T5.3 Network: retry/timeout/cert messages; resume if possible

- Acceptance criteria:
  - Downloads have timeouts and retries.
  - Resume support implemented if feasible; otherwise documented.
  - User-facing errors are actionable.
- Deliverables:
  - downloader improvements + docs
- Dependencies:
  - M2

### T5.4 Crash + diagnostics: panic capture, one-click copy diag info

- Acceptance criteria:
  - Panic is captured and written to crash log.
  - UI offers "Copy diagnostics" including version/paths/deps/recent errors.
- Deliverables:
  - crash logger + UI copy action
- Dependencies:
  - M1, M2

### T5.5 Regression matrix checklist before each release

- Acceptance criteria:
  - Checklist is actionable and references concrete steps.
  - Covers path/perm/network/deps/task/cancel.
- Deliverables:
  - `docs/windows/regression_matrix.md`
- Dependencies:
  - M0-M5
