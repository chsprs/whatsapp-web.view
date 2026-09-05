package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/go-toast/toast"
	"github.com/jchv/go-webview2"
	"golang.org/x/sys/windows"
)

var (
	kernel32                     = windows.NewLazySystemDLL("kernel32.dll")
	user32                       = windows.NewLazySystemDLL("user32.dll")
	dwmapi                       = windows.NewLazySystemDLL("dwmapi.dll")
	procCreateMutex              = kernel32.NewProc("CreateMutexW")
	procFindWindow               = user32.NewProc("FindWindowW")
	procSetFgWindow              = user32.NewProc("SetForegroundWindow")
	procShowNormal               = user32.NewProc("ShowWindow")
	procSetWindowPos             = user32.NewProc("SetWindowPos")
	procDwmSetAttr               = dwmapi.NewProc("DwmSetWindowAttribute")
	procCreateToolhelp32Snapshot = kernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32First           = kernel32.NewProc("Process32FirstW")
	procProcess32Next            = kernel32.NewProc("Process32NextW")
	procEnumWindows              = user32.NewProc("EnumWindows")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
	procGetClassName             = user32.NewProc("GetClassNameW")
	procGetWindowText            = user32.NewProc("GetWindowTextW")
)

const (
	windowTitle = "WhatsApp Desktop"
	appURL      = "https://web.whatsapp.com"
	mutexName   = "WhatsAppDesktopSingleInstanceMutex"
	userAgent   = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36"

	// SetWindowPos flags
	HWND_TOPMOST   = ^uintptr(0) // -1
	HWND_NOTOPMOST = ^uintptr(1) // -2
	SWP_NOSIZE     = 0x0001
	SWP_NOMOVE     = 0x0002
	SWP_NOACTIVATE = 0x0010
	SWP_SHOWWINDOW = 0x0040

	// DWM Window Attributes for Dark Theme
	DWMWA_USE_IMMERSIVE_DARK_MODE_BEFORE_20H1 = 19
	DWMWA_USE_IMMERSIVE_DARK_MODE             = 20
	DWMWA_CAPTION_COLOR                      = 35
	DWMWA_TEXT_COLOR                         = 36
)

type processEntry32 struct {
	Size            uint32
	Usage           uint32
	ProcessID       uint32
	DefaultHeapID   uintptr
	ModuleID        uint32
	Threads         uint32
	ParentProcessID uint32
	PriClassBase    int32
	Flags           uint32
	ExeFile         [260]uint16
}

func getAllRelatedPIDs(rootPID uint32) map[uint32]bool {
	pids := map[uint32]bool{rootPID: true}
	hSnap, _, _ := procCreateToolhelp32Snapshot.Call(2, 0) // TH32CS_SNAPPROCESS
	if hSnap == 0 || hSnap == uintptr(syscall.InvalidHandle) {
		return pids
	}
	defer windows.CloseHandle(windows.Handle(hSnap))

	var entry processEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))

	ret, _, _ := procProcess32First.Call(hSnap, uintptr(unsafe.Pointer(&entry)))
	parentMap := make(map[uint32]uint32)
	for ret != 0 {
		parentMap[entry.ProcessID] = entry.ParentProcessID
		ret, _, _ = procProcess32Next.Call(hSnap, uintptr(unsafe.Pointer(&entry)))
	}

	changed := true
	for changed {
		changed = false
		for child, parent := range parentMap {
			if pids[parent] && !pids[child] {
				pids[child] = true
				changed = true
			}
		}
	}
	return pids
}

func setAllAppWindowsAlwaysOnTop(rootPID uint32, enable bool) {
	target := HWND_NOTOPMOST
	if enable {
		target = HWND_TOPMOST
	}
	pids := getAllRelatedPIDs(rootPID)
	cb := syscall.NewCallback(func(hwnd uintptr, lParam uintptr) uintptr {
		vis, _, _ := procIsWindowVisible.Call(hwnd)
		if vis == 0 {
			return 1
		}
		var pid uint32
		procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
		if pids[pid] {
			var clsBuf [256]uint16
			procGetClassName.Call(hwnd, uintptr(unsafe.Pointer(&clsBuf[0])), 256)
			cls := syscall.UTF16ToString(clsBuf[:])

			// Skip IME system windows
			if cls != "MSCTFIME UI" && cls != "IME" {
				procSetWindowPos.Call(
					hwnd,
					target,
					0,
					0,
					0,
					0,
					SWP_NOMOVE|SWP_NOSIZE|SWP_NOACTIVATE|SWP_SHOWWINDOW,
				)
			}
		}
		return 1
	})
	procEnumWindows.Call(cb, 0)
}

func setAlwaysOnTop(hwnd uintptr, enable bool) {
	target := HWND_NOTOPMOST
	if enable {
		target = HWND_TOPMOST
	}
	procSetWindowPos.Call(
		hwnd,
		target,
		0,
		0,
		0,
		0,
		SWP_NOMOVE|SWP_NOSIZE|SWP_SHOWWINDOW,
	)
}

func setDarkWindowFrame(hwnd uintptr) {
	darkMode := int32(1)
	// Try standard DWMWA_USE_IMMERSIVE_DARK_MODE (Win10 20H1+ & Win11)
	procDwmSetAttr.Call(
		hwnd,
		uintptr(DWMWA_USE_IMMERSIVE_DARK_MODE),
		uintptr(unsafe.Pointer(&darkMode)),
		unsafe.Sizeof(darkMode),
	)
	// Try older Win10 build
	procDwmSetAttr.Call(
		hwnd,
		uintptr(DWMWA_USE_IMMERSIVE_DARK_MODE_BEFORE_20H1),
		uintptr(unsafe.Pointer(&darkMode)),
		unsafe.Sizeof(darkMode),
	)

	// Set dark caption color (COLORREF: 0x00111B21 WhatsApp Dark Header: RGB 17, 27, 33)
	captionColor := uint32(0x00211B11) // 0x00BBGGRR
	procDwmSetAttr.Call(
		hwnd,
		uintptr(DWMWA_CAPTION_COLOR),
		uintptr(unsafe.Pointer(&captionColor)),
		unsafe.Sizeof(captionColor),
	)

	// Set white caption text (RGB 255, 255, 255)
	textColor := uint32(0x00FFFFFF)
	procDwmSetAttr.Call(
		hwnd,
		uintptr(DWMWA_TEXT_COLOR),
		uintptr(unsafe.Pointer(&textColor)),
		unsafe.Sizeof(textColor),
	)
}

func checkSingleInstance() (uintptr, bool) {
	namePtr, _ := syscall.UTF16PtrFromString(mutexName)
	handle, _, err := procCreateMutex.Call(0, 1, uintptr(unsafe.Pointer(namePtr)))
	if err == windows.ERROR_ALREADY_EXISTS {
		titlePtr, _ := syscall.UTF16PtrFromString(windowTitle)
		hwnd, _, _ := procFindWindow.Call(0, uintptr(unsafe.Pointer(titlePtr)))
		if hwnd != 0 {
			procShowNormal.Call(hwnd, 9) // SW_RESTORE
			procSetFgWindow.Call(hwnd)
		}
		return handle, false
	}
	return handle, true
}

func getUserDataDir() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = os.Getenv("APPDATA")
		if configDir == "" {
			configDir = "."
		}
	}
	dir := filepath.Join(configDir, "WhatsAppDesktopLight", "UserData")
	_ = os.MkdirAll(dir, 0755)
	return dir
}

func sanitizeNotificationText(s string) string {
	// Strip CDATA terminator and dangerous control characters
	s = strings.ReplaceAll(s, "]]>", "")
	var sb strings.Builder
	for _, r := range s {
		if r >= 32 || r == '\n' || r == '\r' || r == '\t' {
			sb.WriteRune(r)
		}
	}
	res := strings.TrimSpace(sb.String())
	if len(res) > 250 {
		res = res[:250] + "..."
	}
	return res
}

func showNativeNotification(title, message, iconPath string) {
	title = sanitizeNotificationText(title)
	message = sanitizeNotificationText(message)
	if title == "" && message == "" {
		return
	}
	notification := toast.Notification{
		AppID:   "WhatsApp Desktop",
		Title:   title,
		Message: message,
		Icon:    iconPath,
	}
	_ = notification.Push()
}

func main() {
	_, isSingle := checkSingleInstance()
	if !isSingle {
		os.Exit(0)
	}
	userDataDir := getUserDataDir()
	executablePath, _ := os.Executable()
	iconFullPath := filepath.Join(filepath.Dir(executablePath), "icon.ico")

	opts := webview2.WebViewOptions{
		Window:    nil,
		Debug:     false,
		DataPath:  userDataDir,
		AutoFocus: true,
		WindowOptions: webview2.WindowOptions{
			Title:  windowTitle,
			Width:  1100,
			Height: 750,
			IconId: 2,
			Center: true,
		},
	}

	w := webview2.NewWithOptions(opts)
	if w == nil {
		log.Fatalln("Gagal inisialisasi WebView2")
	}
	defer w.Destroy()

	hwnd := uintptr(w.Window())
	setDarkWindowFrame(hwnd)

	w.SetTitle(windowTitle)
	w.SetSize(1100, 750, webview2.HintNone)

	rootPID := uint32(os.Getpid())

	// Bind native notification bridge
	_ = w.Bind("sendNativeNotification", func(title, body string) {
		go showNativeNotification(title, body, iconFullPath)
	})

	// Bind window topmost bridge for all related windows (main window + popups like "Move to new window")
	_ = w.Bind("setAlwaysOnTop", func(enable bool) {
		setAllAppWindowsAlwaysOnTop(rootPID, enable)
	})

	// Background ticker to keep newly spawned windows (like video call popups) pinned if topmost is active
	var (
		topmostActive bool
		topmostMu     sync.Mutex
	)
	_ = w.Bind("setAlwaysOnTop", func(enable bool) {
		topmostMu.Lock()
		topmostActive = enable
		topmostMu.Unlock()
		setAllAppWindowsAlwaysOnTop(rootPID, enable)
	})

	go func() {
		for {
			time.Sleep(1 * time.Second)
			topmostMu.Lock()
			active := topmostActive
			topmostMu.Unlock()
			if active {
				setAllAppWindowsAlwaysOnTop(rootPID, true)
			}
		}
	}()

	// Inject JS: User-Agent spoofing + Notification API polyfill connecting to Go native Toast + Video Call detector
	initScript := `
		// UserAgent override
		Object.defineProperty(navigator, 'userAgent', {
			get: () => '` + userAgent + `'
		});
		Object.defineProperty(navigator, 'appVersion', {
			get: () => '` + userAgent + `'
		});

		// Native Notification Polyfill for Windows Desktop Toast
		(function() {
			window.Notification = function(title, options) {
				options = options || {};
				var body = options.body || '';
				if (window.sendNativeNotification) {
					window.sendNativeNotification(title, body);
				}
				this.title = title;
				this.onclick = null;
				this.onclose = null;
				this.onerror = null;
				this.onshow = null;
			};
			window.Notification.permission = 'granted';
			window.Notification.requestPermission = function(callback) {
				var p = Promise.resolve('granted');
				if (typeof callback === 'function') {
					callback('granted');
				}
				return p;
			};
		})();

		// Video Call Auto Always-on-Top & Manual Hotkey (Ctrl+F11)
		(function() {
			let isManualPinned = false;
			let isCallActive = false;

			function updateTopmostState() {
				const shouldBeTop = isManualPinned || isCallActive;
				if (window.setAlwaysOnTop) {
					window.setAlwaysOnTop(shouldBeTop);
				}
			}

			// Toggle hotkey Ctrl + F11
			window.addEventListener('keydown', function(e) {
				if (e.ctrlKey && e.key === 'F11') {
					e.preventDefault();
					isManualPinned = !isManualPinned;
					updateTopmostState();
					if (window.sendNativeNotification) {
						window.sendNativeNotification(
							'WhatsApp Desktop',
							isManualPinned ? 'Mode Always-on-Top AKTIF (Selalu di Depan)' : 'Mode Always-on-Top NONAKTIF'
						);
					}
				}
			});

			// Periodic & mutation checking for active video/audio call
			function checkActiveCall() {
				// Detect active call elements in WhatsApp Web:
				// 1. HTML5 video elements that are playing
				// 2. Elements with call-related data-testid or aria-labels
				let callDetected = false;

				const videos = document.querySelectorAll('video');
				for (let i = 0; i < videos.length; i++) {
					const v = videos[i];
					if (v && !v.paused && v.readyState > 0 && v.srcObject) {
						callDetected = true;
						break;
					}
				}

				if (!callDetected) {
					// Fallback selector for WhatsApp Call floating banner / modal / new window popup
					const callContainers = document.querySelectorAll(
						'[data-testid="call-banner"], [data-testid="call-modal"], [data-testid="video-call"], [aria-label*="Call"], [aria-label*="Panggilan"]'
					);
					if (callContainers.length > 0) {
						for (let j = 0; j < callContainers.length; j++) {
							const el = callContainers[j];
							// Check if element is visible and contains call control buttons (hangup, mute, etc.)
							if (el.offsetParent !== null && el.querySelector('button')) {
								callDetected = true;
								break;
							}
						}
					}
				}

				// Check if current window or document is the popout call window itself
				if (!callDetected) {
					const url = window.location.href;
					if (url.includes('call') || document.title.includes('Call') || document.title.includes('Panggilan')) {
						callDetected = true;
					}
				}

				if (callDetected !== isCallActive) {
					isCallActive = callDetected;
					updateTopmostState();
				}
			}

			setInterval(checkActiveCall, 1000);
		})();
	`

	w.Init(initScript)
	w.Navigate(appURL)
	w.Run()
}
