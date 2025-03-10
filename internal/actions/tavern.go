package actions

import (
	"log"
	"roborok/internal/common"
	"roborok/internal/state"
	"roborok/internal/utils"
	"time"
)

// CollectTavernChests attempts to collect free chests from the tavern
func CollectTavernChests(
	deviceID string,
	gameView string,
	detections []common.Detection,
	adbPath string,
	config common.TaskConfig,
	instanceState *state.InstanceState,
) bool {
	log.Printf("Attempting to collect tavern chests on device %s", deviceID)

	// We need to be in city view
	if gameView != "city" {
		log.Println("Not in city view, cannot collect tavern chests")
		return false
	}

	// Check if tavern icon shows clickable indicator
	var tavern *common.Detection
	for _, detection := range detections {
		if (detection.Class == "tavern_clickable" || detection.Class == "tavern_upgradeable_clickable") &&
			detection.Confidence > common.MinConfidence {
			tavern = &detection
			break
		}
	}

	// If tavern not found or not clickable, nothing to do
	if tavern == nil {
		log.Println("Tavern not found or not clickable in detections")
		return false
	}

	// Click on tavern
	if err := utils.TapScreen(deviceID, adbPath, int(tavern.X), int(tavern.Y)); err != nil {
		log.Printf("Failed to tap on tavern: %v", err)
		return false
	}

	// Wait for tavern interface to load
	time.Sleep(1 * time.Second)

	// Click where chest claim buttons would be
	// Silver chest is typically in the middle-left
	if err := utils.TapScreen(deviceID, adbPath, 150, 300); err != nil {
		log.Printf("Failed to tap on silver chest claim: %v", err)
	} else {
		log.Println("Tapped on potential silver chest location")
		// Wait for chest animation
		time.Sleep(1 * time.Second)

		// Click to dismiss rewards
		utils.TapScreen(deviceID, adbPath, 300, 400) // Center of screen
		time.Sleep(1 * time.Second)

		instanceState.TavernState.LastSilverChestTime = time.Now()
	}

	// Gold chest is typically in the middle-right
	if err := utils.TapScreen(deviceID, adbPath, 450, 300); err != nil {
		log.Printf("Failed to tap on gold chest claim: %v", err)
	} else {
		log.Println("Tapped on potential gold chest location")
		// Wait for chest animation
		time.Sleep(1 * time.Second)

		// Click to dismiss rewards
		utils.TapScreen(deviceID, adbPath, 300, 400) // Center of screen
		time.Sleep(1 * time.Second)
	}

	// Close tavern interface (typically top-right corner)
	if err := utils.TapScreen(deviceID, adbPath, 550, 50); err != nil {
		log.Printf("Failed to close tavern interface: %v", err)
	}

	log.Println("Tavern chest collection completed")
	return true
}
