package actions

import (
	"log"
	"roborok/internal/common"
	"roborok/internal/state"
	"roborok/internal/utils"
)

// CollectQuests attempts to collect available quests directly from the sidebar
func CollectQuests(
	deviceID string,
	gameView string,
	detections []common.Detection,
	adbPath string,
	config common.TaskConfig,
	instanceState *state.InstanceState,
) bool {
	log.Printf("Checking for claimable quests on device %s", deviceID)

	// Track if we claimed anything
	claimedAny := false

	// Look for claimable quests in the detections
	for _, det := range detections {
		if det.Class == "main_quest_claimable" && det.Confidence > common.MinConfidence {
			log.Println("Main quest reward available, clicking to claim...")
			if err := utils.TapScreen(deviceID, adbPath, int(det.X), int(det.Y)); err != nil {
				log.Printf("Error clicking on main quest: %v", err)
			} else {
				claimedAny = true
			}
		} else if det.Class == "quests_claimable" && det.Confidence > common.MinConfidence && !config.ClaimOnlyMainQuest {
			log.Println("Regular quest reward available, clicking the top one to claim it...")
			if err := utils.TapScreen(deviceID, adbPath, int(det.X), int(det.Y+78)); err != nil {
				log.Printf("Error clicking on regular quest: %v", err)
			} else {
				claimedAny = true
			}
		}
	}

	if !claimedAny {
		log.Println("No claimable quests detected")
		return false
	}

	log.Println("Quest claims completed")
	return true
}
