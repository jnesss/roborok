package actions

import (
	"log"
	"math/rand"
	"roborok/internal/common"
	"roborok/internal/state"
	"roborok/internal/utils"
	"roborok/internal/vision"
	"strings"
	"time"
)

var civilizationScrollAttempts = 0

// IsTutorialComplete checks if the tutorial has been completed
// It simply checks if we've completed both required steps in the sequence
func IsTutorialComplete(deviceID, adbPath string, instanceState *state.InstanceState) (bool, error) {
	// If we've already marked tutorial as completed, don't re-check
	if instanceState.TutorialCompleted {
		return true, nil
	}

	// Check if we've tracked both steps in the completion sequence
	if instanceState.TutorialUpgradeCompleteClicked &&
		instanceState.TutorialFinalArrowClicked {
		// Tutorial is complete if both steps have been done
		instanceState.TutorialCompleted = true
		return true, nil
	}

	// Tutorial is not complete yet
	return false, nil
}

// RunTutorialAutomation runs the tutorial automation with state tracking
func RunTutorialAutomation(
	deviceID string,
	roboflowAPIKey string,
	roboflowModelID string,
	adbPath string,
	preferredCivilization string,
	instanceState *state.InstanceState,
) bool {
	log.Printf("Starting tutorial automation for device %s", deviceID)
	log.Printf("Using civilization: %s", preferredCivilization)

	// Use the provided API key or fall back to default
	if roboflowAPIKey == "" {
		// Load config to get API key
		config, err := utils.LoadConfig("config.json")
		if err == nil {
			roboflowAPIKey = config.Global.RoboflowAPIKey
		}
	}

	// Use the provided model ID or fall back to tutorial model
	if roboflowModelID == "" {
		// Load config to get model ID
		config, err := utils.LoadConfig("config.json")
		if err == nil {
			roboflowModelID = config.Global.RoboflowTutorialModel
		} else {
			roboflowModelID = common.TutorialModelID
		}
	}

	// Initialize random seed
	rand.Seed(time.Now().UnixNano())

	// Tutorial timeout (10 minutes should be more than enough for the tutorial)
	tutorialTimeout := time.Now().Add(10 * time.Minute)

	// Counters for tracking progress and detecting stuck states
	iterationCount := 0
	stuckIterationCount := 0
	lastState := StateUnknown
	stuckStateCount := 0

	// If we're in the same state for too many iterations, we might be stuck
	const maxStuckIterations = 20

	// Main tutorial automation loop - run until timeout or completion
	for time.Now().Before(tutorialTimeout) {
		iterationCount++

		// Every 50 iterations, check if tutorial is complete
		if iterationCount%50 == 0 {
			isComplete, err := IsTutorialComplete(deviceID, adbPath, instanceState)
			if err != nil {
				log.Printf("Error checking tutorial completion: %v", err)
			} else if isComplete {
				log.Println("Tutorial completed!")
				return true
			}
		}

		// Capture screenshot
		screenshot, err := vision.CaptureScreenshot(deviceID, adbPath)
		if err != nil {
			log.Printf("Error capturing screenshot: %v", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		// Send to Roboflow for analysis
		resp, err := vision.SendToRoboflow(screenshot, roboflowAPIKey, roboflowModelID)
		if err != nil {
			log.Printf("Error sending to Roboflow: %v", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		// Convert to common.Detection format
		var detections []common.Detection
		for _, pred := range resp.Predictions {
			detections = append(detections, common.Detection{
				Class:      pred.Class,
				X:          pred.X,
				Y:          pred.Y,
				Width:      pred.Width,
				Height:     pred.Height,
				Confidence: pred.Confidence,
			})
		}

		// Log occasional detection information
		if len(detections) > 0 {
			log.Printf("Detected %d objects:", len(detections))
			for i, det := range detections {
				log.Printf("  %d. %s (%.2f): (%.1f, %.1f) %.0fx%.0f",
					i+1, det.Class, det.Confidence, det.X, det.Y, det.Width, det.Height)
			}
		}

		// Determine the tutorial state
		tutorialState := determineTutorialState(detections, preferredCivilization, instanceState)

		// Check if we're stuck in the same state
		if tutorialState == lastState {
			stuckStateCount++
		} else {
			stuckStateCount = 0
			lastState = tutorialState
		}

		// If we're stuck in the same state for too long, try a random tap
		if stuckStateCount > maxStuckIterations {
			log.Printf("Stuck in state %s for %d iterations, trying to unstick...",
				tutorialState, stuckStateCount)

			// Try a random tap in the center area
			centerX := 200 + rand.Intn(200) // 200-400
			centerY := 200 + rand.Intn(200) // 200-400
			utils.TapScreen(deviceID, adbPath, centerX, centerY)

			// Reset stuck counter
			stuckStateCount = 0
			time.Sleep(1 * time.Second)
			continue
		}

		if tutorialState != StateUnknown {
			log.Printf("Tutorial state: %s", tutorialState)
		}

		// Handle the current state
		actionTaken := handleTutorialState(
			deviceID,
			adbPath,
			detections,
			tutorialState,
			preferredCivilization,
			instanceState,
		)

		if !actionTaken {
			stuckIterationCount++

			// If no action was taken for many iterations, try a different approach
			if stuckIterationCount > 30 {
				log.Println("No action taken for many iterations, checking for tutorial completion")

				// Check if tutorial is actually complete
				isComplete, _ := IsTutorialComplete(deviceID, adbPath, instanceState)
				if isComplete {
					log.Println("Tutorial was already completed!")
					return true
				}

				// Try tapping center of screen to dismiss any dialogs
				utils.TapScreen(deviceID, adbPath, 240, 400)
				stuckIterationCount = 0
				time.Sleep(1 * time.Second)
			} else {
				// Only sleep if no action was taken
				time.Sleep(500 * time.Millisecond)
			}
		} else {
			// Reset stuck counter when action is taken
			stuckIterationCount = 0
		}

		// Check if we've completed both necessary steps in the sequence
		if instanceState.TutorialUpgradeCompleteClicked &&
			instanceState.TutorialFinalArrowClicked {
			log.Println("Detected complete tutorial sequence (upgrade complete + final arrow)!")
			instanceState.TutorialCompleted = true
			return true
		}
	}

	log.Println("Tutorial automation timed out")
	return false
}

// TutorialState represents the current state of the tutorial
type TutorialState string

const (
	StateUnknown            TutorialState = "unknown"
	StateSkipButton         TutorialState = "skip_button"
	StateCounselorText      TutorialState = "counselor_text"
	StateCivilizationSelect TutorialState = "civilization_select"
	StateConfirmButton      TutorialState = "confirm_button"
	StateArrowAndTarget     TutorialState = "arrow_and_target"
	StateArrowOnly          TutorialState = "arrow_only"
	StateUpgradeComplete    TutorialState = "upgrade_complete"
	StateFinalArrow         TutorialState = "final_arrow"
)

// determineTutorialState analyzes detections to determine the current state
// with awareness of our position in the tutorial completion sequence
func determineTutorialState(
	detections []common.Detection,
	preferredCivilization string,
	instanceState *state.InstanceState,
) TutorialState {
	// If we've already clicked upgrade_complete but not the final arrow,
	// prioritize looking for ANY click_arrow + click_target combination
	if instanceState.TutorialUpgradeCompleteClicked && !instanceState.TutorialFinalArrowClicked {
		// Look for any arrow and target combination
		hasArrow := false
		hasTarget := false

		for _, detection := range detections {
			if detection.Class == "click_arrow" && detection.Confidence > common.MinConfidence {
				hasArrow = true
			}
			if detection.Class == "click_target" && detection.Confidence > common.MinConfidence {
				hasTarget = true
			}
		}

		if hasArrow && hasTarget {
			return StateFinalArrow
		}
	}

	// If we haven't yet clicked upgrade_complete, prioritize finding it
	if !instanceState.TutorialUpgradeCompleteClicked {
		for _, detection := range detections {
			if detection.Class == "upgrade_complete" && detection.Confidence > common.MinConfidence {
				return StateUpgradeComplete
			}
		}
	}

	// Standard tutorial state detection follows below

	// Check for skip button
	for _, detection := range detections {
		if detection.Class == "skip button" && detection.Confidence > common.MinConfidence {
			return StateSkipButton
		}
	}

	// Check for counselor text
	for _, detection := range detections {
		if detection.Class == "counselor text bubble" && detection.Confidence > common.MinConfidence {
			return StateCounselorText
		}
	}

	// Check for civilization selection
	// Look for civilizations to determine if we're on that screen
	civCount := 0
	for _, detection := range detections {
		if isCivilization(detection.Class) {
			civCount++
		}
	}

	// If we see multiple civilizations, we're likely on the selection screen
	if civCount >= 3 {
		return StateCivilizationSelect
	}

	// Check for confirm button
	for _, detection := range detections {
		if detection.Class == "confirm_button" && detection.Confidence > common.MinConfidence {
			return StateConfirmButton
		}
	}

	// Check for both arrow and target
	hasArrow := false
	hasTarget := false

	for _, detection := range detections {
		if detection.Class == "click_arrow" && detection.Confidence > common.MinConfidence {
			hasArrow = true
		}
		if detection.Class == "click_target" && detection.Confidence > common.MinConfidence {
			hasTarget = true
		}
	}

	if hasArrow && hasTarget {
		return StateArrowAndTarget
	}

	if hasArrow {
		return StateArrowOnly
	}

	return StateUnknown
}

// handleTutorialState takes action based on the current state
// and tracks our progress through the tutorial completion sequence
func handleTutorialState(
	deviceID, adbPath string,
	detections []common.Detection,
	state TutorialState,
	preferredCivilization string,
	instanceState *state.InstanceState,
) bool {
	switch state {
	case StateUpgradeComplete:
		if handled := handleUpgradeComplete(deviceID, adbPath, detections, instanceState); handled {
			// Mark that we've clicked on upgrade complete
			instanceState.TutorialUpgradeCompleteClicked = true
			log.Println("Marked 'upgrade_complete' as clicked - looking for final arrow next")
			return true
		}
		return false

	case StateFinalArrow:
		if handled := handleFinalArrow(deviceID, adbPath, detections, instanceState); handled {
			// Mark that we've clicked on the final arrow
			instanceState.TutorialFinalArrowClicked = true
			log.Println("Marked final arrow as clicked - tutorial sequence complete!")
			instanceState.TutorialCompleted = true
			return true
		}
		return false

	case StateSkipButton:
		return handleSkipButton(deviceID, adbPath, detections)

	case StateCounselorText:
		return handleCounselorText(deviceID, adbPath, detections)

	case StateCivilizationSelect:
		return handleCivilizationSelection(deviceID, adbPath, detections, preferredCivilization)

	case StateConfirmButton:
		return handleConfirmButton(deviceID, adbPath, detections)

	case StateArrowAndTarget:
		return handleArrowAndTarget(deviceID, adbPath, detections)

	case StateArrowOnly:
		return handleArrowOnly(deviceID, adbPath, detections)

	default:
		// Don't log anything for unknown state to reduce noise
		return false
	}
}

// Individual handlers for each state

func handleSkipButton(deviceID, adbPath string, detections []common.Detection) bool {
	for _, detection := range detections {
		if detection.Class == "skip button" && detection.Confidence > common.MinConfidence {
			log.Println("Found skip button - clicking...")
			if err := utils.TapScreen(deviceID, adbPath, int(detection.X), int(detection.Y)); err != nil {
				log.Printf("Failed to tap skip button: %v", err)
				return false
			}
			return true
		}
	}
	return false
}

func handleCounselorText(deviceID, adbPath string, detections []common.Detection) bool {
	for _, detection := range detections {
		if detection.Class == "counselor text bubble" && detection.Confidence > common.MinConfidence {
			log.Println("Found counselor text - clicking...")
			if err := utils.TapScreen(deviceID, adbPath, int(detection.X), int(detection.Y)); err != nil {
				log.Printf("Failed to tap counselor text: %v", err)
				return false
			}
			return true
		}
	}
	return false
}

// Updated function with no fallback
func handleCivilizationSelection(deviceID, adbPath string, detections []common.Detection, preferredCivilization string) bool {
	// Check if a civilization is already selected
	// Check if our preferred civilization is selected
	selectedCivClass := strings.ToLower(preferredCivilization) + "_selected"
	for _, detection := range detections {
		if strings.ToLower(detection.Class) == selectedCivClass && detection.Confidence > common.MinConfidence {
			log.Printf("Found %s - preferred civilization selected", detection.Class)

			// Find and click the confirm button
			for _, btn := range detections {
				if btn.Class == "confirm_button" && btn.Confidence > common.MinConfidence {
					log.Println("Found confirm button - clicking...")
					if err := utils.TapScreen(deviceID, adbPath, int(btn.X), int(btn.Y)); err != nil {
						log.Printf("Failed to tap confirm button: %v", err)
						return false
					}
					// Wait for confirmation
					time.Sleep(1 * time.Second)
					return true
				}
			}
			return false
		}
	}

	// Count civilizations
	detectedCivs := 0
	var civDetections []common.Detection

	for _, detection := range detections {
		if isCivilization(detection.Class) {
			detectedCivs++
			civDetections = append(civDetections, detection)
		}
	}

	log.Printf("Counted %d civilizations: ", detectedCivs)
	for _, civ := range civDetections {
		log.Printf("  - %s (confidence: %.2f)", civ.Class, civ.Confidence)
	}

	// Make sure we have enough civilizations visible
	expectedMinCivs := 6
	if detectedCivs < expectedMinCivs {
		log.Printf("Only detected %d civilizations, waiting for better view (expecting at least %d)",
			detectedCivs, expectedMinCivs)
		return false
	}

	// Look for the preferred civilization
	for _, detection := range civDetections {
		if strings.ToLower(detection.Class) == strings.ToLower(preferredCivilization) && detection.Confidence > 0.5 {
			log.Printf("Found %s (confidence: %.2f) - clicking...", preferredCivilization, detection.Confidence)
			if err := utils.TapScreen(deviceID, adbPath, int(detection.X), int(detection.Y)); err != nil {
				log.Printf("Failed to tap %s: %v", preferredCivilization, err)
				return false
			}
			// Wait for selection to take effect
			time.Sleep(1 * time.Second)
			// Reset scroll attempts on success
			civilizationScrollAttempts = 0
			return true
		}
	}

	// If preferred civilization not found, try to scroll right
	maxScrollAttempts := 5 // Increased from 3 to give more chances to find

	// After reaching max scroll attempts, reset and start over
	// This ensures we can circle through all civilizations
	if civilizationScrollAttempts >= maxScrollAttempts {
		log.Printf("Reached maximum scroll attempts (%d), resetting to try again", maxScrollAttempts)
		civilizationScrollAttempts = 0
		time.Sleep(1 * time.Second)
		return false
	}

	log.Printf("Preferred civilization '%s' not found, scrolling right (attempt %d/%d)",
		preferredCivilization, civilizationScrollAttempts+1, maxScrollAttempts)

	// Find rightmost and leftmost civilizations
	var rightmost, leftmost *common.Detection
	var rightmostX, leftmostX float64 = 0, 9999

	for _, civ := range civDetections {
		if civ.X > rightmostX {
			rightmostX = civ.X
			rightmost = &civ
		}
		if civ.X < leftmostX {
			leftmostX = civ.X
			leftmost = &civ
		}
	}

	if rightmost != nil && leftmost != nil {
		// Perform swipe from rightmost to leftmost civilization
		startX := int(rightmost.X)
		startY := int(rightmost.Y)
		endX := int(leftmost.X)
		endY := int(leftmost.Y)

		log.Printf("Swiping from (%d,%d) to (%d,%d)", startX, startY, endX, endY)
		if err := utils.SwipeScreen(deviceID, adbPath, startX, startY, endX, endY, 300); err != nil {
			log.Printf("Failed to swipe: %v", err)
			return false
		} else {
			civilizationScrollAttempts++
			// Wait after scrolling
			time.Sleep(1 * time.Second)
			return true
		}
	}

	log.Printf("Could not find suitable points to scroll. Still looking for '%s'...", preferredCivilization)
	return false
}

func handleConfirmButton(deviceID, adbPath string, detections []common.Detection) bool {
	for _, detection := range detections {
		if detection.Class == "confirm_button" && detection.Confidence > common.MinConfidence {
			log.Println("Found confirm button - clicking...")
			if err := utils.TapScreen(deviceID, adbPath, int(detection.X), int(detection.Y)); err != nil {
				log.Printf("Failed to tap confirm button: %v", err)
				return false
			}
			return true
		}
	}
	return false
}

func handleArrowAndTarget(deviceID, adbPath string, detections []common.Detection) bool {
	var target *common.Detection

	for _, detection := range detections {
		if detection.Class == "click_target" && detection.Confidence > common.MinConfidence {
			target = &detection
			break
		}
	}

	if target != nil {
		log.Println("Found arrow and target - clicking target...")
		if err := utils.TapScreen(deviceID, adbPath, int(target.X), int(target.Y)); err != nil {
			log.Printf("Failed to tap target: %v", err)
			return false
		}
		return true
	}

	return false
}

func handleArrowOnly(deviceID, adbPath string, detections []common.Detection) bool {
	var arrow *common.Detection

	for _, detection := range detections {
		if detection.Class == "click_arrow" && detection.Confidence > common.MinConfidence {
			arrow = &detection
			break
		}
	}

	if arrow != nil {
		// could look at arrow direction and attempt a click 100px in that direction..
	}

	return false
}

// isCivilization checks if a class name is a civilization
func isCivilization(className string) bool {
	// Use ALL civilization names from the roboflow model
	civs := []string{
		"arabia", "britain", "china", "egypt", "france",
		"germany", "greece", "japan", "korea", "maya",
		"rome", "spain", "vikings",
	}
	for _, civ := range civs {
		if className == civ {
			return true
		}
	}

	return false
}

// Handler for upgrade complete notification
func handleUpgradeComplete(deviceID, adbPath string, detections []common.Detection, instanceState *state.InstanceState) bool {
	for _, detection := range detections {
		if detection.Class == "upgrade_complete" && detection.Confidence > common.MinConfidence {
			// Calculate position to click - just outside the bottom edge
			xPos := int(detection.X)                               // Center horizontally
			yPos := int(detection.Y + (detection.Height / 2) + 20) // 20px below the bottom edge

			log.Printf("Found 'upgrade_complete' notification - clicking just below bottom edge at (%d, %d)", xPos, yPos)

			if err := utils.TapScreen(deviceID, adbPath, xPos, yPos); err != nil {
				log.Printf("Failed to tap upgrade_complete: %v", err)
				return false
			}

			// Mark as clicked
			instanceState.TutorialUpgradeCompleteClicked = true
			log.Println("Marked 'upgrade_complete' as clicked - looking for final arrow next")

			// Wait for UI to update
			time.Sleep(1 * time.Second)
			return true
		}
	}
	return false
}

// Handler for the final arrow after upgrade_complete
func handleFinalArrow(deviceID, adbPath string, detections []common.Detection, instanceState *state.InstanceState) bool {
	// Find the best target to click (highest confidence)
	var bestTarget *common.Detection
	var bestConfidence float64

	// Look through all click_targets
	for _, detection := range detections {
		if detection.Class == "click_target" && detection.Confidence > common.MinConfidence {
			if detection.Confidence > bestConfidence {
				bestTarget = &detection
				bestConfidence = detection.Confidence
			}
		}
	}

	// If we found a target, click it
	if bestTarget != nil {
		log.Println("Found final arrow/target - clicking to complete tutorial")
		if err := utils.TapScreen(deviceID, adbPath, int(bestTarget.X), int(bestTarget.Y)); err != nil {
			log.Printf("Failed to tap final target: %v", err)
			return false
		}
		// Wait for the tutorial to fully complete
		time.Sleep(1 * time.Second)
		return true
	}

	return false
}
