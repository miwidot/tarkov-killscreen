// Tarkov Kill Screen Analyzer - Diagnostic Tool
//
// Standalone CLI helper that scans running processes and reports which ones
// might prevent the main screenshoter from capturing or uploading.
//
// Categories checked:
//   1. SPT (Single Player Tarkov) — blocks captures completely
//   2. Image viewers/editors with visible windows — block captures
//   3. Game recording overlays — overlay corrupts OCR
//   4. Screenshot tools — may have an image open and trigger re-capture detection
//   5. Whether EscapeFromTarkov.exe itself is running
//
// Build (release): GOOS=windows GOARCH=amd64 go build -o killcounter_diag.exe ./tools/diag
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

const (
	TH32CS_SNAPPROCESS = 0x00000002
	MAX_PATH           = 260
)

type PROCESSENTRY32W struct {
	Size              uint32
	Usage             uint32
	ProcessID         uint32
	DefaultHeapID     uintptr
	ModuleID          uint32
	Threads           uint32
	ParentProcessID   uint32
	PriClassBase      int32
	Flags             uint32
	ExeFile           [MAX_PATH]uint16
}

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	user32   = syscall.NewLazyDLL("user32.dll")

	procCreateSnapshot = kernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32First = kernel32.NewProc("Process32FirstW")
	procProcess32Next  = kernel32.NewProc("Process32NextW")
	procCloseHandle    = kernel32.NewProc("CloseHandle")

	procEnumWindows              = user32.NewProc("EnumWindows")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
	procIsIconic                 = user32.NewProc("IsIconic")
)

type category struct {
	name           string
	severity       string // "BLOCK", "WARN", "INFO"
	reason         string
	exes           []string
	blocksOnlyIfVisible bool // matches main app behaviour: only blocks when window is visible
	alwaysShow     bool // show even when not actively blocking (e.g. SPT, Tarkov)
}

// Categories — keep in sync with the main app's checks in windows.go.
var categories = []category{
	{
		name:       "SPT (Single Player Tarkov)",
		severity:   "BLOCK",
		reason:     "Wenn SPT laeuft, blockt der Killcounter alle Captures (offline kills sind keine echten kills).",
		alwaysShow: true,
		exes: []string{
			"spt.launcher.exe",
			"spt.server.exe",
			"spt.",
		},
	},
	{
		name:       "Tarkov (online client)",
		severity:   "INFO",
		reason:     "Captures funktionieren nur wenn der echte Tarkov-Client laeuft.",
		alwaysShow: true,
		exes: []string{
			"escapefromtarkov.exe",
		},
	},
	{
		name:                "Bildbetrachter / Editor",
		severity:            "BLOCK",
		blocksOnlyIfVisible: true,
		reason:              "Sichtbares Fenster eines Bildbetrachters blockt Captures (Re-Capture-Schutz).",
		exes: []string{
			"microsoft.photos.exe", "photos.exe", "mspaint.exe",
			"irfanview.exe", "i_view64.exe", "i_view32.exe",
			"xnview.exe", "xnviewmp.exe", "honeyview.exe", "jpegview.exe",
			"faststone.exe", "fsviewer.exe", "fsimageresize.exe",
			"imageglass.exe", "nomacs.exe", "picasa3.exe",
			"123photoviewer.exe", "apowersoftviewer.exe",
			"acdsee.exe", "acdseeultimate.exe", "acdseestandard.exe",
			"photoshop.exe", "lightroom.exe", "bridge.exe",
			"photoshopelements.exe", "illustrator.exe", "adobephotoshopexpress.exe",
			"gimp-2.10.exe", "gimp-2.8.exe", "gimp.exe",
			"paint.net.exe", "paintdotnet.exe", "krita.exe",
			"affinity photo.exe", "afphoto.exe",
			"coreldraw.exe", "paintshoppro.exe", "pspx.exe", "captureone.exe",
			"darktable.exe", "rawtherapee.exe",
		},
	},
	{
		name:                "Recording Overlay",
		severity:            "BLOCK",
		blocksOnlyIfVisible: true,
		reason:              "Game-Recording-Overlays koennen den Screenshot mit ihrem Overlay verfaelschen — OCR schlaegt fehl.",
		exes: []string{
			"outplayed.exe", "insightscapture.exe", "insights capture.exe",
			"plays.exe", "playstv.exe", "plays_ep64.exe",
			"action.exe", "bandicam.exe", "bdcam.exe", "fraps.exe",
			"xsplit.exe", "xsplitbroadcaster.exe", "xsplitgamecaster.exe",
			"d3dgear.exe", "litecam.exe", "raptr.exe", "relive.exe",
			"nvspcaps64.exe", "nvshare.exe", "nvidia share.exe",
		},
	},
	{
		name:                "Screenshot-Tool",
		severity:            "WARN",
		blocksOnlyIfVisible: true,
		reason:              "Screenshot-Tools koennen ein Bild offen haben und Re-Capture ausloesen.",
		exes: []string{
			"snagit32.exe", "snagit64.exe", "snagiteditor.exe",
			"greenshot.exe", "sharex.exe", "lightshot.exe",
			"picpick.exe", "screenpresso.exe",
		},
	},
}

type runningMatch struct {
	cat        category
	exe        string
	pid        uint32
	hasVisible bool
}

func main() {
	fmt.Println("===========================================================")
	fmt.Println("  Tarkov Killcounter — Diagnostic")
	fmt.Println("===========================================================")
	fmt.Println()

	processes, err := enumProcesses()
	if err != nil {
		fmt.Println("FEHLER: konnte Prozessliste nicht lesen:", err)
		pause()
		return
	}

	fmt.Printf("Gescannt: %d laufende Prozesse\n\n", len(processes))

	var matches []runningMatch
	for _, cat := range categories {
		for _, p := range processes {
			lower := strings.ToLower(p.name)
			if !matchesAny(lower, cat.exes) {
				continue
			}
			visible := hasVisibleWindow(p.pid)
			// Skip background-only matches that don't actively block
			// (e.g. Xbox Game Bar in background — only blocks if window visible)
			if cat.blocksOnlyIfVisible && !visible {
				continue
			}
			matches = append(matches, runningMatch{
				cat:        cat,
				exe:        p.name,
				pid:        p.pid,
				hasVisible: visible,
			})
		}
	}

	if len(matches) == 0 {
		fmt.Println("OK — kein blockendes Programm laeuft.")
		fmt.Println()
		fmt.Println("Wenn der Killcounter trotzdem keine Uploads macht:")
		fmt.Println("  - Laeuft EscapeFromTarkov.exe?")
		fmt.Println("  - Ist der API-Token in den Settings korrekt?")
		fmt.Println("  - Pruefe die Internetverbindung.")
		pause()
		return
	}

	// Group by category
	byCategory := map[string][]runningMatch{}
	order := []string{}
	for _, m := range matches {
		key := m.cat.name
		if _, ok := byCategory[key]; !ok {
			order = append(order, key)
		}
		byCategory[key] = append(byCategory[key], m)
	}

	for _, name := range order {
		group := byCategory[name]
		first := group[0].cat
		fmt.Printf("[%s] %s\n", first.severity, name)
		fmt.Printf("  Grund: %s\n", first.reason)
		fmt.Println("  Gefunden:")
		for _, m := range group {
			windowState := ""
			if m.hasVisible {
				windowState = "(sichtbares Fenster)"
			} else if first.alwaysShow {
				windowState = "(im Hintergrund)"
			}
			fmt.Printf("    - %s (PID %d) %s\n", m.exe, m.pid, windowState)
		}
		fmt.Println()
	}

	fmt.Println("Erklaerung:")
	fmt.Println("  [BLOCK]  Blockiert Captures definitiv")
	fmt.Println("  [WARN]   Kann Probleme machen")
	fmt.Println("  [INFO]   Zur Info — kein Block")
	fmt.Println()
	fmt.Println("Hintergrund-Prozesse die NICHT aktiv blocken (z.B. Xbox Game Bar")
	fmt.Println("ohne offenes Overlay) werden hier bewusst nicht angezeigt.")
	fmt.Println()
	fmt.Println("Loesung: blockende Programme schliessen oder Fenster minimieren,")
	fmt.Println("dann Killcounter neu starten.")

	pause()
}

type procInfo struct {
	name string
	pid  uint32
}

func enumProcesses() ([]procInfo, error) {
	snapshot, _, err := procCreateSnapshot.Call(TH32CS_SNAPPROCESS, 0)
	if snapshot == 0 || snapshot == ^uintptr(0) {
		return nil, err
	}
	defer procCloseHandle.Call(snapshot)

	var entry PROCESSENTRY32W
	entry.Size = uint32(unsafe.Sizeof(entry))

	ret, _, _ := procProcess32First.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
	if ret == 0 {
		return nil, fmt.Errorf("Process32First returned 0")
	}

	var result []procInfo
	for {
		name := syscall.UTF16ToString(entry.ExeFile[:])
		result = append(result, procInfo{name: name, pid: entry.ProcessID})

		ret, _, _ = procProcess32Next.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
		if ret == 0 {
			break
		}
	}
	return result, nil
}

func matchesAny(exeLower string, patterns []string) bool {
	for _, p := range patterns {
		p = strings.ToLower(p)
		if strings.HasSuffix(p, ".") {
			if strings.HasPrefix(exeLower, p) {
				return true
			}
			continue
		}
		if exeLower == p {
			return true
		}
		stripped := strings.TrimSuffix(p, ".exe")
		if stripped != "" && strings.Contains(exeLower, stripped) && len(stripped) >= 5 {
			return true
		}
	}
	return false
}

var (
	currentTargetPid uint32
	currentFound     bool
	enumCb           uintptr
)

func hasVisibleWindow(pid uint32) bool {
	if enumCb == 0 {
		enumCb = syscall.NewCallback(func(hwnd uintptr, lParam uintptr) uintptr {
			var p uint32
			procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&p)))
			if p == currentTargetPid {
				visible, _, _ := procIsWindowVisible.Call(hwnd)
				minimized, _, _ := procIsIconic.Call(hwnd)
				if visible != 0 && minimized == 0 {
					currentFound = true
					return 0
				}
			}
			return 1
		})
	}
	currentTargetPid = pid
	currentFound = false
	procEnumWindows.Call(enumCb, 0)
	return currentFound
}

func pause() {
	fmt.Println()
	fmt.Print("Druecke Enter zum Schliessen... ")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}
