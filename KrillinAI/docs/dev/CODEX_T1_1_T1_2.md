# Codex Task: T1.1 + T1.2 (Portable-by-default config path)

Repo: `Video-Translate-AutoCut-Sys/KrillinAI`
Branch: `feat/desktop-native-step1-appcore`

## Context

We have introduced `internal/appdirs` to resolve paths for portable mode and Windows app data.
Windows release should default to portable mode: all data stored next to the executable under `data/`.

Currently `config.LoadConfig()` and `config.SaveConfig()` still use legacy relative paths:
- Load: `./config/config.toml`
- Save: `config/config.toml`

This breaks portable-by-default, and also makes `--diagnose` misleading.

## Goals

1. Introduce a single source of truth for runtime paths (at least for config file) that:
   - defaults to portable mode on Windows releases
   - supports explicit override via env var `KRILLINAI_PORTABLE=1/true` (already exists)
2. Update `config.LoadConfig()` and `config.SaveConfig()` to use the resolved config file path.
3. Ensure directories are created automatically when saving.
4. Keep non-Windows behavior backward compatible (still relative `config/config.toml`).

## Requirements

- Do not introduce new third-party dependencies.
- Keep behavior stable for non-desktop/server builds unless they opt into portable.
- Add tests for config path resolution if feasible; at minimum unit test the path provider.

## Implementation Sketch

- Add a small package (or function) to expose resolved paths, e.g.:
  - `internal/runtimepaths` or `internal/appdirs` extension.
- Prefer not to import internal packages into `config` unless it's already acceptable. If it is acceptable, `config` can call `internal/appdirs.Resolve()`.
- If importing `internal/*` into `config` is undesirable, add a `config.SetConfigPathProvider(func() (string, error))` hook.

## Deliverables

- Code changes implementing the above.
- Update `cmd/desktop/flags.go` diagnose output if needed to match real paths.
- `go test ./...` passes.
