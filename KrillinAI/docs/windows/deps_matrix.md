# Dependency Matrix

This document defines the dependencies required by KrillinAI, their categories, detection methods, and installation strategies.

## Categories

- **Must**: Required for core functionality. The application cannot run without these.
- **Should**: Recommended for better experience. Warnings shown if missing.
- **Optional**: Enhances functionality but not required.

## Dependency List

| Dependency | Category | Purpose | Detection | Installation Strategy |
|------------|----------|---------|-----------|---------------------|
| ffmpeg | Must | Audio/video processing | `exec.LookPath("ffmpeg")` | Auto-download to `CacheDir/bin` on Windows; download to `./bin` on Linux/macOS |
| ffprobe | Must | Media metadata extraction | `exec.LookPath("ffprobe")` | Bundled with ffmpeg download |
| yt-dlp | Must | Video download from URLs | `exec.LookPath("yt-dlp")` | Auto-download to `CacheDir/bin` on Windows; download to `./bin` on Linux/macOS |
| edge-tts | Should | Text-to-speech via Microsoft Edge | `exec.LookPath("edge-tts")` | Auto-download to `CacheDir/bin` on all platforms |

## Transcription Providers (Provider-Specific)

| Provider | Dependency | Category | Detection | Installation |
|----------|------------|----------|-----------|--------------|
| fasterwhisper | faster-whisper-xxl | Must (when selected) | File exists in `./bin/faster-whisper/` | Auto-download large zip (~2GB) |
| whisperkit | whisperkit-cli | Must (when selected) | `which whisperkit-cli` | `brew install whisperkit-cli` on macOS |
| whisperx | whisperx + Python venv | Must (when selected) | File exists in `./bin/whisperx/` | Auto-download and setup venv |
| whispercpp | whisper-cli.exe | Must (when selected) | File exists in `bin/whispercpp/` | Auto-download zip |

## Model Files

| Model | Provider | Size | Location | Auto-Download |
|-------|----------|------|----------|---------------|
| large-v2 | fasterwhisper | ~1.5GB | `./models/faster-whisper-large-v2/` | Yes |
| large-v2 | whisperkit | ~1.5GB | `./models/whisperkit/openai_whisper-large-v2/` | Yes |
| large-v2 | whispercpp | ~1.5GB | `./models/whispercpp/ggml-large-v2.bin` | Yes |

## Storage Paths

All dependencies are stored under the resolved `CacheDir`:

- **Windows (portable)**: `<exe_dir>/data/cache/`
- **Windows (installed)**: `%LOCALAPPDATA%/KrillinAI/cache/`
- **Linux/macOS**: `./cache/` (relative to working directory)

Binaries are placed in `CacheDir/bin/` with platform-specific subdirectories as needed.

## Error Classification

| Error Type | User-Facing Message | Action |
|------------|---------------------|--------|
| Network timeout | "Download timed out. Check your internet connection." | Retry with exponential backoff |
| Certificate error | "SSL certificate verification failed. Check system time and proxy settings." | Show proxy configuration help |
| Permission denied | "Cannot write to installation directory. Run as administrator or choose a different location." | Suggest portable mode or admin rights |
| Disk full | "Insufficient disk space. At least 2GB required for transcription models." | Show disk cleanup suggestions |
| Proxy required | "Cannot reach download server. Configure proxy in settings." | Link to proxy configuration |

## Idempotency

All dependency checks are idempotent:
- Re-running checks detects existing installations
- No duplicate downloads if valid binary exists
- Version verification via `--version` flag where supported
