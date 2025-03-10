package actions

import (
	"log"
	"roborok/internal/common"
	"roborok/internal/state"
	"roborok/internal/utils"
	"time"
)

// CollectVIPRewards attempts to collect VIP rewards
func CollectVIPRewards(
	deviceID string,
	gameView string,
	detections []common.Detection,
	adbPath string,
	config common.TaskConfig,
	instanceState *state.InstanceState,
) bool {
	log.Printf("Attempting to collect VIP rewards on device %s", deviceID)

	// We need to be in city view
	if gameView != "city" {
		log.Println("Not in city view, cannot collect VIP rewards")
		return false
	}

	// Find VIP button in the UI
	var vipButton *common.Detection
	for _, det := range detections {
		if det.Class == "vip_button" && det.Confidence > common.MinConfidence {
			vipButton = &det
			break
		}
	}

	// If VIP button not found, try menu
	if vipButton == nil {
		log.Println("VIP button not found directly, trying via menu")

		// Look for menu button
		var menuButton *common.Detection
		for _, det := range detections {
			if det.Class == "menu_button" && det.Confidence > common.MinConfidence {
				menuButton = &det
				break
			}
		}

		// If menu button not found, cannot proceed
		if menuButton == nil {
			log.Println("Neither VIP button nor menu button found")
			return false
		}

		// Click menu button
		if err := utils.TapScreen(deviceID, adbPath, int(menuButton.X), int(menuButton.Y)); err != nil {
			log.Printf("Failed to tap on menu button: %v", err)
			return false
		}

		// Wait for menu to open, then click where VIP button would be
		time.Sleep(1 * time.Second)
		if err := utils.TapScreen(deviceID, adbPath, 320, 200); err != nil {
			log.Printf("Failed to tap on VIP menu item: %v", err)
			return false
		}
	} else {
		// Click on VIP button directly
		if err := utils.TapScreen(deviceID, adbPath, int(vipButton.X), int(vipButton.Y)); err != nil {
			log.Printf("Failed to tap on VIP button: %v", err)
			return false
		}
	}

	// Wait for VIP interface to open
	time.Sleep(1 * time.Second)

	// Click potential VIP points claim button location (left side)
	if err := utils.TapScreen(deviceID, adbPath, 150, 300); err != nil {
		log.Printf("Failed to tap on VIP points claim button: %v", err)
	} else {
		log.Println("Tapped on potential VIP points claim location")
		time.Sleep(1 * time.Second)
	}

	// Click potential VIP chest claim button location (right side)
	if err := utils.TapScreen(deviceID, adbPath, 450, 300); err != nil {
		log.Printf("Failed to tap on VIP chest claim button: %v", err)
	} else {
		log.Println("Tapped on potential VIP chest claim location")
		time.Sleep(1 * time.Second)
	}

	// Close the interface (typically top-right)
	if err := utils.TapScreen(deviceID, adbPath, 550, 50); err != nil {
		log.Printf("Failed to close VIP interface: %v", err)
	}

	log.Println("VIP rewards collection completed")
	return true
}
