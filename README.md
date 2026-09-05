# WhatsApp Webview Desktop

Lightweight Windows desktop wrapper for [WhatsApp Web](https://web.whatsapp.com), built with Go and Microsoft Edge WebView2.

## Features

- Native Windows WebView2 window instead of Electron.
- Persistent WhatsApp session across app restarts.
- Cookies, LocalStorage, IndexedDB, and service-worker data stored in a dedicated profile.
- Dark Windows title bar and frame.
- Windows toast notification bridge.
- Camera and microphone access for WhatsApp voice and video calls.
- **Auto Always-on-Top saat Video Call:** Jendela WhatsApp otomatis menjadi *always-on-top* di depan aplikasi lain saat mendeteksi panggilan/video call aktif, dan kembali normal saat selesai.
- **Manual Always-on-Top Toggle:** Tekan `Ctrl + F11` untuk mengunci jendela selalu di depan (pin on top) kapan saja secara manual.
- Single-instance protection.
- High-DPI display support.
- Small native executable.

## Requirements

- Windows 10 or newer.
- Microsoft Edge WebView2 Runtime. The app can download it automatically when missing.
- WhatsApp account paired with WhatsApp Web.

## Download & Installation

Unduh file dari [GitHub Releases](https://github.com/chsprs/whatsapp-web.view/releases/latest):

1. **Installer Setup (Disarankan)**: Unduh `WhatsApp-Setup.exe`. Jalankan installer untuk memasang otomatis ke Windows, menambahkan shortcut di **Start Menu** (otomatis muncul di Windows Search) dan **Desktop**, serta menyediakan fitur Uninstall di Windows Settings.
2. **Portable Executable**: Unduh `WhatsApp.exe` untuk penggunaan langsung tanpa instalasi.

Jalankan aplikasi, scan kode QR dengan aplikasi WhatsApp di ponsel, izinkan akses kamera/mikrofon saat diminta Windows jika ingin memakai voice/video call. Sesi login tersimpan otomatis.

## Security & Hardening

- **Sanitized Notification Bridge:** Native toast text is sanitized to prevent CDATA breakout or malicious script execution via Windows notification polyfill.
- **Reproducible Clean Build:** Windows resource object (`rsrc.syso`) is regenerated directly from manifest and clean icon.
- **Isolated Storage:** Data stored strictly within local `%APPDATA%\WhatsAppDesktopLight\UserData`.
- **Zero Telemetry:** Direct HTTPS connection only to `https://web.whatsapp.com`.

## Session Data

Profile data is stored at:

```text
%APPDATA%\WhatsAppDesktopLight\UserData
```

Do not delete this folder if the existing login session must remain available. Closing the app does not clear session data.

## Build From Source

Install Go and a Windows C compiler, then run:

```powershell
go mod download
go build -ldflags="-H windowsgui -s -w" -o WhatsApp.exe .
```

The repository includes the Windows manifest and embedded icon resource used by the build.

## Project Files

- `main.go`: WebView2 window, persistent profile, dark frame, and notifications.
- `app.manifest`: Windows DPI and application manifest.
- `resource.rc`: Windows icon and manifest resource definitions.
- `icon.ico`: Application icon.

## Privacy

This app loads WhatsApp Web directly. Chat data and authentication state are handled by WhatsApp Web and stored locally in the WebView2 profile above. This project is not affiliated with WhatsApp or Meta.

## License

No license has been declared yet.
