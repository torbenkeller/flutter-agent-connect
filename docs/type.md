# fac type

Gibt Text in das aktuell fokussierte Textfeld auf dem Simulator ein.

## Usage

```bash
fac type <text> [flags]
```

## Arguments

| Argument | Beschreibung |
|----------|--------------|
| `text` | Der einzugebende Text |

## Flags

| Flag | Typ | Default | Beschreibung |
|------|-----|---------|--------------|
| `--clear` | bool | `false` | Feld vorher leeren (Select All + Delete) |
| `--enter` | bool | `false` | Nach der Eingabe Enter/Return drücken |
| `--session` | string | aktive Session | Session-ID |

## Was der Command tut

1. Optional: Feld leeren (Cmd+A, dann Delete)
2. Text zeichenweise eingeben
3. Optional: Enter drücken

## API-Mapping

```
POST /sessions/{id}/type
Content-Type: application/json
Authorization: Bearer <token>
```

**Request Body:**
```json
{
  "text": "user@example.com",
  "clear": true,
  "enter": false
}
```

**Response:**
```json
{
  "success": true,
  "text_entered": "user@example.com"
}
```

## Technischer Kontext

### iOS Simulator Text-Eingabe

**Via simctl (bevorzugt, wenn verfügbar):**
```bash
xcrun simctl io <udid> sendkey <key>
```
Unterstützt einzelne Tasten. Für längeren Text: Zeichen einzeln senden.

**Via Pasteboard + Paste (schneller für langen Text):**
```bash
# Text ins Simulator-Pasteboard kopieren
xcrun simctl pbcopy <udid> <<< "user@example.com"
# Dann Cmd+V via AppleScript simulieren
```

**Via AppleScript:**
```applescript
tell application "System Events"
    keystroke "user@example.com"
end tell
```

### Android (Phase 3)

```bash
adb shell input text "user@example.com"
```

Achtung: `adb shell input text` hat Probleme mit Sonderzeichen und Leerzeichen. Workaround: Base64-Encoding oder ADB IME.

### Sonderzeichen

Bestimmte Zeichen brauchen Sonderbehandlung:
- `@`: Shift+2 oder direkte Keycode-Eingabe
- Leerzeichen: OK via keystroke
- Emojis: Schwierig, am besten über Pasteboard

## CLI Output

```bash
$ fac type "user@example.com"
Typed "user@example.com"

$ fac type "password123" --enter
Typed "password123" + Enter

$ fac type "new value" --clear
Cleared field, typed "new value"
```

## Typischer Workflow

```bash
# 1. Auf Textfeld tappen
fac tap --label "Email"

# 2. Text eingeben
fac type "user@example.com"

# 3. Nächstes Feld
fac tap --label "Password"

# 4. Passwort eingeben + absenden
fac type "secret123" --enter
```

## Error Cases

| Fehler | Verhalten |
|--------|-----------|
| Kein Textfeld fokussiert | Warnung (Text wird trotzdem gesendet, geht aber "ins Leere") |
| Sonderzeichen nicht unterstützt | Fallback auf Pasteboard-Methode |
| App/Simulator läuft nicht | Error |
