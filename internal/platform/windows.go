package platform

import (
	"fmt"
	"os/exec"
	"strings"
)

type windowsPlatform struct{}

// psQuote renders s as a PowerShell single-quoted string literal. Inside a
// single-quoted literal PowerShell performs no interpolation, so doubling any
// embedded single quote fully neutralizes command injection.
func psQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

func (w *windowsPlatform) ServiceInstall(execPath string) error {
	script := fmt.Sprintf(
		`$action = New-ScheduledTaskAction -Execute %s -Argument '__run --owner service'; `+
			`$trigger = New-ScheduledTaskTrigger -AtLogOn; `+
			`$settings = New-ScheduledTaskSettingsSet -RestartCount 3 -RestartInterval (New-TimeSpan -Minutes 1); `+
			`Register-ScheduledTask -TaskName 'LazyrayXray' -Action $action -Trigger $trigger -Settings $settings -RunLevel Limited`,
		psQuote(execPath))

	return exec.Command("powershell", "-Command", script).Run()
}

func (w *windowsPlatform) ServiceUninstall() error {
	return exec.Command("powershell", "-Command",
		"Unregister-ScheduledTask -TaskName 'LazyrayXray' -Confirm:$false").Run()
}

func (w *windowsPlatform) ServiceStatus() (bool, bool, error) {
	out, err := exec.Command("powershell", "-Command",
		"(Get-ScheduledTask -TaskName 'LazyrayXray' -ErrorAction SilentlyContinue).State").Output()
	if err != nil {
		return false, false, nil
	}

	state := string(out)
	if state == "" {
		return false, false, nil
	}

	return true, state == "Running\r\n" || state == "Running\n", nil
}

func (w *windowsPlatform) Notify(title, message string) error {
	script := fmt.Sprintf(
		`[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null; `+
			`$template = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastText02); `+
			`$textNodes = $template.GetElementsByTagName('text'); `+
			`$textNodes.Item(0).AppendChild($template.CreateTextNode(%s)); `+
			`$textNodes.Item(1).AppendChild($template.CreateTextNode(%s)); `+
			`$toast = [Windows.UI.Notifications.ToastNotification]::new($template); `+
			`[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('lazyray').Show($toast)`,
		psQuote(title), psQuote(message))

	return exec.Command("powershell", "-Command", script).Run()
}

func (w *windowsPlatform) ClearQuarantine(_ string) error {
	return nil
}

func (w *windowsPlatform) OpenURL(url string) error {
	return exec.Command("cmd", "/c", "start", url).Run()
}
