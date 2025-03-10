package actions

import (
	"log"
	"roborok/internal/common"
	"roborok/internal/state"
	"roborok/internal/utils"
	"time"
)

// NavigateToCity navigates to the city view
func NavigateToCity(
	deviceID string,
	gameView string,
	detections []common.Detection,
	adbPath string,
	config common.TaskConfig,
	instanceState *state.InstanceState,
) bool {
	// If already in city view, nothing to do
	if gameView == "city" {
		return true
	}

	log.Printf("Navigating to city view from %s view", gameView)

	// Look for return to city button
	var returnButton *common.Detection
	for _, det := range detections {
		if det.Class == "return_to_city_button" && det.Confidence > common.MinConfidence {
			returnButton = &det
			break
		}
	}

	// If return button not found, cannot navigate to city
	if returnButton == nil {
		log.Println("Return to city button not found")
		return false
	}

	// Click on return button
	if err := utils.TapScreen(deviceID, adbPath, int(returnButton.X), int(returnButton.Y)); err != nil {
		log.Printf("Error tapping on return button: %v", err)
		return false
	}

	// Wait for navigation
	time.Sleep(2 * time.Second)

	return true
}

// NavigateToMap navigates to the map view
func NavigateToMap(
	deviceID string,
	gameView string,
	detections []common.Detection,
	adbPath string,
	config common.TaskConfig,
	instanceState *state.InstanceState,
) bool {
	// If already in map view, nothing to do
	if gameView == "map" || gameView == "field" {
		return true
	}

	log.Printf("Navigating to map view from %s view", gameView)

	// Look for map button in the UI
	var mapButton *common.Detection
	for _, det := range detections {
		if det.Class == "map_button" && det.Confidence > common.MinConfidence {
			mapButton = &det
			break
		}
	}

	// If map button not found, cannot navigate to map
	if mapButton == nil {
		log.Println("Map button not found")

		// Try clicking at the expected location of the map button (lower left)
		if err := utils.TapScreen(deviceID, adbPath, 50, 800); err != nil {
			log.Printf("Error tapping approximate map button location: %v", err)
			return false
		}

		log.Println("Tried clicking approximate map button location")
		time.Sleep(2 * time.Second)
		return true
	}

	// Click on map button
	if err := utils.TapScreen(deviceID, adbPath, int(mapButton.X), int(mapButton.Y)); err != nil {
		log.Printf("Error tapping on map button: %v", err)
		return false
	}

	// Wait for navigation
	time.Sleep(2 * time.Second)

	return true
}

// ReturnToCity is a dedicated task for periodic return to city
func ReturnToCity(
	deviceID string,
	gameView string,
	detections []common.Detection,
	adbPath string,
	config common.TaskConfig,
	instanceState *state.InstanceState,
) bool {
	// Only execute if we're not already in the city
	if gameView == "city" {
		return false
	}

	log.Printf("Executing periodic return to city from %s view", gameView)

	// Use the shared navigation function
	if NavigateToCity(deviceID, gameView, detections, adbPath, config, instanceState) {
		log.Println("Successfully returned to city")
		return true
	}

	log.Println("Failed to return to city")
	return false
}
