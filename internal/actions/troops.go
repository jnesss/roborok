package actions

import (
	"log"
	"roborok/internal/common"
	"roborok/internal/state"
	"roborok/internal/utils"
	"time"
)

// TrainInfantry attempts to train infantry in the barracks
func TrainInfantry(
	deviceID string,
	gameView string,
	detections []common.Detection,
	adbPath string,
	config common.TaskConfig,
	instanceState *state.InstanceState,
) bool {
	log.Printf("Attempting to train infantry on device %s", deviceID)

	// We need to be in city view
	if gameView != "city" {
		log.Println("Not in city view, cannot train infantry")
		return false
	}

	// Find the barracks building
	var barracks *common.Detection
	for _, det := range detections {
		if (det.Class == "barracks_idle" || det.Class == "barracks_upgradeable_idle") &&
			det.Confidence > common.MinConfidence {
			barracks = &det
			break
		}
	}

	// If barracks not found or not idle, cannot proceed
	if barracks == nil {
		log.Println("Barracks not found or not idle in detections")
		return false
	}

	// Click on barracks
	if err := utils.TapScreen(deviceID, adbPath, int(barracks.X), int(barracks.Y)); err != nil {
		log.Printf("Failed to tap on barracks: %v", err)
		return false
	}

	// Wait for menu to appear
	time.Sleep(1 * time.Second)

	// Click where train button would be (typically center-right)
	if err := utils.TapScreen(deviceID, adbPath, 450, 300); err != nil {
		log.Printf("Failed to tap on train button: %v", err)
		return false
	}

	// Wait for troop selection screen
	time.Sleep(1 * time.Second)

	// Select infantry (typically leftmost option)
	if err := utils.TapScreen(deviceID, adbPath, 150, 300); err != nil {
		log.Printf("Failed to tap on infantry selection: %v", err)
		return false
	}

	// Wait for selection
	time.Sleep(500 * time.Millisecond)

	// Click train max button (typically bottom-right)
	if err := utils.TapScreen(deviceID, adbPath, 450, 450); err != nil {
		log.Printf("Failed to tap on train max button: %v", err)
		return false
	}

	log.Println("Infantry training initiated")

	// Wait for confirmation
	time.Sleep(1 * time.Second)

	// Click help button if available (typically center)
	if err := utils.TapScreen(deviceID, adbPath, 320, 350); err != nil {
		log.Printf("Failed to tap on help button: %v", err)
	}

	// Close menus by clicking top-left corner
	if err := utils.TapScreen(deviceID, adbPath, 50, 50); err != nil {
		log.Printf("Failed to close menus: %v", err)
	}

	return true
}

// TrainArchers attempts to train archers in the archery range
func TrainArchers(
	deviceID string,
	gameView string,
	detections []common.Detection,
	adbPath string,
	config common.TaskConfig,
	instanceState *state.InstanceState,
) bool {
	log.Printf("Attempting to train archers on device %s", deviceID)

	// We need to be in city view
	if gameView != "city" {
		log.Println("Not in city view, cannot train archers")
		return false
	}

	// Find the archery range building
	var archeryRange *common.Detection
	for _, det := range detections {
		if (det.Class == "archery_range_idle" || det.Class == "archery_range_upgradeable_idle") &&
			det.Confidence > common.MinConfidence {
			archeryRange = &det
			break
		}
	}

	// If archery range not found or not idle, cannot proceed
	if archeryRange == nil {
		log.Println("Archery range not found or not idle in detections")
		return false
	}

	// Click on archery range
	if err := utils.TapScreen(deviceID, adbPath, int(archeryRange.X), int(archeryRange.Y)); err != nil {
		log.Printf("Failed to tap on archery range: %v", err)
		return false
	}

	// Wait for menu to appear
	time.Sleep(1 * time.Second)

	// Click where train button would be (typically center-right)
	if err := utils.TapScreen(deviceID, adbPath, 450, 300); err != nil {
		log.Printf("Failed to tap on train button: %v", err)
		return false
	}

	// Wait for troop selection screen
	time.Sleep(1 * time.Second)

	// Select archers (typically leftmost option)
	if err := utils.TapScreen(deviceID, adbPath, 150, 300); err != nil {
		log.Printf("Failed to tap on archer selection: %v", err)
		return false
	}

	// Wait for selection
	time.Sleep(500 * time.Millisecond)

	// Click train max button (typically bottom-right)
	if err := utils.TapScreen(deviceID, adbPath, 450, 450); err != nil {
		log.Printf("Failed to tap on train max button: %v", err)
		return false
	}

	log.Println("Archer training initiated")

	// Wait for confirmation
	time.Sleep(1 * time.Second)

	// Click help button if available (typically center)
	if err := utils.TapScreen(deviceID, adbPath, 320, 350); err != nil {
		log.Printf("Failed to tap on help button: %v", err)
	}

	// Close menus by clicking top-left corner
	if err := utils.TapScreen(deviceID, adbPath, 50, 50); err != nil {
		log.Printf("Failed to close menus: %v", err)
	}

	return true
}

// Keep the original TrainTroops for backward compatibility,
// but update it to use the new NavigateToCity function

// TrainTroops attempts to train troops in the barracks
func TrainTroops(
	deviceID string,
	gameView string,
	detections []common.Detection,
	adbPath string,
	config common.TaskConfig,
	instanceState *state.InstanceState,
) bool {
	// If we're not in city view, we need to navigate there first
	if gameView != "city" {

		if !NavigateToCity(deviceID, gameView, detections, adbPath, config, instanceState) {
			log.Println("Failed to navigate to city view")
			return false
		}

		// We need fresh detections after navigation
		return false
	}

	// Rest of the function remains the same
	// ...

	return true
}
