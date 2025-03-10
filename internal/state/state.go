package state

import (
	"encoding/json"
	"os"
	"time"
)

// BuildingPositions tracks the positions of main buildings that can have multiples
type BuildingPositions struct {
	Farm       struct{ X, Y int } `json:"farm"`
	Quarry     struct{ X, Y int } `json:"quarry"`
	LumberMill struct{ X, Y int } `json:"lumber_mill"`
	Goldmine   struct{ X, Y int } `json:"goldmine"`
	Hospital   struct{ X, Y int } `json:"hospital"`
}

// BuildOrder tracks both completed and upcoming build tasks
type BuildOrder struct {
	CompletedTasks  []BuildTask `json:"completed_tasks"`   // Tasks that have been successfully completed
	UpcomingTasks   []BuildTask `json:"upcoming_tasks"`    // Tasks still to be done
	LastAttemptTime time.Time   `json:"last_attempt_time"` // When we last tried to execute any build task
}

// BuildTask represents a single building or upgrade task in the ordered list
type BuildTask struct {
	Type        string                 `json:"type"`         // "build" or "upgrade"
	Building    string                 `json:"building"`     // Building name
	DetectClass string                 `json:"detect_class"` // Detection class to look for
	Completed   bool                   `json:"completed"`    // Whether this task has been completed
	Attempts    int                    `json:"attempts"`     // Number of attempts made
	LastAttempt time.Time              `json:"last_attempt"` // When we last tried this task
	Config      map[string]interface{} `json:"config"`       // Optional configuration for speedups etc.
}

// InstanceState represents the persistent state of a game instance
type InstanceState struct {
	ID                             string            `json:"id"`
	DeviceID                       string            `json:"device_id"`
	TutorialCompleted              bool              `json:"tutorial_completed"`
	TutorialUpgradeCompleteClicked bool              `json:"tutorial_upgrade_complete_clicked"`
	TutorialFinalArrowClicked      bool              `json:"tutorial_final_arrow_clicked"`
	StartupTasksCompleted          bool              `json:"startup_tasks_completed"`
	TreeClearingCompleted          bool              `json:"tree_clearing_completed"`
	SecondBuilderAdded             bool              `json:"second_builder_added"`
	CityHallLevel                  int               `json:"city_hall_level"`
	LastScreenshotPath             string            `json:"last_screenshot_path"`
	LastReportTime                 time.Time         `json:"last_report_time"`
	GameState                      GameState         `json:"game_state"`
	ActionPoints                   ActionPointInfo   `json:"action_points"`
	VIP                            VIPState          `json:"vip"`
	ScoutState                     ScoutState        `json:"scout_state"`
	BuilderState                   BuilderState      `json:"builder_state"`
	TavernState                    TavernState       `json:"tavern_state"`
	BuildingPositions              BuildingPositions `json:"building_positions"`
	BuildOrder                     BuildOrder        `json:"build_order"`
}

// GameState contains detailed game state information
type GameState struct {
	Power               int                  `json:"power"`
	Resources           ResourceState        `json:"resources"`
	BuildingsInProgress map[string]time.Time `json:"buildings_in_progress"`
}

// ResourceState tracks in-game resources
type ResourceState struct {
	Food  int `json:"food"`
	Wood  int `json:"wood"`
	Stone int `json:"stone"`
	Gold  int `json:"gold"`
	Gems  int `json:"gems"`
}

// ActionPointInfo tracks action points
type ActionPointInfo struct {
	Current    int       `json:"current"`
	Max        int       `json:"max"`
	LastUpdate time.Time `json:"last_update"`
}

// VIPState tracks VIP-related information
type VIPState struct {
	Level         int       `json:"level"`
	LastClaimTime time.Time `json:"last_claim_time"`
}

// ScoutState tracks scout-related information
type ScoutState struct {
	CurrentX     int       `json:"current_x"`
	CurrentY     int       `json:"current_y"`
	IsMoving     bool      `json:"is_moving"`
	LastMoveTime time.Time `json:"last_move_time"`
}

// BuilderState tracks builder-related information
type BuilderState struct {
	SecondBuilderEndTime time.Time `json:"second_builder_end_time"`
}

// TavernState tracks tavern-related information
type TavernState struct {
	LastSilverChestTime time.Time `json:"last_silver_chest_time"`
	LastGoldChestTime   time.Time `json:"last_gold_chest_time"`
}

// BuildingStates tracks the state of various buildings
type BuildingStates struct {
	// City Hall
	CityHallUpgrading        bool      `json:"cityhall_upgrading"`
	CityHallUpgradeStartTime time.Time `json:"cityhall_upgrade_start_time"`

	// Wall
	WallLevel            int       `json:"wall_level"`
	WallUpgrading        bool      `json:"wall_upgrading"`
	WallUpgradeStartTime time.Time `json:"wall_upgrade_start_time"`

	// Academy
	AcademyLevel            int       `json:"academy_level"`
	AcademyUpgrading        bool      `json:"academy_upgrading"`
	AcademyUpgradeStartTime time.Time `json:"academy_upgrade_start_time"`

	// Barracks
	BarracksLevel            int       `json:"barracks_level"`
	BarracksUpgrading        bool      `json:"barracks_upgrading"`
	BarracksUpgradeStartTime time.Time `json:"barracks_upgrade_start_time"`

	// Add other buildings as needed
}

// NewInstanceState creates a new instance state with default values
func NewInstanceState(id, deviceID string) *InstanceState {
	return &InstanceState{
		ID:                             id,
		DeviceID:                       deviceID,
		TutorialCompleted:              false,
		TutorialUpgradeCompleteClicked: false,
		TutorialFinalArrowClicked:      false,
		CityHallLevel:                  2, // Default to level 2 at end of tutoriail
		LastReportTime:                 time.Time{},
		GameState: GameState{
			Power: 0,
			Resources: ResourceState{
				Food:  0,
				Wood:  0,
				Stone: 0,
				Gold:  0,
				Gems:  0,
			},
			BuildingsInProgress: make(map[string]time.Time),
		},
		ActionPoints: ActionPointInfo{
			Current:    0,
			Max:        0,
			LastUpdate: time.Time{},
		},
		VIP: VIPState{
			Level:         0,
			LastClaimTime: time.Time{},
		},
		ScoutState: ScoutState{
			CurrentX:     0,
			CurrentY:     0,
			IsMoving:     false,
			LastMoveTime: time.Time{},
		},
		BuilderState: BuilderState{
			SecondBuilderEndTime: time.Time{},
		},
		TavernState: TavernState{
			LastSilverChestTime: time.Time{},
			LastGoldChestTime:   time.Time{},
		},
		BuildingPositions: BuildingPositions{},
		BuildOrder: BuildOrder{
			CompletedTasks: []BuildTask{},
			UpcomingTasks:  []BuildTask{},
		},
	}
}

// SaveInstanceStates saves instance states to a JSON file
func SaveInstanceStates(filepath string, states map[string]*InstanceState) error {
	data, err := json.MarshalIndent(states, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath, data, 0644)
}

// LoadInstanceStates loads instance states from a JSON file
func LoadInstanceStates(filepath string) (map[string]*InstanceState, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist yet, return empty map
			return make(map[string]*InstanceState), nil
		}
		return nil, err
	}

	var states map[string]*InstanceState
	if err := json.Unmarshal(data, &states); err != nil {
		return nil, err
	}

	return states, nil
}
