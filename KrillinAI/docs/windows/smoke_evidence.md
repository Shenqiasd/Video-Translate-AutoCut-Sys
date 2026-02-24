# Windows Smoke Evidence (v1)

Use this file as a checklist + place to paste evidence (screenshots / command output).

## Build Info

- Commit:
- Artifact:
- Machine:

## 1. First Launch

Evidence:

- Folder before launch (screenshot):
- After launch directories exist:
  - `data\config\`
  - `data\logs\`
  - `data\output\`
  - `data\cache\`
- Config exists: `data\config\config.toml`

## 2. Diagnose Output

Command:

```powershell
./KrillinAI.exe --diagnose
```

Paste output:

```

```

## 3. Dependency Check UX

Evidence:

- If ffmpeg missing, UI shows missing deps and offers Fix/Install (screenshot):

## 4. Subtitle Task (Happy Path)

Evidence:

- Input URL or local file:
- Start task, task visible, stage/progress visible:
- Output files in `data\output\...` (screenshot):
- UI can open output directory:

## 5. Failure Path

Evidence:

- Offline start task -> failure is actionable:
- Logs exist under `data\logs\`:

## 6. Path Edge Cases

Evidence:

- Install folder contains spaces:
- Install folder contains Chinese characters:
