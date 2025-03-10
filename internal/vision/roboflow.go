package vision

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"roborok/internal/common"
	"time"
)

// RoboflowResponse represents the response from Roboflow API
type RoboflowResponse struct {
	Predictions []struct {
		X          float64 `json:"x"`
		Y          float64 `json:"y"`
		Width      float64 `json:"width"`
		Height     float64 `json:"height"`
		Confidence float64 `json:"confidence"`
		Class      string  `json:"class"`
	} `json:"predictions"`
	Time  float64 `json:"time"`
	Image struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"image"`
}

// SendToRoboflow sends an image to Roboflow for inference
func SendToRoboflow(imageBytes []byte, apiKey, modelID string) (*RoboflowResponse, error) {
	// Add detailed logging before the API call
	// log.Printf("Sending request to Roboflow API - Model ID: %s", modelID)

	// Safely log API key (just first few chars)
	if len(apiKey) > 4 {
		// log.Printf("API Key (first 4 chars): %s...", apiKey[:4]) // Only log first 4 chars for security
	} else {
		log.Printf("API Key: [TOO SHORT - POSSIBLE ERROR]")
	}

	url := fmt.Sprintf("https://detect.roboflow.com/%s?api_key=%s", modelID, apiKey)

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add the file
	part, err := writer.CreateFormFile("file", "screenshot.png")
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	_, err = io.Copy(part, bytes.NewReader(imageBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to copy image data: %w", err)
	}

	writer.Close()

	// Create and send the request
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check for success with more detailed error reporting
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("Roboflow API error - Status: %d, Response: %s", resp.StatusCode, string(bodyBytes))

		// Try to parse the error message for more details
		var errorResponse struct {
			Message string `json:"message"`
			Detail  string `json:"detail,omitempty"`
		}

		if err := json.Unmarshal(bodyBytes, &errorResponse); err == nil && errorResponse.Detail != "" {
			return nil, fmt.Errorf("API error (status %d): %s - %s",
				resp.StatusCode, errorResponse.Message, errorResponse.Detail)
		}

		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse the response
	var result RoboflowResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Log detailed information about detected objects
	if len(result.Predictions) > 0 {
		log.Printf("Roboflow API request successful - Detected %d objects:", len(result.Predictions))
		for i, pred := range result.Predictions {
			log.Printf("  %d. %s (%.2f): (%.1f, %.1f) %.0fx%.0f",
				i+1, pred.Class, pred.Confidence, pred.X, pred.Y, pred.Width, pred.Height)
		}
	} else {
		log.Printf("Roboflow API request successful - No objects detected")
	}

	return &result, nil
}

// AnalyzeGameState analyzes the current game state and view
func AnalyzeGameState(
	screenshot []byte,
	apiKey string,
	modelID string,
) (string, []common.Detection, error) {
	// Send screenshot to Roboflow
	resp, err := SendToRoboflow(screenshot, apiKey, modelID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to analyze screenshot: %w", err)
	}

	// Convert response to detections
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

	// Determine the view (city or map)
	gameView := DetermineGameView(detections)

	return gameView, detections, nil
}

// DetermineGameView determines if we're in city view, map view, or unknown view
func DetermineGameView(detections []common.Detection) string {
	// First, check explicit view indicators
	for _, detection := range detections {
		if detection.Class == "on_field" && detection.Confidence > common.MinConfidence {
			return "field"
		}

		if detection.Class == "in_city" && detection.Confidence > common.MinConfidence {
			return "city"
		}
	}

	// If no explicit indicator, check for view-specific elements
	cityIndicators := 0
	mapIndicators := 0

	for _, detection := range detections {
		// City view indicators
		if detection.Class == "city_hall" ||
			detection.Class == "city_hall_upgradeable" ||
			detection.Class == "barracks" ||
			detection.Class == "barracks_upgradeable" ||
			detection.Class == "barracks_upgradeable_idle" ||
			detection.Class == "farm" ||
			detection.Class == "builders_hut" ||
			detection.Class == "builders_hut_idle" ||
			detection.Class == "tavern" ||
			detection.Class == "tavern_upgradeable_clickable" {
			cityIndicators++
		}

		// Map view indicators
		if detection.Class == "return_to_city_button" ||
			detection.Class == "world_map" ||
			detection.Class == "barbarian" ||
			detection.Class == "resource_node" {
			mapIndicators++
		}
	}

	// Determine view based on the count of indicators
	if cityIndicators > 0 && cityIndicators > mapIndicators {
		return "city"
	} else if mapIndicators > 0 {
		return "map"
	}

	// Default to unknown if we can't determine
	return "city"
}
