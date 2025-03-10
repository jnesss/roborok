package utils

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// TapScreen simulates a tap at the given coordinates
func TapScreen(deviceID, adbPath string, x, y int) error {
	cmd := exec.Command(
		adbPath,
		"-s",
		deviceID,
		"shell",
		"input",
		"tap",
		fmt.Sprintf("%d", x),
		fmt.Sprintf("%d", y),
	)

	return cmd.Run()
}

// SwipeScreen simulates a swipe from (x1, y1) to (x2, y2) with the given duration
func SwipeScreen(deviceID, adbPath string, x1, y1, x2, y2, durationMS int) error {
	cmd := exec.Command(
		adbPath,
		"-s",
		deviceID,
		"shell",
		"input",
		"swipe",
		fmt.Sprintf("%d", x1),
		fmt.Sprintf("%d", y1),
		fmt.Sprintf("%d", x2),
		fmt.Sprintf("%d", y2),
		fmt.Sprintf("%d", durationMS),
	)

	return cmd.Run()
}

// SendText sends text input to the device
func SendText(deviceID, adbPath, text string) error {
	// Replace spaces with %s
	text = strings.Replace(text, " ", "%s", -1)

	cmd := exec.Command(
		adbPath,
		"-s",
		deviceID,
		"shell",
		"input",
		"text",
		text,
	)

	return cmd.Run()
}

// PressKey simulates pressing a key
func PressKey(deviceID, adbPath, keycode string) error {
	cmd := exec.Command(
		adbPath,
		"-s",
		deviceID,
		"shell",
		"input",
		"keyevent",
		keycode,
	)

	return cmd.Run()
}

// IsDeviceConnected checks if the device is connected
func IsDeviceConnected(deviceID, adbPath string) bool {
	cmd := exec.Command(adbPath, "devices")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// Check if device ID is in the output
	return strings.Contains(string(output), deviceID)
}

// RestartApp closes and reopens the ROK app
func RestartApp(deviceID, adbPath string) error {
	// Force stop ROK app
	stopCmd := exec.Command(
		adbPath,
		"-s",
		deviceID,
		"shell",
		"am",
		"force-stop",
		"com.lilithgame.roc.gp", // ROK package name
	)
	if err := stopCmd.Run(); err != nil {
		return fmt.Errorf("failed to stop ROK app: %w", err)
	}

	// Wait a bit
	time.Sleep(2 * time.Second)

	// Start ROK app
	startCmd := exec.Command(
		adbPath,
		"-s",
		deviceID,
		"shell",
		"monkey",
		"-p",
		"com.lilithgame.roc.gp",
		"-c",
		"android.intent.category.LAUNCHER",
		"1",
	)
	if err := startCmd.Run(); err != nil {
		return fmt.Errorf("failed to start ROK app: %w", err)
	}

	// Wait for app to load
	time.Sleep(10 * time.Second)

	return nil
}
