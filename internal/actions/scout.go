package actions

import (
	"log"
	"roborok/internal/common"
	"roborok/internal/state"
	"roborok/internal/utils"
	"roborok/internal/vision"
	"time"
)

// ManageScouts handles scout management including expedition to fog
func ManageScouts(
	deviceID string,
	gameView string,
	detections []common.Detection,
	adbPath string,
	config common.TaskConfig,
	instanceState *state.InstanceState,
) bool {
	log.Printf("Managing scouts on device %s", deviceID)

	// If we're in city view, we need to go to the map view
	if gameView == "city" {
		// Use the standard navigation function instead of the local navigateToMap
		if !NavigateToMap(deviceID, gameView, detections, adbPath, config, instanceState) {
			log.Println("Failed to navigate to map view")
			return false
		}

		// We need fresh detections after navigation
		return false
	}

	// Check if scout is idle
	isScoutIdle, err := IsScoutIdle(deviceID, gameView, detections, adbPath)
	if err != nil {
		log.Printf("Failed to check if scout is idle: %v", err)
		return false
	}

	if !isScoutIdle {
		log.Println("Scout is not idle, skipping management")
		return false
	}

	// If scout is idle, send it to explore fog
	return SendScoutToFog(deviceID, gameView, detections, adbPath, config, instanceState)
}

// IsScoutIdle checks if the scout is idle
func IsScoutIdle(
	deviceID string,
	gameView string,
	detections []common.Detection,
	adbPath string,
) (bool, error) {
	// This is a placeholder implementation
	// In a real implementation, we would:
	// 1. Look for the scout camp in the city
	// 2. Look for indicators that scouts are available
	// 3. Check if there's a "New Scout" button

	// For testing, assume scout is idle
	return true, nil
}

// SendScoutToFog sends a scout to explore fog
func SendScoutToFog(
	deviceID string,
	gameView string,
	detections []common.Detection,
	adbPath string,
	config common.TaskConfig,
	instanceState *state.InstanceState,
) bool {
	log.Println("Sending scout to explore fog")

	// Reference the scout state from the instance state
	scoutState := &instanceState.ScoutState

	// Look for the scout camp or scout button
	var scoutButton *common.Detection
	for _, detection := range detections {
		if (detection.Class == "scout_camp" || detection.Class == "scout_button") && detection.Confidence > 0.7 {
			scoutButton = &detection
			break
		}
	}

	// If scout button not found, we can't proceed
	if scoutButton == nil {
		log.Println("Scout camp/button not found")
		return false
	}

	// Click on scout camp/button
	if err := utils.TapScreen(deviceID, adbPath, int(scoutButton.X), int(scoutButton.Y)); err != nil {
		log.Printf("Failed to tap on scout camp/button: %v", err)
		return false
	}

	// Wait for scout interface to open
	time.Sleep(1 * time.Second)

	// Take new screenshot and use CaptureAndDetect
	log.Println("Taking screenshot to find explore button")
	exploreDetections, err := vision.CaptureAndDetect(deviceID, adbPath)
	if err != nil {
		log.Printf("Failed to get detections for explore button: %v", err)
		return false
	}

	// Look for "Explore" button
	exploreButton := vision.FindDetectionByClass(exploreDetections, "explore_button", common.MinConfidence)

	// If explore button not found, close the interface and return
	if exploreButton == nil {
		log.Println("Explore button not found")
		// Try to close the interface
		utils.TapScreen(deviceID, adbPath, 10, 10)
		return false
	}

	// Click on explore button
	if err := utils.TapScreen(deviceID, adbPath, int(exploreButton.X), int(exploreButton.Y)); err != nil {
		log.Printf("Failed to tap on explore button: %v", err)
		return false
	}

	// Wait for confirmation dialog
	time.Sleep(1 * time.Second)

	// Take new screenshot to find march button
	log.Println("Taking screenshot to find march button")
	marchDetections, err := vision.CaptureAndDetect(deviceID, adbPath)
	if err != nil {
		log.Printf("Failed to get detections for march button: %v", err)
		return false
	}

	// Look for "March" button
	marchButton := vision.FindDetectionByClass(marchDetections, "march_button", common.MinConfidence)

	// If march button not found, close the dialog and return
	if marchButton == nil {
		log.Println("March button not found")
		// Try to close the dialog
		utils.TapScreen(deviceID, adbPath, 10, 10)
		return false
	}

	// Click on march button
	if err := utils.TapScreen(deviceID, adbPath, int(marchButton.X), int(marchButton.Y)); err != nil {
		log.Printf("Failed to tap on march button: %v", err)
		return false
	}

	log.Println("Scout sent to explore fog successfully")

	// Update scout state
	scoutState.IsMoving = true
	scoutState.LastMoveTime = time.Now()

	return true
}
