package actions

import (
	"fmt"
	"log"
	"os"
	"roborok/internal/common"
	"roborok/internal/state"
	"roborok/internal/utils"
	"time"
)

// TreeCoordinates defines locations of trees in the city
var TreeCoordinates = []struct {
	X, Y int
}{
	{134, 137},
	{195, 169},
	{261, 217},
	{243, 173},
	{292, 216},
	{325, 235},
	{333, 204},
	{371, 225},
	{378, 196},
	{415, 215},
	{479, 278},
	{590, 363},
	{521, 388},
	{574, 261},
	{352, 417},
	{368, 391},
	{299, 411},
	{203, 382},
	{176, 354},
}

// HarvestCoordinates defines where to click for the harvest button
var HarvestCoordinates = []struct {
	X, Y int
}{
	{158, 256},
	{218, 256},
	{286, 256},
	{268, 257},
	{319, 269},
	{356, 292},
	{364, 249},
	{399, 275},
	{406, 244},
	{441, 269},
	{504, 324},
	{582, 415},
	{548, 429},
	{600, 314},
	{375, 467},
	{393, 439},
	{327, 462},
	{228, 426},
	{203, 406},
}

// Global state tracking
var (
	treeIndex        int  // Current tree index being processed
	clearingComplete bool // Whether all trees have been cleared
	viewResetDone    bool // Whether the view reset has been completed
)

// HomeButtonCoordinates for resetting view
const (
	HomeButtonX = 31
	HomeButtonY = 450
)

// ClearTrees attempts to clear trees in the city using hardcoded coordinates
func ClearTrees(
	deviceID string,
	gameView string,
	detections []common.Detection,
	adbPath string,
	config common.TaskConfig,
	instanceState *state.InstanceState,
) bool {
	// Skip if tree clearing was already completed
	if clearingComplete {
		log.Println("Tree clearing already completed, moving to next task")
		return false
	}

	log.Printf("Tree harvesting with pre-set coordinates for device %s", deviceID)

	// Handle view reset between sets of trees
	if treeIndex == 13 && !viewResetDone {
		log.Println("Resetting view to get to next set of trees...")

		// First tap on home button
		if err := utils.TapScreen(deviceID, adbPath, HomeButtonX, HomeButtonY); err != nil {
			log.Printf("Error tapping home button (first tap): %v", err)
			return false
		}

		// Wait briefly
		time.Sleep(500 * time.Millisecond)

		// Second tap on home button
		if err := utils.TapScreen(deviceID, adbPath, HomeButtonX, HomeButtonY); err != nil {
			log.Printf("Error tapping home button (second tap): %v", err)
			return false
		}

		// Wait for view to reset
		time.Sleep(1000 * time.Millisecond)
		log.Println("View reset completed, ready for next trees")

		// Mark view reset as done to avoid looping
		viewResetDone = true

		return false // Return to get a fresh game state
	}

	// If we've gone through all trees, perform final reset and mark as complete
	if treeIndex >= len(TreeCoordinates) {
		log.Println("All tree coordinates have been processed, performing final view reset...")

		// First tap on home button
		if err := utils.TapScreen(deviceID, adbPath, HomeButtonX, HomeButtonY); err != nil {
			log.Printf("Error tapping home button for final reset (first tap): %v", err)
			// Continue even if there's an error
		}

		// Wait briefly
		time.Sleep(500 * time.Millisecond)

		// Second tap on home button
		if err := utils.TapScreen(deviceID, adbPath, HomeButtonX, HomeButtonY); err != nil {
			log.Printf("Error tapping home button for final reset (second tap): %v", err)
			// Continue even if there's an error
		}

		// Wait for view to reset
		time.Sleep(1000 * time.Millisecond)
		log.Println("Final view reset completed")

		// Mark as complete
		clearingComplete = true

		// Record completion in a file
		f, err := os.OpenFile("tree_clearing_complete.txt", os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			defer f.Close()
			f.WriteString(fmt.Sprintf("Tree clearing completed at %s\n", time.Now().Format(time.RFC3339)))
		}

		return false
	}

	// Get current tree coordinates
	tree := TreeCoordinates[treeIndex]
	log.Printf("Processing tree %d/%d at position (%d, %d)",
		treeIndex+1, len(TreeCoordinates), tree.X, tree.Y)

	// Click on the tree
	if err := utils.TapScreen(deviceID, adbPath, tree.X, tree.Y); err != nil {
		log.Printf("Error clicking tree at (%d, %d): %v", tree.X, tree.Y, err)
		treeIndex++ // Move to next tree even if this one failed
		return false
	}

	// Get corresponding harvest coordinates
	harvest := HarvestCoordinates[treeIndex]
	log.Printf("Clicking harvest at (%d, %d)", harvest.X, harvest.Y)

	// Wait briefly for harvest button to appear
	time.Sleep(500 * time.Millisecond)

	// Click the harvest button
	if err := utils.TapScreen(deviceID, adbPath, harvest.X, harvest.Y); err != nil {
		log.Printf("Error clicking harvest at (%d, %d): %v", harvest.X, harvest.Y, err)
		treeIndex++ // Move to next tree even if harvest failed
		return false
	}

	// Log success
	log.Printf("Successfully harvested tree %d/%d", treeIndex+1, len(TreeCoordinates))

	// Increment tree index for next run
	treeIndex++

	return true
}

// ResetTreeClearing resets the tree clearing state
// This can be called if you want to restart the process
func ResetTreeClearing() {
	treeIndex = 0
	clearingComplete = false
	viewResetDone = false
	log.Println("Tree clearing state has been reset")
}

func IsTreeClearingComplete() bool {
	return clearingComplete
}
