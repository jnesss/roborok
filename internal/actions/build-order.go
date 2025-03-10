package actions

import (
	"log"
	"roborok/internal/common"
	"roborok/internal/state"
	"roborok/internal/utils"
	"roborok/internal/vision"
	"strings"
	"time"
)

// DetectionClassesByBuilding maps building names to their detection classes
var DetectionClassesByBuilding = map[string]string{
	"cityhall":      "cityhall",
	"wall":          "wall",
	"academy":       "academy",
	"barracks":      "barracks",
	"archery_range": "archery_range",
	"stable":        "stable",
	"hospital":      "hospital",
	"farm":          "farm",
	"lumber_mill":   "lumber_mill",
	"quarry":        "quarry",
	"goldmine":      "goldmine",
	"storehouse":    "storehouse",
	"scout_camp":    "scout_camp",
	"trading_post":  "trading_post",
	"tavern":        "tavern",
	// Add other buildings as needed
}

// DefineDefaultBuildOrder returns a simplified build order
func DefineDefaultBuildOrder() []state.BuildTask {
	return []state.BuildTask{
		{Type: "upgrade", Building: "cityhall", DetectClass: "cityhall"}, // lvl 3

		// Requires Lvl 2 City Hall:
		{Type: "build_new", Building: "archery_range", DetectClass: "military:build_archery_range"}, // lvl 1
		{Type: "upgrade", Building: "barracks", DetectClass: "barracks"},                            // lvl 2
		{Type: "upgrade", Building: "scout_camp", DetectClass: "scout_camp"},                        // lvl 2
		{Type: "upgrade", Building: "farm", DetectClass: "farm"},                                    // lvl 2
		{Type: "upgrade", Building: "farm", DetectClass: "farm"},                                    // lvl 3
		{Type: "upgrade", Building: "tavern", DetectClass: "tavern"},                                // lvl 2
		{Type: "upgrade", Building: "hospital", DetectClass: "hospital"},                            // lvl 2
		{Type: "build_new", Building: "lumber_mill", DetectClass: "economic:build_lumber_mill"},
		{Type: "upgrade", Building: "lumber_mill", DetectClass: "lumber_mill"}, // lvl 2
		{Type: "upgrade", Building: "lumber_mill", DetectClass: "lumber_mill"}, // lvl 3

		// Got City Hall 3
		{Type: "upgrade", Building: "wall", DetectClass: "wall"},                      // lvl 3
		{Type: "upgrade", Building: "archery_range", DetectClass: "archery_range"},    // lvl 2
		{Type: "upgrade", Building: "cityhall", DetectClass: "cityhall"},              // lvl 4
		{Type: "upgrade", Building: "barracks", DetectClass: "barracks"},              // lvl 3
		{Type: "build_new", Building: "stable", DetectClass: "military:build_stable"}, // lvl 1
		{Type: "upgrade", Building: "scout_camp", DetectClass: "scout_camp"},          // lvl 3
		{Type: "upgrade", Building: "hospital", DetectClass: "hospital"},              // lvl 3
		{Type: "upgrade", Building: "lumber_mill", DetectClass: "lumber_mill"},        // lvl 4
		{Type: "upgrade", Building: "tavern", DetectClass: "tavern"},                  // lvl 3
		{Type: "upgrade", Building: "storehouse", DetectClass: "storehouse"},          // lvl 2
		{Type: "upgrade", Building: "storehouse", DetectClass: "storehouse"},          // lvl 3

		{Type: "build_new", Building: "siege_workshop", DetectClass: "military:build_siege_workshop"}, // lvl 1
		{Type: "upgrade", Building: "stable", DetectClass: "stable"},                                  // lvl 2
		{Type: "upgrade", Building: "stable", DetectClass: "stable"},                                  // lvl 3
		{Type: "upgrade", Building: "siege_workshop", DetectClass: "siege_workshop"},                  // lvl 2
		{Type: "upgrade", Building: "hospital", DetectClass: "hospital"},                              // lvl 4
		{Type: "upgrade", Building: "wall", DetectClass: "wall"},                                      // lvl 4
		{Type: "upgrade", Building: "cityhall", DetectClass: "cityhall"},                              // lvl 5

		{Type: "upgrade", Building: "archery_range", DetectClass: "archery_range"}, // lvl 3
		{Type: "upgrade", Building: "archery_range", DetectClass: "archery_range"}, // lvl 4
		{Type: "upgrade", Building: "barracks", DetectClass: "barracks"},           // lvl 4
		{Type: "upgrade", Building: "farm", DetectClass: "farm"},                   // lvl 4

		{Type: "build_new", Building: "alliance_center", DetectClass: "economic:build_alliance_center"},

		//{Type: "upgrade", Building: "stable", DetectClass: "alliance_center"},                // lvl 2
		//{Type: "upgrade", Building: "stable", DetectClass: "alliance_center"},                // lvl 2
		{Type: "upgrade", Building: "barracks", DetectClass: "barracks"},     // lvl 3
		{Type: "upgrade", Building: "scout_camp", DetectClass: "scout_camp"}, // lvl 3
		{Type: "upgrade", Building: "farm", DetectClass: "farm"},             // lvl 3
		{Type: "upgrade", Building: "tavern", DetectClass: "tavern"},         // lvl 3
		{Type: "upgrade", Building: "hospital", DetectClass: "hospital"},     // lvl 3

		// Got City Hall 4?
		{Type: "upgrade", Building: "alliance_center", DetectClass: "alliance_center"}, // lvl 3
		{Type: "upgrade", Building: "cityhall", DetectClass: "cityhall"},               // lvl 4
		{Type: "upgrade", Building: "scout_camp", DetectClass: "scout_camp"},           // lvl 3

		// Requires Lvl 4 City Hall:
		{Type: "build_new", Building: "academy", DetectClass: "economic:build_academy"},
		{Type: "upgrade", Building: "academy", DetectClass: "academy"},
		{Type: "upgrade", Building: "academy", DetectClass: "academy"},

		{Type: "build_new", Building: "stable", DetectClass: "military:build_stable"},
		{Type: "upgrade", Building: "stable", DetectClass: "stable"},
		{Type: "upgrade", Building: "stable", DetectClass: "stable"},
		{Type: "upgrade", Building: "stable", DetectClass: "stable"},

		// Requires Lvl 5 City Hall
		{Type: "build_new", Building: "siege_workshop", DetectClass: "military:build_siege_workshop"},
		{Type: "upgrade", Building: "siege_workshop", DetectClass: "siege_workshop"},
		{Type: "upgrade", Building: "siege_workshop", DetectClass: "siege_workshop"},
		{Type: "upgrade", Building: "siege_workshop", DetectClass: "siege_workshop"},
	}
}

// isMultipleTypeBuilding checks if this building type can have multiple instances
func isMultipleTypeBuilding(buildingType string) bool {
	multipleTypes := map[string]bool{
		"farm":        true,
		"quarry":      true,
		"lumber_mill": true,
		"goldmine":    true,
		"hospital":    true,
	}

	return multipleTypes[buildingType]
}

// UpdateMainBuildingPosition updates the position of a main building if not already set
func UpdateMainBuildingPosition(buildingType string, x, y int, instanceState *state.InstanceState) {
	switch buildingType {
	case "farm":
		// Only set if not already set (X and Y are both zero)
		if instanceState.BuildingPositions.Farm.X == 0 && instanceState.BuildingPositions.Farm.Y == 0 {
			instanceState.BuildingPositions.Farm.X = x
			instanceState.BuildingPositions.Farm.Y = y
			log.Printf("Set main farm position to (%d, %d)", x, y)
		}
	case "quarry":
		if instanceState.BuildingPositions.Quarry.X == 0 && instanceState.BuildingPositions.Quarry.Y == 0 {
			instanceState.BuildingPositions.Quarry.X = x
			instanceState.BuildingPositions.Quarry.Y = y
			log.Printf("Set main quarry position to (%d, %d)", x, y)
		}
	case "lumber_mill":
		if instanceState.BuildingPositions.LumberMill.X == 0 && instanceState.BuildingPositions.LumberMill.Y == 0 {
			instanceState.BuildingPositions.LumberMill.X = x
			instanceState.BuildingPositions.LumberMill.Y = y
			log.Printf("Set main lumber mill position to (%d, %d)", x, y)
		}
	case "goldmine":
		if instanceState.BuildingPositions.Goldmine.X == 0 && instanceState.BuildingPositions.Goldmine.Y == 0 {
			instanceState.BuildingPositions.Goldmine.X = x
			instanceState.BuildingPositions.Goldmine.Y = y
			log.Printf("Set main goldmine position to (%d, %d)", x, y)
		}
	case "hospital":
		if instanceState.BuildingPositions.Hospital.X == 0 && instanceState.BuildingPositions.Hospital.Y == 0 {
			instanceState.BuildingPositions.Hospital.X = x
			instanceState.BuildingPositions.Hospital.Y = y
			log.Printf("Set main hospital position to (%d, %d)", x, y)
		}
	}
}

// GetMainBuildingPosition gets the position of a main building
func GetMainBuildingPosition(buildingType string, instanceState *state.InstanceState) (x, y int, isSet bool) {
	switch buildingType {
	case "farm":
		return instanceState.BuildingPositions.Farm.X, instanceState.BuildingPositions.Farm.Y,
			instanceState.BuildingPositions.Farm.X != 0 || instanceState.BuildingPositions.Farm.Y != 0
	case "quarry":
		return instanceState.BuildingPositions.Quarry.X, instanceState.BuildingPositions.Quarry.Y,
			instanceState.BuildingPositions.Quarry.X != 0 || instanceState.BuildingPositions.Quarry.Y != 0
	case "lumber_mill":
		return instanceState.BuildingPositions.LumberMill.X, instanceState.BuildingPositions.LumberMill.Y,
			instanceState.BuildingPositions.LumberMill.X != 0 || instanceState.BuildingPositions.LumberMill.Y != 0
	case "goldmine":
		return instanceState.BuildingPositions.Goldmine.X, instanceState.BuildingPositions.Goldmine.Y,
			instanceState.BuildingPositions.Goldmine.X != 0 || instanceState.BuildingPositions.Goldmine.Y != 0
	case "hospital":
		return instanceState.BuildingPositions.Hospital.X, instanceState.BuildingPositions.Hospital.Y,
			instanceState.BuildingPositions.Hospital.X != 0 || instanceState.BuildingPositions.Hospital.Y != 0
	}
	return 0, 0, false
}

// ProcessBuildOrder processes the build order tasks
func ProcessBuildOrder(
	deviceID string,
	gameView string,
	detections []common.Detection,
	adbPath string,
	config common.TaskConfig,
	instanceState *state.InstanceState,
) bool {
	// Skip if not in city view
	if gameView != "city" {
		log.Println("Not in city view, can't process build tasks")
		return false
	}

	// Check if a builder is available
	builderAvailable := false
	for _, det := range detections {
		if det.Class == "builders_hut" && det.Confidence > common.MinConfidence {
			builderAvailable = true
			log.Printf("Found idle builder at (%.1f, %.1f) with confidence %.2f",
				det.X, det.Y, det.Confidence)
			break
		}
	}

	if !builderAvailable {
		log.Println("No builder available, skipping build tasks")
		return false
	}

	// Check if the last build attempt was too recent (global cooldown)
	//if time.Since(instanceState.BuildOrder.LastAttemptTime) < (2 * time.Second) {
	//	timeRemaining := 2*time.Second - time.Since(instanceState.BuildOrder.LastAttemptTime)
	//	log.Printf("Build order on cooldown for %.1f more seconds", timeRemaining.Seconds())
	//	return false
	//}

	// Update main building positions for buildings that can have multiples
	for _, det := range detections {
		// Check if this is a building that can have multiples
		switch det.Class {
		case "farm", "quarry", "lumber_mill", "goldmine", "hospital":
			// Update the position if not set yet
			UpdateMainBuildingPosition(det.Class, int(det.X), int(det.Y), instanceState)
		}
	}

	// Initialize build order if it's empty
	if len(instanceState.BuildOrder.UpcomingTasks) == 0 {
		log.Println("Initializing build order tasks")
		instanceState.BuildOrder.UpcomingTasks = DefineDefaultBuildOrder()

		// Log the initialized task list
		log.Println("Build order initialized with the following tasks:")
		for i, task := range instanceState.BuildOrder.UpcomingTasks {
			log.Printf("  %d. %s %s", i+1, task.Type, task.Building)
		}
	}

	// Print a summary of build tasks
	completedCount := 0
	for _, task := range instanceState.BuildOrder.UpcomingTasks {
		if task.Completed {
			completedCount++
		}
	}
	log.Printf("Build order status: %d/%d tasks completed",
		completedCount, len(instanceState.BuildOrder.UpcomingTasks))

	// Loop through tasks until we find ONLY the first non-completed task
	for i := 0; i < len(instanceState.BuildOrder.UpcomingTasks); i++ {
		currentTask := &instanceState.BuildOrder.UpcomingTasks[i]

		// Skip completed tasks
		if currentTask.Completed {
			continue
		}

		log.Printf("Attempting task %d/%d: %s %s (attempt %d)",
			i+1, len(instanceState.BuildOrder.UpcomingTasks),
			currentTask.Type, currentTask.Building, currentTask.Attempts+1)

		// Check if this task has a cooldown and respect it
		if !currentTask.LastAttempt.IsZero() && time.Since(currentTask.LastAttempt) < (30*time.Second) {
			timeRemaining := 30*time.Second - time.Since(currentTask.LastAttempt)
			log.Printf("Task '%s %s' is on cooldown for %.1f more seconds",
				currentTask.Type, currentTask.Building, timeRemaining.Seconds())
			return false
		}

		success := false
		switch currentTask.Type {
		case "build_new":
			success = BuildNewBuilding(deviceID, gameView, detections, adbPath, currentTask, instanceState)
		case "upgrade":
			success = UpgradeBuilding(deviceID, gameView, detections, adbPath, currentTask, instanceState)
		}

		// Update task state
		currentTask.Attempts++
		currentTask.LastAttempt = time.Now()

		// Update the global last attempt time regardless of success
		instanceState.BuildOrder.LastAttemptTime = time.Now()

		if success {
			// Mark as completed and add to completed tasks list
			currentTask.Completed = true

			completedTask := *currentTask
			instanceState.BuildOrder.CompletedTasks = append(
				instanceState.BuildOrder.CompletedTasks, completedTask)

			log.Printf("Build task completed: %s %s", currentTask.Type, currentTask.Building)

			return true
		} else {
			log.Printf("Build task failed (attempt %d): %s %s",
				currentTask.Attempts, currentTask.Type, currentTask.Building)
			return false // Return false after one failure
		}
	}

	log.Println("No available tasks (all completed)")
	return false
}

// BuildNewBuilding handles building a new structure
func BuildNewBuilding(
	deviceID string,
	gameView string,
	detections []common.Detection,
	adbPath string,
	task *state.BuildTask,
	instanceState *state.InstanceState,
) bool {
	log.Printf("Starting new building: %s", task.Building)

	// Log all the available detections for debugging
	log.Printf("Available detections for building (%d total):", len(detections))
	for i, det := range detections {
		if det.Confidence > common.MinConfidence {
			log.Printf("  %d. %s (%.2f): (%.1f, %.1f) %.0fx%.0f",
				i+1, det.Class, det.Confidence, det.X, det.Y, det.Width, det.Height)
		}
	}

	// First, click the "build new" button
	buildNewButton := vision.FindDetectionByClass(detections, "build_available", common.MinConfidence)
	if buildNewButton == nil {
		log.Println("No build button found, checking for build_new_button instead")
		buildNewButton = vision.FindDetectionByClass(detections, "build_new_button", common.MinConfidence)

		if buildNewButton == nil {
			log.Println("No build buttons found at all, returning")
			return false
		}
	}

	// Click the found button
	log.Printf("Found build button at (%.1f, %.1f), clicking...", buildNewButton.X, buildNewButton.Y)
	if err := utils.TapScreen(deviceID, adbPath, int(buildNewButton.X), int(buildNewButton.Y)); err != nil {
		log.Printf("Error tapping build button: %v", err)
		return false
	}

	// Wait for building menu to appear
	log.Println("Waiting for building menu to appear...")
	time.Sleep(1 * time.Second)

	// Parse the detect class to check for category prefix (economic: or military:)
	detectionParams := strings.Split(task.DetectClass, ":")
	var category, buildingClass string

	if len(detectionParams) > 1 {
		// Format is "category:building_class"
		category = strings.TrimSpace(detectionParams[0])
		buildingClass = strings.TrimSpace(detectionParams[1])
		log.Printf("Using category '%s' for building class '%s'", category, buildingClass)
	} else {
		// No category specified, use the whole string as the building class
		buildingClass = strings.TrimSpace(task.DetectClass)
		log.Printf("No category specified in %s, returning..", buildingClass)
		return false
	}

	// Take new screenshot and detect building options
	log.Println("Taking new screenshot to detect building options...")
	buildMenuDetections, err := vision.CaptureAndDetect(deviceID, adbPath)
	if err != nil {
		log.Printf("Error getting detections for building menu: %v", err)
		resetView(deviceID, adbPath)
		return false
	}

	// Log the building menu detections
	log.Printf("Building menu detections (%d total):", len(buildMenuDetections))
	for i, det := range buildMenuDetections {
		if det.Confidence > common.MinConfidence {
			log.Printf("  %d. %s (%.2f): (%.1f, %.1f) %.0fx%.0f",
				i+1, det.Class, det.Confidence, det.X, det.Y, det.Width, det.Height)
		}
	}

	var categoryButton *common.Detection

	switch strings.ToLower(category) {
	case "economic":
		categoryButton = vision.FindDetectionByClass(buildMenuDetections, "build_economic", common.MinConfidence)
		log.Println("Looking for economic buildings tab")
	case "military":
		categoryButton = vision.FindDetectionByClass(buildMenuDetections, "build_military", common.MinConfidence)
		log.Println("Looking for military buildings tab")
	default:
		log.Printf("Unknown build interface category: %s", category)
		resetView(deviceID, adbPath)
		return false
	}

	if categoryButton != nil {
		log.Printf("Clicking on %s buildings tab at (%.1f, %.1f)",
			category, categoryButton.X, categoryButton.Y)
		if err := utils.TapScreen(deviceID, adbPath, int(categoryButton.X), int(categoryButton.Y)); err != nil {
			log.Printf("Error tapping %s tab: %v", category, err)
			resetView(deviceID, adbPath)
			return false
		}

		// Wait for tab to activate
		time.Sleep(1 * time.Second)

		// Get fresh detections after switching tabs
		log.Println("Getting fresh detections after switching tabs...")
		buildMenuDetections, err = vision.CaptureAndDetect(deviceID, adbPath)
		if err != nil {
			log.Printf("Error getting detections after switching to %s tab: %v", category, err)
			resetView(deviceID, adbPath)
			return false
		}

		// Log the updated building menu detections
		log.Printf("Updated building menu detections after tab switch (%d total):", len(buildMenuDetections))
		for i, det := range buildMenuDetections {
			if det.Confidence > common.MinConfidence {
				log.Printf("  %d. %s (%.2f): (%.1f, %.1f) %.0fx%.0f",
					i+1, det.Class, det.Confidence, det.X, det.Y, det.Width, det.Height)
			}
		}
	} else {
		log.Printf("Could not find %s tab button", category)
		resetView(deviceID, adbPath)
		return false
	}

	// Look for the specific building type option
	buildingButton := vision.FindDetectionByClass(buildMenuDetections, buildingClass, common.MinConfidence)
	if buildingButton == nil {
		log.Printf("Building option for '%s' not found, checking for alternative format...", buildingClass)

		// Try with "build_" prefix
		buildingButtonAlt := vision.FindDetectionByClass(buildMenuDetections, "build_"+buildingClass, common.MinConfidence)
		if buildingButtonAlt != nil {
			buildingButton = buildingButtonAlt
			log.Printf("Found alternative format 'build_%s'", buildingClass)
		} else {
			log.Printf("Building option for '%s' not found with any format", buildingClass)
			resetView(deviceID, adbPath)
			return false
		}
	}

	// Click on the building option
	log.Printf("Clicking on %s building at (%.1f, %.1f)", buildingClass, buildingButton.X, buildingButton.Y)
	if err := utils.TapScreen(deviceID, adbPath, int(buildingButton.X), int(buildingButton.Y)); err != nil {
		log.Printf("Error tapping building option: %v", err)
		resetView(deviceID, adbPath)
		return false
	}

	// Wait for placement mode
	log.Println("Waiting for placement mode...")
	time.Sleep(1 * time.Second)

	// Look for confirm button
	log.Println("Taking screenshot to find confirm button...")
	confirmDetections, err := vision.CaptureAndDetect(deviceID, adbPath)
	if err != nil {
		log.Printf("Error getting detections for confirm button: %v", err)
		resetView(deviceID, adbPath)
		return false
	}

	// Log the confirm screen detections
	log.Printf("Confirm screen detections (%d total):", len(confirmDetections))
	for i, det := range confirmDetections {
		if det.Confidence > common.MinConfidence {
			log.Printf("  %d. %s (%.2f): (%.1f, %.1f) %.0fx%.0f",
				i+1, det.Class, det.Confidence, det.X, det.Y, det.Width, det.Height)
		}
	}

	// First check if both builders are busy
	buildersBusy := vision.FindDetectionByClass(confirmDetections, "builders_hut", common.MinConfidence)
	if buildersBusy != nil {
		log.Printf("Both builders are busy, cannot build new %s at this time", task.Building)

		// Look for exit_dialog_button to gracefully exit
		exitDialogButton := vision.FindDetectionByClass(confirmDetections, "exit_dialog_button", common.MinConfidence)
		if exitDialogButton != nil {
			log.Printf("Found exit_dialog_button at (%.1f, %.1f), clicking to exit builders busy dialog...",
				exitDialogButton.X, exitDialogButton.Y)
			if err := utils.TapScreen(deviceID, adbPath, int(exitDialogButton.X), int(exitDialogButton.Y)); err != nil {
				log.Printf("Error clicking on exit dialog button: %v", err)
			}
		}

		resetView(deviceID, adbPath)
		return false
	}

	confirmButton := vision.FindDetectionByClass(confirmDetections, "accept_build_location", common.MinConfidence)
	if confirmButton == nil {
		log.Println("Accept build location button not found, checking for confirm_button instead")
		confirmButton = vision.FindDetectionByClass(confirmDetections, "confirm_button", common.MinConfidence)

		if confirmButton == nil {
			log.Println("No confirmation buttons found, failing build operation")
			resetView(deviceID, adbPath)
			return false
		} else {
			// Click confirm button
			log.Printf("Found confirm_button at (%.1f, %.1f), clicking...", confirmButton.X, confirmButton.Y)
			if err := utils.TapScreen(deviceID, adbPath, int(confirmButton.X), int(confirmButton.Y)); err != nil {
				log.Printf("Error tapping confirm button: %v", err)
				resetView(deviceID, adbPath)
				return false
			}
		}
	} else {
		log.Printf("Found accept_build_location at (%.1f, %.1f), clicking...", confirmButton.X, confirmButton.Y)
		if err := utils.TapScreen(deviceID, adbPath, int(confirmButton.X), int(confirmButton.Y)); err != nil {
			log.Printf("Error tapping confirm button: %v", err)
			resetView(deviceID, adbPath)
			return false
		}
	}

	// Wait for confirmation
	log.Println("Waiting for confirmation dialog...")
	time.Sleep(1 * time.Second)

	// Check for alliance help request if available
	log.Println("Taking screenshot to check for alliance help request...")
	helpDetections, err := vision.CaptureAndDetect(deviceID, adbPath)
	if err != nil {
		log.Printf("Error getting detections for alliance help: %v", err)
	} else {
		// Log the help screen detections
		log.Printf("Help screen detections (%d total):", len(helpDetections))
		for i, det := range helpDetections {
			if det.Confidence > common.MinConfidence {
				log.Printf("  %d. %s (%.2f): (%.1f, %.1f) %.0fx%.0f",
					i+1, det.Class, det.Confidence, det.X, det.Y, det.Width, det.Height)
			}
		}

		helpButton := vision.FindDetectionByClass(helpDetections, "alliance_help_button", common.MinConfidence)
		if helpButton != nil {
			log.Printf("Clicking on alliance help request button at (%.1f, %.1f)...", helpButton.X, helpButton.Y)
			if err := utils.TapScreen(deviceID, adbPath, int(helpButton.X), int(helpButton.Y)); err != nil {
				log.Printf("Error tapping help button: %v", err)
			} else {
				log.Println("Successfully clicked alliance help button")
				time.Sleep(500 * time.Millisecond)
			}
		} else {
			log.Println("No alliance help button found")
		}
	}

	// Get the current number of buildings of this type (for tracking purposes)
	var hasExistingBuilding bool
	switch task.Building {
	case "farm":
		hasExistingBuilding = instanceState.BuildingPositions.Farm.X != 0 || instanceState.BuildingPositions.Farm.Y != 0
	case "quarry":
		hasExistingBuilding = instanceState.BuildingPositions.Quarry.X != 0 || instanceState.BuildingPositions.Quarry.Y != 0
	case "lumber_mill":
		hasExistingBuilding = instanceState.BuildingPositions.LumberMill.X != 0 || instanceState.BuildingPositions.LumberMill.Y != 0
	case "goldmine":
		hasExistingBuilding = instanceState.BuildingPositions.Goldmine.X != 0 || instanceState.BuildingPositions.Goldmine.Y != 0
	case "hospital":
		hasExistingBuilding = instanceState.BuildingPositions.Hospital.X != 0 || instanceState.BuildingPositions.Hospital.Y != 0
	}

	if hasExistingBuilding {
		log.Printf("Already tracking a main %s building", task.Building)
	} else {
		log.Printf("No existing %s tracked yet", task.Building)
	}

	// Take another screenshot to try to detect the new building
	time.Sleep(1 * time.Second) // Wait for UI to update after building placement

	log.Println("Taking screenshot to detect newly placed building...")
	newDetections, err := vision.CaptureAndDetect(deviceID, adbPath)
	if err != nil {
		log.Printf("Error getting detections for new building: %v", err)
	} else {
		// Log the new building detections
		log.Printf("New building detections (%d total):", len(newDetections))
		for i, det := range newDetections {
			if det.Confidence > common.MinConfidence {
				log.Printf("  %d. %s (%.2f): (%.1f, %.1f) %.0fx%.0f",
					i+1, det.Class, det.Confidence, det.X, det.Y, det.Width, det.Height)
			}
		}

		// Update building positions with the new detections
		for _, det := range newDetections {
			if det.Class == task.Building && isMultipleTypeBuilding(task.Building) {
				UpdateMainBuildingPosition(task.Building, int(det.X), int(det.Y), instanceState)
				log.Printf("Updated position for %s to (%d, %d)", task.Building, int(det.X), int(det.Y))
			}
		}
	}

	log.Println("Build new operation complete, resetting view...")
	// Reset view to ensure we're back in a known state
	resetView(deviceID, adbPath)
	return true
}

// UpgradeBuilding handles upgrading an existing building
func UpgradeBuilding(
	deviceID string,
	gameView string,
	detections []common.Detection,
	adbPath string,
	task *state.BuildTask,
	instanceState *state.InstanceState,
) bool {
	log.Printf("Attempting to upgrade %s", task.Building)

	// Log all the available detections for debugging
	log.Printf("Available detections for upgrading (%d total):", len(detections))
	for i, det := range detections {
		if det.Confidence > common.MinConfidence {
			log.Printf("  %d. %s (%.2f): (%.1f, %.1f) %.0fx%.0f",
				i+1, det.Class, det.Confidence, det.X, det.Y, det.Width, det.Height)
		}
	}

	// Check if we have a stored position for the main building of this type
	var clickX, clickY int
	var useStoredPosition bool

	if isMultipleTypeBuilding(task.Building) {
		mainX, mainY, hasMainPosition := GetMainBuildingPosition(task.Building, instanceState)
		if hasMainPosition {
			// Use the stored position for the main building
			log.Printf("Using stored position (%d, %d) for main %s", mainX, mainY, task.Building)
			clickX = mainX
			clickY = mainY
			useStoredPosition = true
		}
	}

	// If not using a stored position, find the building in detections
	if !useStoredPosition {
		// Split the detection classes by comma
		detectClasses := strings.Split(task.DetectClass, ",")
		var building *common.Detection

		log.Printf("Looking for building with classes: %s", task.DetectClass)

		// Try each detection class
		for _, class := range detectClasses {
			class = strings.TrimSpace(class)
			log.Printf("Checking for class: '%s'", class)
			building = vision.FindDetectionByClass(detections, class, common.MinConfidence)
			if building != nil {
				log.Printf("Found building with class '%s'", class)
				break // Found it with one of the classes
			}
		}

		// If building not found, cannot proceed
		if building == nil {
			log.Printf("%s not found in detections with any of the specified classes", task.Building)
			return false
		}

		clickX = int(building.X)
		clickY = int(building.Y)
		log.Printf("Found %s at position (%d, %d)", task.Building, clickX, clickY)

		// If this is a building that can have multiples, store the position
		if isMultipleTypeBuilding(task.Building) {
			UpdateMainBuildingPosition(task.Building, clickX, clickY, instanceState)
			log.Printf("Updated position for multiple-type building %s to (%d, %d)",
				task.Building, clickX, clickY)
		}
	}

	// Click on the building
	log.Printf("Clicking on %s at (%d, %d)", task.Building, clickX, clickY)
	if err := utils.TapScreen(deviceID, adbPath, clickX, clickY); err != nil {
		log.Printf("Error clicking on %s: %v", task.Building, err)
		resetView(deviceID, adbPath)
		return false
	}

	// Wait for menu to appear
	log.Println("Waiting for building menu to appear...")
	time.Sleep(1 * time.Second)

	// Take another screenshot and get detections to find the upgrade button
	log.Println("Taking screenshot to find upgrade button...")
	upgradeDetections, err := vision.CaptureAndDetect(deviceID, adbPath)
	if err != nil {
		log.Printf("Error getting detections for upgrade button: %v", err)
		resetView(deviceID, adbPath)
		return false
	}

	// Log the upgrade menu detections
	log.Printf("Upgrade menu detections (%d total):", len(upgradeDetections))
	for i, det := range upgradeDetections {
		if det.Confidence > common.MinConfidence {
			log.Printf("  %d. %s (%.2f): (%.1f, %.1f) %.0fx%.0f",
				i+1, det.Class, det.Confidence, det.X, det.Y, det.Width, det.Height)
		}
	}

	// Look for upgrade button
	upgradeButton := vision.FindDetectionByClass(upgradeDetections, "upgrade_button", common.MinConfidence)

	// If upgrade button not found, try alternative names or reset view and exit
	if upgradeButton == nil {
		log.Println("Upgrade button not found, checking for alternative button names")

		// Try alternatives like "upgrade_available" etc.
		alternativeNames := []string{"upgrade_available", "upgrade_building", "building_upgrade"}
		for _, altName := range alternativeNames {
			upgradeButton = vision.FindDetectionByClass(upgradeDetections, altName, common.MinConfidence)
			if upgradeButton != nil {
				log.Printf("Found alternative upgrade button: %s", altName)
				break
			}
		}

		if upgradeButton == nil {
			log.Printf("No upgrade button found for %s with any name, resetting view", task.Building)
			resetView(deviceID, adbPath)
			return false
		}
	}

	// Click on upgrade button
	log.Printf("Clicking on upgrade button at (%.1f, %.1f)...", upgradeButton.X, upgradeButton.Y)
	if err := utils.TapScreen(deviceID, adbPath, int(upgradeButton.X), int(upgradeButton.Y)); err != nil {
		log.Printf("Error clicking on upgrade button: %v", err)
		resetView(deviceID, adbPath)
		return false
	}

	// Wait for upgrade dialog to appear
	log.Println("Waiting for upgrade dialog to appear...")
	time.Sleep(800 * time.Millisecond)

	// Take another screenshot to find the confirmation button
	log.Println("Taking screenshot to find confirmation button...")
	confirmDetections, err := vision.CaptureAndDetect(deviceID, adbPath)
	if err != nil {
		log.Printf("Error getting detections for confirm button: %v", err)
		resetView(deviceID, adbPath)
		return false
	}

	// Log the confirm dialog detections
	log.Printf("Confirm dialog detections (%d total):", len(confirmDetections))
	for i, det := range confirmDetections {
		if det.Confidence > common.MinConfidence {
			log.Printf("  %d. %s (%.2f): (%.1f, %.1f) %.0fx%.0f",
				i+1, det.Class, det.Confidence, det.X, det.Y, det.Width, det.Height)
		}
	}

	// First check if both builders are busy
	buildersBusy := vision.FindDetectionByClass(confirmDetections, "builders_hut_busy", common.MinConfidence)
	if buildersBusy != nil {
		log.Printf("Both builders are busy, cannot upgrade %s at this time", task.Building)

		// Look for exit_dialog_button to gracefully exit
		exitDialogButton := vision.FindDetectionByClass(confirmDetections, "exit_dialog_button", common.MinConfidence)
		if exitDialogButton != nil {
			log.Printf("Found exit_dialog_button at (%.1f, %.1f), clicking to exit builders busy dialog...",
				exitDialogButton.X, exitDialogButton.Y)
			if err := utils.TapScreen(deviceID, adbPath, int(exitDialogButton.X), int(exitDialogButton.Y)); err != nil {
				log.Printf("Error clicking on exit dialog button: %v", err)
			}
		}

		resetView(deviceID, adbPath)
		return false
	}

	// Check if we have a "requirements not met" message
	requirementsNotMet := vision.FindDetectionByClass(confirmDetections, "upgrade_not_available", common.MinConfidence)

	if requirementsNotMet != nil {
		log.Printf("Requirements not met for upgrading %s - detected 'upgrade_not_available'", task.Building)

		log.Println("Looking for exit_dialog_button to dismiss requirements message")
		exitDialogButton := vision.FindDetectionByClass(confirmDetections, "exit_dialog_button", common.MinConfidence)

		if exitDialogButton != nil {
			log.Printf("Found exit_dialog_button at (%.1f, %.1f), clicking...",
				exitDialogButton.X, exitDialogButton.Y)
			if err := utils.TapScreen(deviceID, adbPath, int(exitDialogButton.X), int(exitDialogButton.Y)); err != nil {
				log.Printf("Error clicking on exit dialog button: %v", err)
				resetView(deviceID, adbPath)
				return false
			}
		} else {
			log.Println("Could not find exit_dialog_button, trying to reset view")
			resetView(deviceID, adbPath)
			return false
		}
	}

	// Look for upgrade_available_button or confirm_button
	confirmButton := vision.FindDetectionByClass(confirmDetections, "upgrade_available_button", common.MinConfidence)
	if confirmButton == nil {
		log.Println("upgrade_available_button not found, checking for confirm_button")
		confirmButton = vision.FindDetectionByClass(confirmDetections, "confirm_button", common.MinConfidence)
	}

	// If confirmation button found, click it
	if confirmButton != nil {
		log.Printf("Found confirm button at (%.1f, %.1f), clicking...", confirmButton.X, confirmButton.Y)
		if err := utils.TapScreen(deviceID, adbPath, int(confirmButton.X), int(confirmButton.Y)); err != nil {
			log.Printf("Error clicking on confirm button: %v", err)
			resetView(deviceID, adbPath)
			return false
		}
	} else {
		log.Println("No confirm button found, failing upgrade operation")
		resetView(deviceID, adbPath)
		return false
	}

	// Wait for processing
	log.Println("Waiting for processing...")
	time.Sleep(1 * time.Second)

	// Take one more screenshot to check for alliance help request
	log.Println("Taking screenshot to check for alliance help button...")
	helpDetections, err := vision.CaptureAndDetect(deviceID, adbPath)
	if err != nil {
		log.Printf("Error getting detections for help button: %v", err)
		// Continue anyway as the upgrade should have started
	} else {
		// Log the help screen detections
		log.Printf("Help request detections (%d total):", len(helpDetections))
		for i, det := range helpDetections {
			if det.Confidence > common.MinConfidence {
				log.Printf("  %d. %s (%.2f): (%.1f, %.1f) %.0fx%.0f",
					i+1, det.Class, det.Confidence, det.X, det.Y, det.Width, det.Height)
			}
		}

		// Look for alliance_help_button
		helpButton := vision.FindDetectionByClass(helpDetections, "alliance_help_button", common.MinConfidence)

		// If help button found, click it
		if helpButton != nil {
			log.Printf("Clicking on alliance help request button at (%.1f, %.1f)...", helpButton.X, helpButton.Y)
			if err := utils.TapScreen(deviceID, adbPath, int(helpButton.X), int(helpButton.Y)); err != nil {
				log.Printf("Error tapping help button: %v", err)
			} else {
				log.Println("Successfully clicked alliance help button")
				time.Sleep(500 * time.Millisecond)
			}
		} else {
			log.Println("No alliance help button found")
		}
	}

	log.Printf("%s upgrade initiated successfully", task.Building)

	// Reset view to ensure we're back in a known state
	log.Println("Upgrade operation complete, resetting view...")
	resetView(deviceID, adbPath)

	return true
}

// Helper function to reset the view by clicking in the home button area
// Helper function to reset the view by determining the current state and taking appropriate action
func resetView(deviceID, adbPath string) {
	log.Println("Resetting view to return to normal city view...")

	// Take a screenshot to detect current state
	screenshot, err := vision.CaptureScreenshot(deviceID, adbPath)
	if err != nil {
		log.Printf("Error capturing screenshot for view reset: %v", err)
		// Fallback to default reset approach if we can't detect the state
		defaultReset(deviceID, adbPath)
		return
	}

	// Analyze the current state
	_, detections, err := vision.AnalyzeGameState(
		screenshot,
		utils.GetRoboflowAPIKey(),
		utils.GetRoboflowGameplayModel(),
	)
	if err != nil {
		log.Printf("Error analyzing game state for view reset: %v", err)
		// Fallback to default reset approach if we can't analyze
		defaultReset(deviceID, adbPath)
		return
	}

	// Check if we're in build menu
	inBuild := false
	for _, det := range detections {
		if det.Class == "in_build" && det.Confidence > common.MinConfidence {
			inBuild = true
			break
		}
	}

	if inBuild {
		log.Println("Detected we're in build menu, using escape key to exit...")
		// Use Android back button (escape key) for build menu
		utils.PressKey(deviceID, adbPath, "4") // Android back button keycode
		time.Sleep(800 * time.Millisecond)

		// Press again just to be sure
		utils.PressKey(deviceID, adbPath, "4")
		time.Sleep(800 * time.Millisecond)
	} else {
		// For normal city/field view, use the home button approach
		log.Println("Using home button approach for normal view reset")
		utils.TapScreen(deviceID, adbPath, 31, 450) // Home button location
		time.Sleep(800 * time.Millisecond)
		utils.TapScreen(deviceID, adbPath, 31, 450) // Second click just to be sure
		time.Sleep(800 * time.Millisecond)
	}

	log.Println("View reset sequence completed")
}

// Default reset method as fallback
func defaultReset(deviceID, adbPath string) {
	log.Println("Using default reset approach (home button + escape key)")
	// Try home button first
	utils.TapScreen(deviceID, adbPath, 31, 450)
	time.Sleep(800 * time.Millisecond)

	// Then try escape key
	utils.PressKey(deviceID, adbPath, "4") // Android back button keycode
	time.Sleep(800 * time.Millisecond)
}

// RunBuildOrderTask is the handler function to be called from the task system
func RunBuildOrderTask(
	deviceID string,
	gameView string,
	detections []common.Detection,
	adbPath string,
	config common.TaskConfig,
	instanceState *state.InstanceState,
) bool {
	return ProcessBuildOrder(deviceID, gameView, detections, adbPath, config, instanceState)
}
