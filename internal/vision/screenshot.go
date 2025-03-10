package vision

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"roborok/internal/common"
	"roborok/internal/utils"
	"time"
)

// CaptureScreenshot captures a screenshot from the device
func CaptureScreenshot(deviceID, adbPath string) ([]byte, error) {
	cmd := exec.Command(adbPath, "-s", deviceID, "exec-out", "screencap", "-p")
	return cmd.Output()
}

// SaveScreenshot saves a screenshot to disk
func SaveScreenshot(screenshot []byte, path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write file
	if err := os.WriteFile(path, screenshot, 0644); err != nil {
		return fmt.Errorf("failed to write screenshot file: %w", err)
	}

	return nil
}

// CaptureAndDetect is a helper function that captures a screenshot,
// sends it to Roboflow for analysis, and returns the detections.
// It handles all the error cases and returns a clean detections array.
// CaptureAndDetect is a helper function that captures a screenshot,
// sends it to Roboflow for analysis, dismisses any help bubbles, and returns the clean detections.
func CaptureAndDetect(
	deviceID string,
	adbPath string,
) ([]common.Detection, error) {
	// Get API key and model ID from global config
	apiKey := utils.GetRoboflowAPIKey()
	modelID := utils.GetRoboflowGameplayModel()

	// Maximum attempts to dismiss help bubbles
	const maxAttempts = 5

	// Attempt capture and detect
	for attempts := 0; attempts < maxAttempts; attempts++ {
		// Capture screenshot
		screenshot, err := CaptureScreenshot(deviceID, adbPath)
		if err != nil {
			return nil, fmt.Errorf("error capturing screenshot: %w", err)
		}

		// Send to Roboflow
		resp, err := SendToRoboflow(screenshot, apiKey, modelID)
		if err != nil {
			return nil, fmt.Errorf("error sending to Roboflow: %w", err)
		}

		// Convert response to common.Detection format
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

		// Log the results for debugging
		if len(detections) > 0 {
			log.Printf("Detected %d objects:", len(detections))
			for i, det := range detections {
				if det.Confidence > common.MinConfidence {
					log.Printf("  %d. %s (%.2f): (%.1f, %.1f) %.0fx%.0f",
						i+1, det.Class, det.Confidence, det.X, det.Y, det.Width, det.Height)
				}
			}
		} else {
			log.Printf("No objects detected")
		}

		// Check for help bubbles
		helpBubbleFound := false
		for _, det := range detections {
			if (det.Class == "help_chat_bubble" || det.Class == "help_bubble") &&
				det.Confidence > common.MinConfidence {
				// Found a help bubble, click it to dismiss
				log.Printf("Found %s, dismissing popup at (%.1f, %.1f)...",
					det.Class, det.X, det.Y)

				if err := utils.TapScreen(deviceID, adbPath, int(det.X), int(det.Y)); err != nil {
					log.Printf("Error dismissing help bubble: %v", err)
				} else {
					helpBubbleFound = true
					log.Printf("Help bubble dismissed (attempt %d/%d)", attempts+1, maxAttempts)
					// Wait for bubble animation and any subsequent bubbles to appear
					time.Sleep(1 * time.Second)
				}
				break
			}
		}

		// If we found and dismissed a help bubble, try again
		if helpBubbleFound {
			continue
		}

		// No help bubbles found, return the detections
		return detections, nil
	}

	// If we got here, we've reached the maximum attempts
	log.Printf("Maximum help bubble dismissal attempts (%d) reached, continuing anyway", maxAttempts)

	// Try one more time to get clean detections
	screenshot, err := CaptureScreenshot(deviceID, adbPath)
	if err != nil {
		return nil, fmt.Errorf("error capturing final screenshot: %w", err)
	}

	resp, err := SendToRoboflow(screenshot, apiKey, modelID)
	if err != nil {
		return nil, fmt.Errorf("error sending final screenshot to Roboflow: %w", err)
	}

	var finalDetections []common.Detection
	for _, pred := range resp.Predictions {
		finalDetections = append(finalDetections, common.Detection{
			Class:      pred.Class,
			X:          pred.X,
			Y:          pred.Y,
			Width:      pred.Width,
			Height:     pred.Height,
			Confidence: pred.Confidence,
		})
	}

	return finalDetections, nil
}

// FindDetectionByClass finds a detection with the specified class name
// above the minimum confidence threshold
func FindDetectionByClass(
	detections []common.Detection,
	className string,
	minConfidence float64,
) *common.Detection {
	for _, det := range detections {
		if det.Class == className && det.Confidence > minConfidence {
			return &det
		}
	}
	return nil
}

// FindDetectionsByClass finds all detections with the specified class name
// above the minimum confidence threshold
func FindDetectionsByClass(
	detections []common.Detection,
	className string,
	minConfidence float64,
) []common.Detection {
	var result []common.Detection
	for _, det := range detections {
		if det.Class == className && det.Confidence > minConfidence {
			result = append(result, det)
		}
	}
	return result
}

// FindDetectionByClasses finds a detection with any of the specified class names
// above the minimum confidence threshold
func FindDetectionByClasses(
	detections []common.Detection,
	classNames []string,
	minConfidence float64,
) *common.Detection {
	for _, det := range detections {
		for _, className := range classNames {
			if det.Class == className && det.Confidence > minConfidence {
				return &det
			}
		}
	}
	return nil
}
