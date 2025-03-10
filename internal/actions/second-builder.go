package actions

import (
	"log"
	"roborok/internal/common"
	"roborok/internal/state"
	"roborok/internal/utils"
	"roborok/internal/vision"
	"time"
)

// Global state tracking for second builder
var (
	secondBuilderAdded bool // Whether the second builder has been added
)

// RecruitSecondBuilder attempts to recruit the second builder
func RecruitSecondBuilder(
	deviceID string,
	gameView string,
	detections []common.Detection,
	adbPath string,
	config common.TaskConfig,
	instanceState *state.InstanceState,
) bool {
	// Skip if already completed
	if instanceState.SecondBuilderAdded {
		log.Println("Second builder already added")
		return false
	}

	log.Printf("Attempting to recruit second builder on device %s", deviceID)

	// First, we need to make sure we're in city view
	if gameView != "city" {
		log.Println("Not in city view, cannot recruit second builder")
		return false
	}

	// Step 1: Find and click on the Builder's Hut
	var buildersHut *common.Detection
	for _, det := range detections {
		if (det.Class == "builders_hut_idle" || det.Class == "builders_hut") && det.Confidence > common.MinConfidence {
			buildersHut = &det
			break
		}
	}

	if buildersHut == nil {
		log.Println("Builder's Hut not found in detections")
		return false
	}

	// Click on Builder's Hut
	log.Printf("Step 1: Clicking on Builder's Hut at (%.1f, %.1f)", buildersHut.X, buildersHut.Y)
	if err := utils.TapScreen(deviceID, adbPath, int(buildersHut.X), int(buildersHut.Y)); err != nil {
		log.Printf("Error clicking on Builder's Hut: %v", err)
		return false
	}

	// Wait longer for menu to appear and any help bubbles to show
	log.Println("Waiting for builder's hut menu to fully appear...")
	time.Sleep(5 * time.Second)

	// Step 2: Look for builders_hut_button
	log.Println("Step 2: Looking for builders_hut_button...")
	hireButtonDetections, err := vision.CaptureAndDetect(deviceID, adbPath)
	if err != nil {
		log.Printf("Error getting detections for builders_hut_button: %v", err)
		resetView(deviceID, adbPath)
		return false
	}

	// Log what we see for debugging
	log.Printf("Detected %d objects after clicking Builder's Hut:", len(hireButtonDetections))
	for i, det := range hireButtonDetections {
		if det.Confidence > common.MinConfidence {
			log.Printf("  %d. %s (%.2f): (%.1f, %.1f) %.0fx%.0f",
				i+1, det.Class, det.Confidence, det.X, det.Y, det.Width, det.Height)
		}
	}

	var buildersHutButton *common.Detection
	for _, det := range hireButtonDetections {
		if det.Class == "builders_hut_button" && det.Confidence > common.MinConfidence {
			buildersHutButton = &det
			break
		}
	}

	if buildersHutButton == nil {
		log.Println("builders_hut_button not found, looking for alternative buttons...")
		// Try alternative buttons like "confirm_button" or general buttons at the bottom
		for _, det := range hireButtonDetections {
			if (det.Class == "confirm_button" || det.Class == "button") &&
				det.Y > 350 && det.Confidence > common.MinConfidence {
				buildersHutButton = &det
				log.Printf("Found alternative button: %s at (%.1f, %.1f)",
					det.Class, det.X, det.Y)
				break
			}
		}

		if buildersHutButton == nil {
			log.Println("No suitable button found, resetting view")
			resetView(deviceID, adbPath)
			return false
		}
	}

	// Click on builders_hut_button
	log.Printf("Clicking on builders_hut_button at (%.1f, %.1f)", buildersHutButton.X, buildersHutButton.Y)
	if err := utils.TapScreen(deviceID, adbPath, int(buildersHutButton.X), int(buildersHutButton.Y)); err != nil {
		log.Printf("Error clicking on builders_hut_button: %v", err)
		resetView(deviceID, adbPath)
		return false
	}

	// Wait longer for hire dialog and any help bubbles to show
	log.Println("Waiting for hire dialog to fully appear...")
	time.Sleep(3 * time.Second)

	// Step 3: Look for builders_hut_hire_button
	log.Println("Step 3: Looking for builders_hut_hire_button...")
	hireConfirmDetections, err := vision.CaptureAndDetect(deviceID, adbPath)
	if err != nil {
		log.Printf("Error getting detections for builders_hut_hire_button: %v", err)
		resetView(deviceID, adbPath)
		return false
	}

	// Log what we see for debugging
	log.Printf("Detected %d objects in hire dialog:", len(hireConfirmDetections))
	for i, det := range hireConfirmDetections {
		if det.Confidence > common.MinConfidence {
			log.Printf("  %d. %s (%.2f): (%.1f, %.1f) %.0fx%.0f",
				i+1, det.Class, det.Confidence, det.X, det.Y, det.Width, det.Height)
		}
	}

	var hireButton *common.Detection
	for _, det := range hireConfirmDetections {
		if det.Class == "builders_hut_hire_button" && det.Confidence > common.MinConfidence {
			hireButton = &det
			break
		}
	}

	if hireButton == nil {
		log.Println("builders_hut_hire_button not found, looking for alternative buttons...")
		// Try alternative buttons like "confirm_button" or general buttons
		for _, det := range hireConfirmDetections {
			if (det.Class == "confirm_button" || det.Class == "button") &&
				det.Y > 350 && det.Confidence > common.MinConfidence {
				hireButton = &det
				log.Printf("Found alternative hire button: %s at (%.1f, %.1f)",
					det.Class, det.X, det.Y)
				break
			}
		}

		if hireButton == nil {
			log.Println("No suitable hire button found, resetting view")
			resetView(deviceID, adbPath)
			return false
		}
	}

	// Click on hire button
	log.Printf("Clicking on hire button at (%.1f, %.1f)", hireButton.X, hireButton.Y)
	if err := utils.TapScreen(deviceID, adbPath, int(hireButton.X), int(hireButton.Y)); err != nil {
		log.Printf("Error clicking on hire button: %v", err)
		resetView(deviceID, adbPath)
		return false
	}

	// Wait longer for confirmation dialog and any help bubbles to show
	log.Println("Waiting for confirmation dialog to fully appear...")
	time.Sleep(3 * time.Second)

	// Step 4: Look for exit_dialog_button
	log.Println("Step 4: Looking for exit_dialog_button...")
	successDetections, err := vision.CaptureAndDetect(deviceID, adbPath)
	if err != nil {
		log.Printf("Error getting detections for success dialog: %v", err)
		resetView(deviceID, adbPath)
		return false
	}

	// Log what we see for debugging
	log.Printf("Detected %d objects in success/confirmation dialog:", len(successDetections))
	for i, det := range successDetections {
		if det.Confidence > common.MinConfidence {
			log.Printf("  %d. %s (%.2f): (%.1f, %.1f) %.0fx%.0f",
				i+1, det.Class, det.Confidence, det.X, det.Y, det.Width, det.Height)
		}
	}

	// Check for exit buttons or success indicators
	var exitButton *common.Detection
	var successDialog *common.Detection

	for _, det := range successDetections {
		if det.Class == "exit_dialog_button" && det.Confidence > common.MinConfidence {
			exitButton = &det
		}
		if det.Class == "builders_hut_hire_success" && det.Confidence > common.MinConfidence {
			successDialog = &det
		}
	}

	// If we found the success dialog, we know it worked
	if successDialog != nil {
		log.Println("Found builders_hut_hire_success - recruitment successful")
	}

	// If we found an exit button, click it
	if exitButton != nil {
		log.Printf("Clicking exit_dialog_button at (%.1f, %.1f)", exitButton.X, exitButton.Y)
		if err := utils.TapScreen(deviceID, adbPath, int(exitButton.X), int(exitButton.Y)); err != nil {
			log.Printf("Error clicking exit_dialog_button: %v", err)
		} else {
			log.Println("Successfully clicked exit button")
		}
	} else {
		log.Println("No exit button found, clicking center of screen")
		utils.TapScreen(deviceID, adbPath, 320, 240)
	}

	// Wait a moment after dismissing dialogs
	time.Sleep(1 * time.Second)

	// Mark as complete and update state
	instanceState.SecondBuilderAdded = true
	secondBuilderAdded = true                                                            // Also update the global tracking variable
	instanceState.BuilderState.SecondBuilderEndTime = time.Now().Add(3 * 24 * time.Hour) // 3 days from now
	log.Println("Second builder successfully recruited!")

	// Reset view to ensure we're back in a known state
	log.Println("Resetting view after successful recruitment")
	resetView(deviceID, adbPath)
	return true
}

// IsSecondBuilderAdded checks if the second builder has been added
func IsSecondBuilderAdded() bool {
	return secondBuilderAdded
}
