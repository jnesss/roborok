package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"roborok/internal/common"
	"sync"
)

// Global config instance and mutex for thread-safety
var (
	globalConfig     *Config
	globalConfigOnce sync.Once
	configMutex      sync.RWMutex
)

// Config represents the application configuration
type Config struct {
	Global    GlobalConfig              `json:"global"`
	Instances map[string]InstanceConfig `json:"instances"`
	Gameplay  GameplayConfig            `json:"gameplay"`
}

// GlobalConfig contains global settings
type GlobalConfig struct {
	RoboflowAPIKey        string `json:"roboflow_api_key"`
	RoboflowTutorialModel string `json:"roboflow_tutorial_model_id"`
	RoboflowGameplayModel string `json:"roboflow_gameplay_model_id"`
	RefreshIntervalMS     int    `json:"refresh_interval_ms"`
	ReportEndpoint        string `json:"report_endpoint"`
	ReportingIntervalS    int    `json:"reporting_interval_s"`
}

// InstanceConfig contains per-instance settings
type InstanceConfig struct {
	DeviceID                   string `json:"device_id"`
	PreferredCivilization      string `json:"preferred_civilization"`
	ClaimQuests                bool   `json:"claim_quests"`
	ClaimOnlyMainQuest         bool   `json:"claim_only_main_quest"`
	EnableScoutMicromanagement bool   `json:"enable_scout_micromanagement"`
}

// GameplayConfig contains gameplay settings
type GameplayConfig struct {
	ADBPath            string         `json:"adb_path"`
	StartupTasks       []string       `json:"startup_tasks"`
	MaxCityHallLevel   int            `json:"max_city_hall_level"`
	PreferredAlliance  string         `json:"preferred_alliance"`
	JoinRandomAlliance bool           `json:"join_random_alliance"`
	ResearchPath       []string       `json:"research_path"`
	BuildingLevels     map[string]int `json:"building_levels"`
	TroopLevels        map[string]int `json:"troop_levels"`
}

// LoadConfig loads the configuration from a JSON file
func LoadConfig(filepath string) (*Config, error) {
	// Read the file
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	// Parse the JSON
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	// Set default values if not provided
	if config.Gameplay.MaxCityHallLevel == 0 {
		config.Gameplay.MaxCityHallLevel = 25 // Default to max level
	}

	if config.Gameplay.BuildingLevels == nil {
		// Set default building level caps
		config.Gameplay.BuildingLevels = map[string]int{
			"city_hall":      25,
			"barracks":       25,
			"archery_range":  25,
			"stable":         25,
			"siege_workshop": 25,
			"academy":        25,
		}
	}

	if config.Gameplay.TroopLevels == nil {
		// Default to training max level troops
		config.Gameplay.TroopLevels = map[string]int{
			"infantry": 0, // 0 means max available
			"archers":  0,
			"cavalry":  0,
			"siege":    0,
		}
	}

	// Validate required fields
	if config.Global.RoboflowAPIKey == "" {
		config.Global.RoboflowAPIKey = common.DefaultRoboflowAPIKey
	}

	// Set default model IDs if not provided
	if config.Global.RoboflowTutorialModel == "" {
		config.Global.RoboflowTutorialModel = common.TutorialModelID
	}

	if config.Global.RoboflowGameplayModel == "" {
		config.Global.RoboflowGameplayModel = common.GameplayModelID
	}

	if config.Gameplay.ADBPath == "" {
		return nil, fmt.Errorf("missing required field: gameplay.adb_path")
	}

	if len(config.Instances) == 0 {
		return nil, fmt.Errorf("no instances defined in config")
	}

	return &config, nil
}

// InitGlobalConfig initializes the global configuration
// This should be called once during application startup
func InitGlobalConfig(configPath string) error {
	var initErr error

	globalConfigOnce.Do(func() {
		var config *Config
		config, initErr = LoadConfig(configPath)
		if initErr != nil {
			return
		}

		configMutex.Lock()
		globalConfig = config
		configMutex.Unlock()

		log.Println("Global configuration initialized successfully")
	})

	return initErr
}

// GetConfig returns the global configuration
// It will panic if the configuration hasn't been initialized
func GetConfig() *Config {
	configMutex.RLock()
	defer configMutex.RUnlock()

	if globalConfig == nil {
		log.Fatal("Attempted to access global config before initialization")
	}

	return globalConfig
}

// GetRoboflowAPIKey returns the Roboflow API key from the global config
func GetRoboflowAPIKey() string {
	return GetConfig().Global.RoboflowAPIKey
}

// GetRoboflowGameplayModel returns the Roboflow gameplay model ID from the global config
func GetRoboflowGameplayModel() string {
	return GetConfig().Global.RoboflowGameplayModel
}

// GetRoboflowTutorialModel returns the Roboflow tutorial model ID from the global config
func GetRoboflowTutorialModel() string {
	return GetConfig().Global.RoboflowTutorialModel
}
