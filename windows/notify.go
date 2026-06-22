//go:build windows

package main

import (
	"fmt"
	"io"
	"os"
)

// toastScript builds a Windows toast notification using the WinRT toast API
// via PowerShell (no CGO, no third-party deps). Title and body are passed
// through environment variables to avoid quoting/injection issues.
const toastScript = `
$ErrorActionPreference = 'Stop'
try {
    [Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
    $template = [Windows.UI.Notifications.ToastTemplateType]::ToastText02
    $xml = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent($template)
    $texts = $xml.GetElementsByTagName('text')
    [void]$texts.Item(0).AppendChild($xml.CreateTextNode($env:HOTFIX_TOAST_TITLE))
    [void]$texts.Item(1).AppendChild($xml.CreateTextNode($env:HOTFIX_TOAST_BODY))
    $toast = [Windows.UI.Notifications.ToastNotification]::new($xml)
    $notifier = [Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('Hotfix')
    $notifier.Show($toast)
} catch {
    # Fall back to a tray balloon tip if WinRT toasts are unavailable.
    Add-Type -AssemblyName System.Windows.Forms
    $n = New-Object System.Windows.Forms.NotifyIcon
    $n.Icon = [System.Drawing.SystemIcons]::Information
    $n.BalloonTipTitle = $env:HOTFIX_TOAST_TITLE
    $n.BalloonTipText = $env:HOTFIX_TOAST_BODY
    $n.Visible = $true
    $n.ShowBalloonTip(5000)
    Start-Sleep -Seconds 6
    $n.Dispose()
}
`

// notifyToast shows a Windows desktop notification with the given title/body.
// It runs PowerShell hidden (no flash window) and never blocks the caller.
func notifyToast(title, body string) {
	go func() {
		cmd := hiddenCmd("powershell", "-NonInteractive", "-NoProfile",
			"-WindowStyle", "Hidden", "-Command", toastScript)
		cmd.Env = append(os.Environ(),
			"HOTFIX_TOAST_TITLE="+title,
			"HOTFIX_TOAST_BODY="+body,
		)
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard

		if err := cmd.Run(); err != nil {
			logf("notify: toast failed: %v", err)
		}
	}()
}

// notifyKilledToast is the desktop-notification counterpart to the tray label
// update in notifyKilled. Kept separate so callers can choose either or both.
func notifyKilledToast(name string, pid int, cpu float64) {
	title := "Hotfix — Process Terminated"
	body := fmt.Sprintf("%s (PID %d) was using %.0f%% CPU and has been terminated.", name, pid, cpu)
	notifyToast(title, body)
}
