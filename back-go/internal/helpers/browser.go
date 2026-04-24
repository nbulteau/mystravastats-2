package helpers

import (
	"fmt"
	"log"
	"mystravastats/internal/platform/runtimeconfig"
	"os/exec"
	"runtime"
)

func OpenBrowser(url string) {
	if isBrowserAutoOpenDisabled() {
		log.Printf("Browser auto-open disabled; open this URL manually: %s", url)
		return
	}

	var err error
	switch goos := runtime.GOOS; goos {
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Printf("Failed to open browser: %v", err)
	}
}

func isBrowserAutoOpenDisabled() bool {
	return !runtimeconfig.BoolValue("OPEN_BROWSER", true)
}
