# Was der Killcounter sendet / What the kill counter sends

Diese Seite listet vollstaendig auf, welche Daten den Rechner verlassen, wann
das passiert, und wo im Quellcode das jeweils steht. Jede Angabe ist mit einem
Link auf die konkrete Codezeile belegt, damit sie nachpruefbar ist statt
geglaubt werden zu muessen.

Stand: Version 1.0.13.

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
[hotkey.go, Zeile 419](https://github.com/miwidot/tarkov-killscreen/blob/v1.0.13/hotkey.go#L419).

Es gibt keine automatische Aufnahme, keine zeitgesteuerte Aufnahme und keine
Aufnahme im Hintergrund. Ohne Tastendruck passiert nichts.

### Was aufgenommen wird

Der komplette Bildschirm des Monitors, auf dem Tarkov laeuft:
[hotkey.go, Zeile 450](https://github.com/miwidot/tarkov-killscreen/blob/v1.0.13/hotkey.go#L450).

Das ist bewusst offen gesagt: es ist ein Vollbild-Screenshot dieses einen
Monitors, kein Ausschnitt nur des Kill-Bereichs. Weil Tarkov im Vordergrund
sein muss, zeigt dieser Bildschirm Tarkov. Wenn du auf demselben Monitor ein
Overlay eingeblendet hast, ist es mit auf dem Bild. Andere Monitore werden nicht
erfasst.

### Was gesendet wird

Beim Hochladen an die OCR-Schnittstelle
([upload.go](https://github.com/miwidot/tarkov-killscreen/blob/v1.0.13/upload.go#L337-L373)):

| Feld | Inhalt |
|---|---|
| Bild | Der Screenshot als JPEG |
| `mode` | Betriebsmodus aus der Konfiguration |
| `client_version` | Versionsnummer der App, z. B. `1.0.13` |
| `Authorization` | Dein API-Token, damit der Server weiss, wem die Kills gehoeren |
| `X-Device-ID` | Eine Zufalls-ID, siehe unten |

Nach der Auswertung werden die erkannten Kill-Daten gespeichert
([upload.go, SaveKills](https://github.com/miwidot/tarkov-killscreen/blob/v1.0.13/upload.go#L532-L575)):
Map, Zeitpunkt, Kill-Zahlen, Waffen, die einzelnen Kill-Eintraege, die
Bild-Hashes, die Client-Version und — nur wenn du eines ausgewaehlt hast — die
ID des Kill-Events.

**Das ist die vollstaendige Liste.** Mehr wird nicht uebertragen.

### Wiederholung bei Verbindungsproblemen

Schlaegt ein Upload fehl, weil die Verbindung abbricht oder der Server gerade
nicht antwortet, versucht die App es bis zu dreimal erneut, nach 5, 15 und 30
Sekunden
([app.go, Zeilen 409-430](https://github.com/miwidot/tarkov-killscreen/blob/v1.0.13/app.go#L409-L430)).
Dabei werden exakt dieselben Daten noch einmal gesendet, nichts Zusaetzliches.
Die Screenshots liegen waehrenddessen nur im Arbeitsspeicher und werden nicht
auf die Festplatte geschrieben. Nach dem letzten Versuch werden sie verworfen.

### Zur Device-ID

Die `X-Device-ID` ist eine zufaellige UUID v4, die beim ersten Start mit
`crypto/rand` erzeugt und in `%APPDATA%\TarkovKillcounter\device.id` abgelegt
wird:
[device.go, Zeilen 20-53](https://github.com/miwidot/tarkov-killscreen/blob/v1.0.13/device.go#L20-L53).

Sie ist **nicht** aus deiner Hardware abgeleitet. Es werden keine Seriennummern,
keine MAC-Adresse, keine Windows-Installations-ID und kein Benutzername
ausgelesen. Es ist eine Zufallszahl. Ihr einziger Zweck ist, dein Token an ein
Geraet zu binden, damit ein gestohlenes Token nicht anderswo benutzt werden
kann. Wenn du die Datei loeschst, bekommst du eine neue.

### Wohin gesendet wird

[debug_release.go, Zeilen 15 und 19](https://github.com/miwidot/tarkov-killscreen/blob/v1.0.13/debug_release.go#L15-L19):

- `kc.tarkov-stammtisch.de` — die eigentliche Schnittstelle
- `kc.notgood.cc` — Ausweich-Adresse, falls die erste nicht erreichbar ist

Die zweite Adresse sieht auf den ersten Blick fremd aus, deshalb hier klar
gesagt: das ist ebenfalls ein Server des Projekts. Er wurde in Version 1.0.9
eingefuehrt, weil kaputte ISP-DNS-Konfigurationen die Hauptadresse zeitweise
unerreichbar gemacht haben. Er wird nur angesprochen, wenn die Hauptadresse
fehlschlaegt, und bekommt exakt dieselben Daten wie sie.

Zusaetzlich gibt es drei Verbindungen, die keine Nutzerdaten enthalten — kein
Token, keine Device-ID, kein Bild:

- `api.github.com` — beim Start und danach alle 30 Minuten wird die Liste der
  Releases abgefragt, um zu pruefen ob eine neuere Version vorliegt
  ([version.go, Zeile 64](https://github.com/miwidot/tarkov-killscreen/blob/v1.0.13/version.go#L64)).
- `github.com` und GitHubs Download-Server (`githubusercontent.com`) — nur
  wenn du ein Update bestaetigst, wird von dort die neue `screenshoter.exe`
  heruntergeladen
  ([selfupdate.go, Zeile 168](https://github.com/miwidot/tarkov-killscreen/blob/v1.0.13/selfupdate.go#L168)).
- `1.1.1.1` und `1.0.0.1` (Cloudflare DNS) — nur wenn die Namensaufloesung
  deines Systems fehlschlaegt, wird der Hostname des Servers stattdessen dort
  nachgeschlagen
  ([dns_fallback.go, Zeilen 27-38](https://github.com/miwidot/tarkov-killscreen/blob/v1.0.13/dns_fallback.go#L27-L38)).
  Cloudflare sieht dabei den abgefragten Hostnamen und deine IP-Adresse, sonst
  nichts. Im Normalfall findet diese Anfrage nicht statt.

Andere Ziele gibt es nicht. Das laesst sich mit jeder Firewall oder mit
Wireshark nachpruefen.

### Updates

Ein Update wird nie ungefragt installiert. Die App meldet, dass eine neue
Version vorliegt, und laedt sie erst nach deiner Bestaetigung.

Bevor die heruntergeladene Datei die laufende Exe ersetzt, wird ihre digitale
Signatur geprueft: sie muss gueltig sein, auf eine von Windows als
vertrauenswuerdig eingestufte Zertifizierungsstelle zurueckgehen und auf
`Open Source Developer Martin Wilke` ausgestellt sein
([authenticode.go, Zeilen 39-51](https://github.com/miwidot/tarkov-killscreen/blob/v1.0.13/authenticode.go#L39-L51)).
Eine ausgetauschte oder veraenderte Datei besteht diese Pruefung nicht und wird
geloescht statt installiert. Das schuetzt auch fuer den Fall, dass das
GitHub-Konto des Projekts uebernommen wuerde.

### Was die App auf dem Rechner liest, aber nicht sendet

Ehrlichkeitshalber gehoert das hierher, weil es beim Lesen des Codes auffaellt:

Die App fragt Windows nach der Liste der laufenden Prozesse
([windows.go, Zeile 125](https://github.com/miwidot/tarkov-killscreen/blob/v1.0.13/windows.go#L125)).
Sie prueft darin genau drei Dinge: laeuft `EscapeFromTarkov.exe`, laeuft Single
Player Tarkov (dann wird blockiert, weil SPT-Kills keine Online-Kills sind), und
laeuft ein Bildbetrachter (dann wird blockiert, damit kein bereits
hochgeladenes Bild erneut abfotografiert wird).

Diese Liste wird **nicht** uebertragen. Sie verlaesst den Rechner nicht. Sie
wird nur lokal ausgewertet, um zu entscheiden ob eine Aufnahme erlaubt ist. Im
Upload oben ist kein Feld dafuer vorgesehen.

### Was die App auf dem Rechner aendert

Auch das gehoert zur vollstaendigen Auskunft:

| Ort | Was | Wann |
|---|---|---|
| `%APPDATA%\TarkovKillcounter\config.json` | Deine Einstellungen und eine Sicherungskopie des Tokens | Immer |
| `%APPDATA%\TarkovKillcounter\device.id` | Die Zufalls-ID von oben | Immer |
| Windows-Anmeldeinformationsverwaltung, Eintrag `TarkovScreenshoter_APIToken` | Dein API-Token | Sobald du es eingibst |
| Registry `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, Wert `TarkovKillcounter` | Pfad zur Exe | Nur wenn du Autostart einschaltest |
| Registry `HKCU\Control Panel\Keyboard`, Wert `PrintScreenKeyForSnippingEnabled` | Wird auf 0 gesetzt | Nur wenn die Druck-Taste als Aufnahmetaste gewaehlt ist |
| Registry `HKCU\Software\TarkovKillcounter`, Wert `SnippingToolPrevious` | Merkt sich die vorherige Snipping-Tool-Einstellung | Nur solange die Zeile darueber aktiv ist |
| Die Exe selbst | Wird beim Update ersetzt | Nur nach deiner Bestaetigung |

Zur Snipping-Tool-Einstellung: Windows 11 oeffnet mit der Druck-Taste das
Snipping Tool. Damit die Taste fuer die Aufnahme frei ist, schaltet die App das
ab und merkt sich vorher, was eingestellt war
([hotkey.go, Zeilen 257-307](https://github.com/miwidot/tarkov-killscreen/blob/v1.0.13/hotkey.go#L257-L307)).
Beim Beenden ueber das Tray-Menue oder beim Wechsel auf eine andere Taste wird
genau dieser vorherige Zustand wiederhergestellt. War die Funktion schon vorher
abgeschaltet, wird sie nicht angefasst. Wird die App nicht sauber beendet — etwa
beim Herunterfahren von Windows — bleibt die Einstellung zunaechst abgeschaltet
und wird beim naechsten sauberen Beenden zurueckgesetzt.

Zur Sicherungskopie des Tokens: sie liegt mit AES verschluesselt in der
`config.json`, damit das Token nicht verloren geht wenn ein Aufraeum-Programm
die Anmeldeinformationsverwaltung leert. Der Schluessel dafuer steht im
oeffentlichen Quellcode
([credential.go, Zeile 58](https://github.com/miwidot/tarkov-killscreen/blob/v1.0.13/credential.go#L58)).
Das schuetzt also nur vor beilaeufigem Mitlesen, nicht vor jemandem, der Zugriff
auf dein Benutzerkonto hat. Behandle die `config.json` deshalb wie das Token
selbst und gib sie nicht weiter.

Ausserhalb dieser Orte schreibt die App nichts.

### Was die App nicht tut

- Die Zwischenablage wird nicht gelesen. Es gibt keinen einzigen
  Clipboard-Aufruf im Code. Eine sehr fruehe Version nutzte die Zwischenablage,
  die aktuelle nicht mehr.
- Keine Tastenprotokollierung. Es wird eine einzige Taste global registriert,
  die Aufnahmetaste. Andere Tastendruecke werden weder gelesen noch gespeichert.
- Kein Zugriff auf Spielespeicher, keine DLL-Injection, keine Veraenderung von
  Spieldateien, keine automatisierten Eingaben.
- Keine Telemetrie. Ausser dem oben Aufgelisteten wird nichts erhoben.
- Keine Dateien vom Rechner werden hochgeladen. Gelesen werden nur die eigene
  Konfiguration, die Device-ID und — falls du einen eigenen Kill-Sound
  eingestellt hast — diese eine WAV-Datei zum Abspielen.

### Wie du das selbst pruefst

1. **Quellcode lesen.** Dieses Repository ist das Programm. Die relevanten
   Dateien sind `upload.go` (was gesendet wird), `hotkey.go` (wann aufgenommen
   wird), `device.go` (die Device-ID) und `authenticode.go` (die Pruefung von
   Updates).
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
[hotkey.go line 419](https://github.com/miwidot/tarkov-killscreen/blob/v1.0.13/hotkey.go#L419).

There is no automatic capture, no timed capture and no background capture.
Without a key press, nothing happens.

### What is captured

The full screen of the monitor Tarkov runs on:
[hotkey.go line 450](https://github.com/miwidot/tarkov-killscreen/blob/v1.0.13/hotkey.go#L450).

Stated plainly: this is a full-screen shot of that one monitor, not a crop of
the kill area only. Because Tarkov has to be in the foreground, that screen
shows Tarkov. If you have an overlay on the same monitor, it is in the image.
Other monitors are not captured.

### What is sent

On upload to the OCR endpoint
([upload.go](https://github.com/miwidot/tarkov-killscreen/blob/v1.0.13/upload.go#L337-L373)):

| Field | Content |
|---|---|
| image | The screenshot as JPEG |
| `mode` | Operating mode from the configuration |
| `client_version` | App version, e.g. `1.0.13` |
| `Authorization` | Your API token, so the server knows whose kills these are |
| `X-Device-ID` | A random ID, see below |

After analysis the detected kill data is saved
([upload.go, SaveKills](https://github.com/miwidot/tarkov-killscreen/blob/v1.0.13/upload.go#L532-L575)):
map, timestamp, kill counts, weapons, the individual kill entries, image hashes,
the client version and — only if you selected one — the ID of the kill event.

**That is the complete list.** Nothing else is transmitted.

### Retries on connection problems

If an upload fails because the connection drops or the server is not answering,
the app tries again up to three times, after 5, 15 and 30 seconds
([app.go lines 409-430](https://github.com/miwidot/tarkov-killscreen/blob/v1.0.13/app.go#L409-L430)).
Exactly the same data is sent again, nothing extra. The screenshots are held in
memory only during that time and are never written to disk. After the last
attempt they are discarded.

### About the device ID

`X-Device-ID` is a random UUID v4, generated on first start with `crypto/rand`
and stored in `%APPDATA%\TarkovKillcounter\device.id`:
[device.go lines 20-53](https://github.com/miwidot/tarkov-killscreen/blob/v1.0.13/device.go#L20-L53).

It is **not** derived from your hardware. No serial numbers, no MAC address, no
Windows installation ID and no user name are read. It is a random number. Its
only purpose is binding your token to one device so a stolen token cannot be
used elsewhere. Delete the file and you get a new one.

### Where it is sent

[debug_release.go lines 15 and 19](https://github.com/miwidot/tarkov-killscreen/blob/v1.0.13/debug_release.go#L15-L19):

- `kc.tarkov-stammtisch.de` — the actual endpoint
- `kc.notgood.cc` — fallback address if the first one is unreachable

The second address looks unfamiliar at first glance, so to be clear: it is also
a server belonging to this project. It was introduced in version 1.0.9 because
broken ISP DNS configurations made the main address temporarily unreachable. It
is contacted only when the primary fails, and receives exactly the same data.

Additionally there are three connections that carry no user data — no token, no
device ID, no image:

- `api.github.com` — on start and every 30 minutes after that, the list of
  releases is queried to check for a newer version
  ([version.go line 64](https://github.com/miwidot/tarkov-killscreen/blob/v1.0.13/version.go#L64)).
- `github.com` and GitHub's download servers (`githubusercontent.com`) — only
  when you confirm an update, the new `screenshoter.exe` is downloaded from
  there
  ([selfupdate.go line 168](https://github.com/miwidot/tarkov-killscreen/blob/v1.0.13/selfupdate.go#L168)).
- `1.1.1.1` and `1.0.0.1` (Cloudflare DNS) — only when your system's name
  resolution fails, the server's host name is looked up there instead
  ([dns_fallback.go lines 27-38](https://github.com/miwidot/tarkov-killscreen/blob/v1.0.13/dns_fallback.go#L27-L38)).
  Cloudflare sees the host name being looked up and your IP address, nothing
  else. Normally this request does not happen at all.

There are no other destinations. Any firewall or Wireshark will confirm this.

### Updates

An update is never installed without asking. The app tells you a new version is
available and downloads it only after you confirm.

Before the downloaded file replaces the running exe, its digital signature is
checked: it must be valid, chain to a certificate authority Windows trusts, and
be issued to `Open Source Developer Martin Wilke`
([authenticode.go lines 39-51](https://github.com/miwidot/tarkov-killscreen/blob/v1.0.13/authenticode.go#L39-L51)).
A swapped or modified file fails this check and is deleted instead of
installed. This also protects you if the project's GitHub account were ever
taken over.

### What the app reads locally but does not send

This belongs here for the sake of honesty, because it stands out when reading
the code:

The app asks Windows for the list of running processes
([windows.go line 125](https://github.com/miwidot/tarkov-killscreen/blob/v1.0.13/windows.go#L125)).
It checks exactly three things in it: is `EscapeFromTarkov.exe` running, is
Single Player Tarkov running (captures are blocked then, since SPT kills are not
online kills), and is an image viewer running (blocked too, so an already
uploaded image cannot be re-captured).

That list is **not** transmitted. It never leaves the machine. It is only
evaluated locally to decide whether a capture is allowed. The upload above has
no field for it.

### What the app changes on your machine

This is part of a complete account as well:

| Location | What | When |
|---|---|---|
| `%APPDATA%\TarkovKillcounter\config.json` | Your settings and a backup copy of the token | Always |
| `%APPDATA%\TarkovKillcounter\device.id` | The random ID described above | Always |
| Windows Credential Manager, entry `TarkovScreenshoter_APIToken` | Your API token | Once you enter it |
| Registry `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, value `TarkovKillcounter` | Path to the exe | Only if you enable autostart |
| Registry `HKCU\Control Panel\Keyboard`, value `PrintScreenKeyForSnippingEnabled` | Set to 0 | Only while Print Screen is the capture key |
| Registry `HKCU\Software\TarkovKillcounter`, value `SnippingToolPrevious` | Remembers the previous Snipping Tool setting | Only while the row above is active |
| The exe itself | Replaced on update | Only after you confirm |

About the Snipping Tool setting: Windows 11 opens the Snipping Tool on Print
Screen. To free the key for capturing, the app switches that off and first
records what was set
([hotkey.go lines 257-307](https://github.com/miwidot/tarkov-killscreen/blob/v1.0.13/hotkey.go#L257-L307)).
When you exit through the tray menu or switch to a different key, exactly that
previous state is restored. If the feature was already off before, it is left
alone. If the app does not exit cleanly — for example when Windows shuts down —
the setting stays off for the moment and is put back on the next clean exit.

About the token backup: it is stored AES-encrypted in `config.json` so the
token is not lost when a cleanup tool empties the Credential Manager. The key
for it is in the public source code
([credential.go line 58](https://github.com/miwidot/tarkov-killscreen/blob/v1.0.13/credential.go#L58)).
So it only protects against casual reading, not against someone with access to
your user account. Treat `config.json` like the token itself and do not share
it.

The app writes nowhere else.

### What the app does not do

- The clipboard is not read. There is not a single clipboard call in the code.
  A very early version used the clipboard, the current one does not.
- No keylogging. Exactly one key is registered globally, the capture key. Other
  key presses are neither read nor stored.
- No game memory access, no DLL injection, no modification of game files, no
  automated input.
- No telemetry. Nothing is collected beyond what is listed above.
- No files are uploaded from your machine. The only files read are its own
  configuration, the device ID and — if you set a custom kill sound — that one
  WAV file, to play it.

### How to verify this yourself

1. **Read the source.** This repository is the program. The relevant files are
   `upload.go` (what is sent), `hotkey.go` (when a capture happens),
   `device.go` (the device ID) and `authenticode.go` (the update check).
2. **Capture the network traffic.** Wireshark or a firewall shows every
   connection. Only the addresses listed above may appear.
3. **Check the signature.** Right-click `screenshoter.exe`, Properties, Digital
   Signatures. Issued to Martin Wilke by Certum.
4. **Build it yourself.** If you do not trust the prebuilt exe, build from
   source. See "Building from Source" in the README.

On the antivirus warning, see the "Antivirus False Positives" section in the
[README](README.md).
