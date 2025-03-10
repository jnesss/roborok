package common

import (
	"roborok/internal/state"
	"time"
)

// Detection represents a detected object
type Detection struct {
	Class      string
	X          float64
	Y          float64
	Width      float64
	Height     float64
	Confidence float64
}

// Task defines a gameplay task with priority and cooldown
type Task struct {
	Name         string
	Priority     int // Higher number = higher priority
	CooldownSec  int // Minimum seconds between executions
	LastExecuted time.Time
	Config       TaskConfig // Custom configuration for the task
	Handler      func(deviceID, gameView string, detections []Detection, adbPath string, config TaskConfig, instanceState *state.InstanceState) bool
}

// TaskConfig contains custom configuration options for tasks
type TaskConfig struct {
	// Building related configurations
	MaxLevelDesired int // Maximum level to upgrade this building to

	// Quest related configurations
	ClaimOnlyMainQuest bool // Only claim main quest line

	// Training related configurations
	TroopLevelDesired int // Level of troops to train (0 = max available)

	// Research related configurations
	ResearchPath []string // Ordered list of technologies to research

	// Combat related configurations
	BarbLevel         int    // Barbarian level to target
	AllianceName      string // Preferred alliance to join
	UseRandomAlliance bool   // Join a random alliance if preferred not found
}

// DetectionRequirement defines what detection classes are needed for a task
type DetectionRequirement struct {
	// RequiresAny represents detection classes where at least one must be present
	RequiresAny []string

	// RequiresAll represents detection classes where all must be present
	RequiresAll []string

	// RequiresNone represents detection classes that must NOT be present
	RequiresNone []string
}

// IsMet returns true if the requirements are met based on the provided detections
func (req DetectionRequirement) IsMet(detections []Detection) bool {
	// Check RequiresAny (at least one must be present)
	if len(req.RequiresAny) > 0 {
		anyMet := false
		for _, className := range req.RequiresAny {
			for _, det := range detections {
				if det.Class == className && det.Confidence > MinConfidence {
					anyMet = true
					break
				}
			}
			if anyMet {
				break
			}
		}
		if !anyMet {
			return false
		}
	}

	// Check RequiresAll (all must be present)
	for _, className := range req.RequiresAll {
		found := false
		for _, det := range detections {
			if det.Class == className && det.Confidence > MinConfidence {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check RequiresNone (none must be present)
	for _, className := range req.RequiresNone {
		for _, det := range detections {
			if det.Class == className && det.Confidence > MinConfidence {
				return false
			}
		}
	}

	return true
}
