# Was der Killcounter sendet / What the kill counter sends

Diese Seite listet vollstaendig auf, welche Daten den Rechner verlassen, wann
das passiert, und wo im Quellcode das jeweils steht. Jede Angabe ist mit einem
Link auf die konkrete Codezeile belegt, damit sie nachpruefbar ist statt
geglaubt werden zu muessen.

Stand: Version 1.0.11.

This page lists in full what data leaves your machine, when it happens, and
where in the source each claim can be checked. Every statement links to the
exact line of code, so it can be verified rather than believed. English version
below the German one.

---

## Deutsch

### Wann ueberhaupt etwas passiert

Ein Screenshot wird **ausschliesslich** dann erstellt, wenn beide Bedingungen
gleichzeitig erfuellt sind:

1. Du drueckst selbst die Aufnahmetaste
2. Escape from Tarkov ist das aktive Fenster im Vordergrund

Ist Tarkov nicht im Vordergrund, passiert nichts — die Funktion bricht sofort
ab. Das ist der Check in
[hotkey.go, Zeile 354](https://github.com/miwidot/tarkov-killscreen/blob/v1.0.11/hotkey.go#L354).

Es gibt keine automatische Aufnahme, keine zeitgesteuerte Aufnahme und keine
Aufnahme im Hintergrund. Ohne Tastendruck passiert nichts.

### Was aufgenommen wird

Der komplette Bildschirm des Monitors, auf dem Tarkov laeuft:
[hotkey.go, Zeile 385](https://github.com/miwidot/tarkov-killscreen/blob/v1.0.11/hotkey.go#L385).

Das ist bewusst offen gesagt: es ist ein Vollbild-Screenshot dieses einen
Monitors, kein Ausschnitt nur des Kill-Bereichs. Weil Tarkov im Vordergrund
sein muss, zeigt dieser Bildschirm Tarkov. Wenn du auf demselben Monitor ein
Overlay eingeblendet hast, ist es mit auf dem Bild. Andere Monitore werden nicht
erfasst.

### Was gesendet wird

Beim Hochladen an die OCR-Schnittstelle
([upload.go](https://github.com/miwidot/tarkov-killscreen/blob/v1.0.11/upload.go#L260-L300)):

| Feld | Inhalt |
|---|---|
| Bild | Der Screenshot als JPEG |
| `mode` | Betriebsmodus aus der Konfiguration |
| `client_version` | Versionsnummer der App, z. B. `1.0.11` |
| `Authorization` | Dein API-Token, damit der Server weiss, wem die Kills gehoeren |
| `X-Device-ID` | Eine Zufalls-ID, siehe unten |

Nach der Auswertung werden die erkannten Kill-Daten gespeichert
([upload.go, SaveKills](https://github.com/miwidot/tarkov-killscreen/blob/v1.0.11/upload.go#L455-L500)):
Map, Zeitpunkt, Kill-Zahlen, Waffen, die einzelnen Kill-Eintraege, die
Bild-Hashes und die Client-Version.

**Das ist die vollstaendige Liste.** Mehr wird nicht uebertragen.

### Zur Device-ID

Die `X-Device-ID` ist eine zufaellige UUID v4, die beim ersten Start mit
`crypto/rand` erzeugt und in `%APPDATA%\TarkovKillcounter\device.id` abgelegt
wird:
[device.go, Zeilen 20-53](https://github.com/miwidot/tarkov-killscreen/blob/v1.0.11/device.go#L20-L53).

Sie ist **nicht** aus deiner Hardware abgeleitet. Es werden keine Seriennummern,
keine MAC-Adresse, keine Windows-Installations-ID und kein Benutzername
ausgelesen. Es ist eine Zufallszahl. Ihr einziger Zweck ist, dein Token an ein
Geraet zu binden, damit ein gestohlenes Token nicht anderswo benutzt werden
kann. Wenn du die Datei loeschst, bekommst du eine neue.

### Wohin gesendet wird

[debug_release.go, Zeilen 15 und 19](https://github.com/miwidot/tarkov-killscreen/blob/v1.0.11/debug_release.go#L15-L19):

- `kc.tarkov-stammtisch.de` — die eigentliche Schnittstelle
- `kc.notgood.cc` — Ausweich-Adresse, falls die erste nicht erreichbar ist

Die zweite Adresse sieht auf den ersten Blick fremd aus, deshalb hier klar
gesagt: das ist ebenfalls ein Server des Projekts. Er wurde in Version 1.0.9
eingefuehrt, weil kaputte ISP-DNS-Konfigurationen die Hauptadresse zeitweise
unerreichbar gemacht haben. Er wird nur angesprochen, wenn die Hauptadresse
fehlschlaegt, und bekommt exakt dieselben Daten wie sie.

Zusaetzlich wird `api.github.com` abgefragt, um zu pruefen ob eine neuere
Version vorliegt. Diese Anfrage enthaelt keinerlei Nutzerdaten, kein Token und
keine Device-ID — sie fragt nur die Liste der Releases ab.

Andere Ziele gibt es nicht. Das laesst sich mit jeder Firewall oder mit
Wireshark nachpruefen.

### Was die App auf dem Rechner liest, aber nicht sendet

Ehrlichkeitshalber gehoert das hierher, weil es beim Lesen des Codes auffaellt:

Die App fragt Windows nach der Liste der laufenden Prozesse
([windows.go, Zeile 125](https://github.com/miwidot/tarkov-killscreen/blob/v1.0.11/windows.go#L125)).
Sie prueft darin genau drei Dinge: laeuft `EscapeFromTarkov.exe`, laeuft Single
Player Tarkov (dann wird blockiert, weil SPT-Kills keine Online-Kills sind), und
laeuft ein Bildbetrachter (dann wird blockiert, damit kein bereits
hochgeladenes Bild erneut abfotografiert wird).

Diese Liste wird **nicht** uebertragen. Sie verlaesst den Rechner nicht. Sie
wird nur lokal ausgewertet, um zu entscheiden ob eine Aufnahme erlaubt ist. Im
Upload oben ist kein Feld dafuer vorgesehen.

### Was die App nicht tut

- Die Zwischenablage wird nicht gelesen. Es gibt keinen einzigen
  Clipboard-Aufruf im Code. Eine sehr fruehe Version nutzte die Zwischenablage,
  die aktuelle nicht mehr.
- Keine Tastenprotokollierung. Es wird eine einzige Taste global registriert,
  die Aufnahmetaste. Andere Tastendruecke werden weder gelesen noch gespeichert.
- Kein Zugriff auf Spielespeicher, keine DLL-Injection, keine Veraenderung von
  Spieldateien, keine automatisierten Eingaben.
- Keine Telemetrie. Ausser dem oben Aufgelisteten wird nichts erhoben.
- Keine Dateien vom Rechner werden gelesen oder hochgeladen, ausser der eigenen
  Konfiguration und der Device-ID.

### Wie du das selbst pruefst

1. **Quellcode lesen.** Dieses Repository ist das Programm. Die relevanten
   Dateien sind `upload.go` (was gesendet wird), `hotkey.go` (wann aufgenommen
   wird) und `device.go` (die Device-ID).
2. **Netzwerkverkehr mitschneiden.** Wireshark oder eine Firewall zeigen jede
   Verbindung. Es duerfen nur die oben genannten Adressen auftauchen.
3. **Signatur pruefen.** Rechtsklick auf `screenshoter.exe`, Eigenschaften,
   Digitale Signaturen. Ausgestellt auf Martin Wilke, von Certum.
4. **Selbst bauen.** Wenn du der fertigen Exe nicht traust, bau sie aus dem
   Quellcode. Siehe "Building from Source" im README.

Zum Thema Virenscanner-Warnung siehe den Abschnitt "Antivirus False Positives"
im [README](README.md).

---

## English

### When anything happens at all

A screenshot is taken **only** when both conditions are true at the same time:

1. You press the capture key yourself
2. Escape from Tarkov is the active foreground window

If Tarkov is not in the foreground, nothing happens and the function returns
immediately — see
[hotkey.go line 354](https://github.com/miwidot/tarkov-killscreen/blob/v1.0.11/hotkey.go#L354).

There is no automatic capture, no timed capture and no background capture.
Without a key press, nothing happens.

### What is captured

The full screen of the monitor Tarkov runs on:
[hotkey.go line 385](https://github.com/miwidot/tarkov-killscreen/blob/v1.0.11/hotkey.go#L385).

Stated plainly: this is a full-screen shot of that one monitor, not a crop of
the kill area only. Because Tarkov has to be in the foreground, that screen
shows Tarkov. If you have an overlay on the same monitor, it is in the image.
Other monitors are not captured.

### What is sent

On upload to the OCR endpoint
([upload.go](https://github.com/miwidot/tarkov-killscreen/blob/v1.0.11/upload.go#L260-L300)):

| Field | Content |
|---|---|
| image | The screenshot as JPEG |
| `mode` | Operating mode from the configuration |
| `client_version` | App version, e.g. `1.0.11` |
| `Authorization` | Your API token, so the server knows whose kills these are |
| `X-Device-ID` | A random ID, see below |

After analysis the detected kill data is saved
([upload.go, SaveKills](https://github.com/miwidot/tarkov-killscreen/blob/v1.0.11/upload.go#L455-L500)):
map, timestamp, kill counts, weapons, the individual kill entries, image hashes
and the client version.

**That is the complete list.** Nothing else is transmitted.

### About the device ID

`X-Device-ID` is a random UUID v4, generated on first start with `crypto/rand`
and stored in `%APPDATA%\TarkovKillcounter\device.id`:
[device.go lines 20-53](https://github.com/miwidot/tarkov-killscreen/blob/v1.0.11/device.go#L20-L53).

It is **not** derived from your hardware. No serial numbers, no MAC address, no
Windows installation ID and no user name are read. It is a random number. Its
only purpose is binding your token to one device so a stolen token cannot be
used elsewhere. Delete the file and you get a new one.

### Where it is sent

[debug_release.go lines 15 and 19](https://github.com/miwidot/tarkov-killscreen/blob/v1.0.11/debug_release.go#L15-L19):

- `kc.tarkov-stammtisch.de` — the actual endpoint
- `kc.notgood.cc` — fallback address if the first one is unreachable

The second address looks unfamiliar at first glance, so to be clear: it is also
a server belonging to this project. It was introduced in version 1.0.9 because
broken ISP DNS configurations made the main address temporarily unreachable. It
is contacted only when the primary fails, and receives exactly the same data.

Additionally `api.github.com` is queried to check for a newer version. That
request contains no user data, no token and no device ID — it only asks for the
list of releases.

There are no other destinations. Any firewall or Wireshark will confirm this.

### What the app reads locally but does not send

This belongs here for the sake of honesty, because it stands out when reading
the code:

The app asks Windows for the list of running processes
([windows.go line 125](https://github.com/miwidot/tarkov-killscreen/blob/v1.0.11/windows.go#L125)).
It checks exactly three things in it: is `EscapeFromTarkov.exe` running, is
Single Player Tarkov running (captures are blocked then, since SPT kills are not
online kills), and is an image viewer running (blocked too, so an already
uploaded image cannot be re-captured).

That list is **not** transmitted. It never leaves the machine. It is only
evaluated locally to decide whether a capture is allowed. The upload above has
no field for it.

### What the app does not do

- The clipboard is not read. There is not a single clipboard call in the code.
  A very early version used the clipboard, the current one does not.
- No keylogging. Exactly one key is registered globally, the capture key. Other
  key presses are neither read nor stored.
- No game memory access, no DLL injection, no modification of game files, no
  automated input.
- No telemetry. Nothing is collected beyond what is listed above.
- No files are read or uploaded from your machine other than its own
  configuration and the device ID.

### How to verify this yourself

1. **Read the source.** This repository is the program. The relevant files are
   `upload.go` (what is sent), `hotkey.go` (when a capture happens) and
   `device.go` (the device ID).
2. **Capture the network traffic.** Wireshark or a firewall shows every
   connection. Only the addresses listed above may appear.
3. **Check the signature.** Right-click `screenshoter.exe`, Properties, Digital
   Signatures. Issued to Martin Wilke by Certum.
4. **Build it yourself.** If you do not trust the prebuilt exe, build from
   source. See "Building from Source" in the README.

On the antivirus warning, see the "Antivirus False Positives" section in the
[README](README.md).
