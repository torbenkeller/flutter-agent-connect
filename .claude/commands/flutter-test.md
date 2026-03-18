You have access to FAC (Flutter Agent Connect), a CLI tool that lets you control Flutter simulators/emulators remotely from this container.

## Install FAC (if not already installed)

First, check if `fac` is available. If not, install it:
```bash
which fac || (curl -fsSL https://github.com/torbenkeller/flutter-agent-connect/releases/latest/download/fac-linux-arm64 -o /usr/local/bin/fac && chmod +x /usr/local/bin/fac)
```

## Setup (run once per work session)

Before interacting with the Flutter app, you MUST set up the connection and session:

```bash
# 1. Connect to the FAC server running on the host Mac
fac connect http://host.docker.internal:8420 --agent $HOSTNAME

# 2. Create a session (this boots a simulator)
# For iOS:
fac session create --platform ios --name ios
# For Android:
fac session create --platform android --name android

# 3. If the app has a backend, forward its port
fac forward 8080 -e BACKEND_URL

# 4. Start the Flutter app
fac flutter run
```

## Development loop

After setup, use this cycle to develop and verify changes:

1. **Edit code** — make your changes to the Flutter source files
2. **Hot reload** — apply changes: `fac flutter hot-reload`
   - If hot reload fails, use: `fac flutter hot-restart`
3. **Screenshot** — verify the result visually:
   ```bash
   SCREENSHOT=$(fac device screenshot)
   ```
   Then read the screenshot file to see what the app looks like.
4. **Interact** — tap buttons, enter text, navigate:
   ```bash
   fac device tap --label "Login"        # tap by semantics label
   fac device tap --label "Überspringen" # works with any language
   fac device type "user@example.com"    # type into focused field
   fac device swipe --down               # scroll
   ```
5. **Screenshot again** — verify the interaction worked

## Important commands reference

| Command | What it does |
|---------|-------------|
| `fac flutter run` | Build and start the app |
| `fac flutter hot-reload` | Apply code changes (state preserved) |
| `fac flutter hot-restart` | Full restart (state reset) |
| `fac flutter stop` | Stop the app |
| `fac flutter clean` | Clear build cache |
| `fac flutter pub-get` | Install dependencies |
| `fac device screenshot` | Take screenshot (outputs file path only) |
| `fac device tap --label "X"` | Tap widget by semantics label |
| `fac device tap <x> <y>` | Tap at pixel coordinates |
| `fac device type "text"` | Type text into focused field |
| `fac device swipe --down` | Swipe/scroll |
| `fac session use <name>` | Switch between iOS/Android sessions |
| `fac session destroy` | Clean up session when done |

## Tips

- **Finding tap targets**: If you don't know the label, try tapping with a wrong label — the error message lists all available labels.
- **After pubspec.yaml changes**: Run `fac flutter pub-get` then `fac flutter hot-restart`
- **Build errors**: Try `fac flutter clean` then `fac flutter run`
- **Screenshots**: The command outputs ONLY the file path. Read the file to see the image.
- **Multiple platforms**: Create multiple sessions, switch with `fac session use ios` / `fac session use android`

## Cleanup

When you're done testing, destroy the session to free up simulator resources:
```bash
fac session destroy
```

## Current task

$ARGUMENTS
