---
name: flutter-test
description: Use this skill when the user wants to run, test, or interact with a Flutter app on a simulator/emulator. Triggers on requests like "start the app", "test the UI", "check what the app looks like", "tap the button", "take a screenshot".
allowed-tools: Bash(fac *), Bash(which fac), Bash(curl *), Read
hooks:
  SessionStart:
    - hooks:
        - type: command
          command: "${CLAUDE_SKILL_DIR}/scripts/setup.sh"
---

# Flutter App Testing via FAC

You have access to FAC (Flutter Agent Connect), a CLI that lets you control Flutter simulators/emulators on the host Mac from this container.

## Setup (run once per work session)

```bash
# Connect to the FAC server on the host
fac connect http://host.docker.internal:8420 --agent $HOSTNAME

# Create simulator sessions
fac session create --platform ios --name ios

# If the app has a backend running in this container, forward it
# fac forward 8080 -e BACKEND_URL

# Start the Flutter app
fac flutter run
```

## Development Loop

1. **Edit code** in the Flutter project
2. **Reload**: `fac flutter hot-reload` (or `fac flutter hot-restart` for bigger changes)
3. **Screenshot**: Read the file returned by `fac device screenshot`
4. **Interact**:
   - `fac device tap --label "Button Text"` — tap by semantics label
   - `fac device type "text"` — type into focused field
   - `fac device swipe --down` — scroll
5. **Screenshot again** to verify

## Command Reference

| Command | Purpose |
|---------|---------|
| `fac flutter run` | Build and start the app |
| `fac flutter hot-reload` | Apply code changes (state preserved) |
| `fac flutter hot-restart` | Full restart (state reset) |
| `fac flutter stop` | Stop the app |
| `fac flutter clean` | Clear build cache |
| `fac flutter pub-get` | Install dependencies |
| `fac device screenshot` | Screenshot (outputs file path) |
| `fac device tap --label "X"` | Tap widget by semantics label |
| `fac device tap <x> <y>` | Tap at coordinates |
| `fac device type "text"` | Type text |
| `fac device swipe --down` | Scroll |
| `fac session use <name>` | Switch iOS/Android session |
| `fac session destroy` | Clean up when done |

## Tips

- **Finding tap targets**: Use a wrong label — the error lists all available labels
- **After pubspec.yaml changes**: `fac flutter pub-get` then `fac flutter hot-restart`
- **Build errors**: `fac flutter clean` then `fac flutter run`
- **Screenshots**: Output is ONLY the file path. Read the file to see the image.
- **Cleanup**: Run `fac session destroy` when done to free simulator resources

## Task

$ARGUMENTS
