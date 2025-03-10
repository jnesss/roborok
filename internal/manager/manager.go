package manager

import (
	"fmt"
	"log"
	"path/filepath"
	"roborok/internal/actions"
	"roborok/internal/common"
	"roborok/internal/report"
	"roborok/internal/state"
	"roborok/internal/utils"
	"roborok/internal/vision"
	"sync"
	"time"
)

// Manager handles all game instances
type Manager struct {
	Config              *utils.Config
	Instances           map[string]*Instance
	Reporter            *report.Reporter
	StatePath           string
	paused              bool
	pauseLock           sync.Mutex
	noActionCount       map[string]int  // Track consecutive no-action iterations per instance
	lastBuildSuccessful map[string]bool // Track success by instance ID
}

// Instance represents a game instance
type Instance struct {
	ID       string
	DeviceID string
	Config   utils.InstanceConfig
	State    *state.InstanceState
	Tasks    []common.Task
}

// Define detection requirements for each task
var taskRequirements = map[string]common.DetectionRequirement{
	"request_alliance_help": {
		RequiresAny: []string{"alliance_help_available"},
	},
	"provide_alliance_help": {
		RequiresAny: []string{"alliance_help_requested"},
	},
	"process_build_order": {
		RequiresAll: []string{"in_city"},
	},
	"collect_quests": {
		RequiresAny: []string{"main_quest_claimable", "quests_claimable"},
	},
	"join_alliance": {
		RequiresNone: []string{"in_alliance"},
	},
	"collect_tavern_chests": {
		RequiresAll: []string{"in_city"},
		RequiresAny: []string{"tavern_clickable", "tavern_upgradeable_clickable"},
	},
	"research_academy": {
		RequiresAll: []string{"in_city"},
		RequiresAny: []string{"academy_idle", "academy_upgradeable_idle"},
	},
	"heal_troops": {
		RequiresAll: []string{"in_city"},
		RequiresAny: []string{"hospital_clickable", "hospital_upgradeable_clickable"},
	},
	"train_infantry": {
		RequiresAll: []string{"in_city"},
		RequiresAny: []string{"barracks_idle", "barracks_upgradeable_idle"},
	},
	"train_archers": {
		RequiresAll: []string{"in_city"},
		RequiresAny: []string{"archery_range_idle", "archery_range_upgradeable_idle"},
	},
	"train_cavalry": {
		RequiresAll: []string{"in_city"},
		RequiresAny: []string{"stable_idle", "stable_upgradeable_idle"},
	},
	"train_siege": {
		RequiresAll: []string{"in_city"},
		RequiresAny: []string{"siege_workshop_idle", "siege_workshop_upgradeable_idle"},
	},
	"manage_scouts": {
		RequiresAny: []string{"scout_camp_idle", "scout_camp_upgradeable_idle"},
	},
	// Farm barbarians and challenge barbarians have no specific detection requirements
	// They'll be handled based on troop availability in their handlers
}

// NewManager creates a new instance manager
func NewManager(config *utils.Config, reporter *report.Reporter) *Manager {
	return &Manager{
		Config:              config,
		Instances:           make(map[string]*Instance),
		Reporter:            reporter,
		StatePath:           filepath.Join(".", "instance_states.json"),
		noActionCount:       make(map[string]int),
		lastBuildSuccessful: make(map[string]bool),
	}
}

// LoadInstanceStates loads all instance states from disk
func (m *Manager) LoadInstanceStates(filepath string) error {
	m.StatePath = filepath

	// Load states from file
	states, err := state.LoadInstanceStates(filepath)
	if err != nil {
		return err
	}

	// Initialize instances from config
	for id, cfg := range m.Config.Instances {
		// Check if we have existing state
		instanceState, exists := states[id]
		if !exists {
			// Create new state if none exists
			instanceState = state.NewInstanceState(id, cfg.DeviceID)
		} else {
			// Reset all build order task cooldowns on startup
			for i := range instanceState.BuildOrder.UpcomingTasks {
				if !instanceState.BuildOrder.UpcomingTasks[i].Completed {
					instanceState.BuildOrder.UpcomingTasks[i].LastAttempt = time.Time{}
					log.Printf("[%s] Reset cooldown for task: %s %s",
						id, instanceState.BuildOrder.UpcomingTasks[i].Type,
						instanceState.BuildOrder.UpcomingTasks[i].Building)
				}
			}
		}

		// Ensure device ID is up to date (in case config changed)
		instanceState.DeviceID = cfg.DeviceID

		// Create instance
		m.Instances[id] = &Instance{
			ID:       id,
			DeviceID: cfg.DeviceID,
			Config:   cfg,
			State:    instanceState,
			Tasks:    []common.Task{},
		}

		// Initialize tasks for this instance
		m.initializeTasks(m.Instances[id])
	}

	// Save immediately to ensure file exists and format is correct
	return m.SaveInstanceStates()
}

// initializeTasks sets up the task list for an instance
func (m *Manager) initializeTasks(instance *Instance) {
	// Define tasks in priority order (highest priority first)
	instance.Tasks = []common.Task{
		/*
		   // These tasks will be implemented later
		   {
		       Name:        "request_alliance_help",
		       Priority:    150,  // Very highest priority
		       CooldownSec: 0,    // No cooldown
		       Handler:     actions.RequestAllianceHelp,
		   },
		   {
		       Name:        "provide_alliance_help",
		       Priority:    140,  // Very high priority
		       CooldownSec: 0,    // No cooldown
		       Handler:     actions.ProvideAllianceHelp,
		   },
		*/
		{
			Name:        "process_build_order",
			Priority:    95, // High priority, just below city hall
			CooldownSec: 0,  // Check every second if not successful
			Config:      common.TaskConfig{},
			Handler:     actions.RunBuildOrderTask,
		},
		{
			Name:        "collect_quests",
			Priority:    90,
			CooldownSec: 0, // no cooldown if there are more quests to claim
			Config: common.TaskConfig{
				ClaimOnlyMainQuest: false, // Claim all quests by default
			},
			Handler: actions.CollectQuests,
		},
		/*
		   // These will be implemented next
		   {
		       Name:        "collect_tavern_chests",
		       Priority:    80,
		       CooldownSec: 3600,  // 1 hour
		       Handler:     actions.CollectTavernChests,
		   },
		*/
	}
}

// SaveInstanceStates saves all instance states to disk
func (m *Manager) SaveInstanceStates() error {
	// Convert to map of InstanceState
	states := make(map[string]*state.InstanceState)
	for id, instance := range m.Instances {
		states[id] = instance.State
	}

	return state.SaveInstanceStates(m.StatePath, states)
}

// RunInstanceLoop runs the main loop for a specific instance
func (m *Manager) RunInstanceLoop(id string, instance *Instance) {
	log.Printf("[%s] Starting instance loop", id)

	// First check if tutorial is completed
	if !instance.State.TutorialCompleted {
		log.Printf("[%s] Tutorial not completed, running tutorial automation", id)
		m.RunTutorial(instance)
	}

	// Main gameplay loop
	iterationCount := 0
	for {
		// Check if automation is paused
		m.pauseLock.Lock()
		paused := m.paused
		m.pauseLock.Unlock()

		// Only increment iteration and proceed if not paused
		if !paused {
			// Log iteration count for debugging
			iterationCount++
			log.Printf("[%s] Starting gameplay iteration #%d", id, iterationCount)

			// Run gameplay iteration
			m.RunGameplayIteration(instance)

			// Save state periodically
			if err := m.SaveInstanceStates(); err != nil {
				log.Printf("[%s] Error saving state: %v", id, err)
			}
		} else {
			// If paused, just sleep a bit and check again
			time.Sleep(500 * time.Millisecond)
			continue
		}

		// Calculate delay based on current activity
		delay := time.Duration(m.Config.Global.RefreshIntervalMS) * time.Millisecond

		// Check if we're paused before sleeping
		m.pauseLock.Lock()
		paused = m.paused
		m.pauseLock.Unlock()

		if paused {
			log.Printf("[%s] Iteration #%d completed. Automation is paused. Will continue when you type 'r'",
				id, iterationCount)
		} else {
			log.Printf("[%s] Iteration #%d completed. Sleeping %v ms before restarting loop",
				id, iterationCount, m.Config.Global.RefreshIntervalMS)
		}

		time.Sleep(delay)
	}
}

// RunTutorial handles the tutorial automation
func (m *Manager) RunTutorial(instance *Instance) {
	log.Printf("[%s] Starting tutorial sequence", instance.ID)

	// Maximum attempts for tutorial completion
	const maxTutorialAttempts = 3
	attempts := 0

	for attempts < maxTutorialAttempts {
		// Check if the tutorial is actually done based on our sequence tracking
		isComplete, err := actions.IsTutorialComplete(
			instance.DeviceID,
			m.Config.Gameplay.ADBPath,
			instance.State, // Pass the state for tracking
		)
		if err != nil {
			log.Printf("[%s] Error checking tutorial status: %v", instance.ID, err)
			time.Sleep(1 * time.Second)
			continue
		}

		if isComplete {
			log.Printf("[%s] Tutorial completed, updating state", instance.ID)
			instance.State.TutorialCompleted = true
			instance.State.CityHallLevel = 2 // Tutorial leaves us at CH level 2
			m.SaveInstanceStates()
			return
		}

		log.Printf("[%s] Tutorial needs to be completed (attempt %d/%d), running automation",
			instance.ID, attempts+1, maxTutorialAttempts)

		// Call the tutorial automation function with configuration and state tracking
		success := actions.RunTutorialAutomation(
			instance.DeviceID,
			m.Config.Global.RoboflowAPIKey,
			m.Config.Global.RoboflowTutorialModel,
			m.Config.Gameplay.ADBPath,
			instance.Config.PreferredCivilization,
			instance.State, // Pass the state for tracking
		)

		if success {
			// The automation should have updated the state already
			log.Printf("[%s] Tutorial completed successfully", instance.ID)
			instance.State.TutorialCompleted = true
			instance.State.CityHallLevel = 2
			m.SaveInstanceStates()
			return
		} else {
			log.Printf("[%s] Tutorial automation failed, retrying", instance.ID)
		}

		attempts++

		// If we've tried multiple times and failed, try restarting the app
		if attempts == maxTutorialAttempts-1 {
			log.Printf("[%s] Multiple tutorial attempts failed, restarting the application", instance.ID)
			utils.RestartApp(instance.DeviceID, m.Config.Gameplay.ADBPath)

			// Reset tracking state since we're restarting
			instance.State.TutorialUpgradeCompleteClicked = false
			instance.State.TutorialFinalArrowClicked = false

			time.Sleep(15 * time.Second) // Wait longer for app to fully restart
		} else {
			time.Sleep(5 * time.Second) // Wait before next attempt
		}

		// Save state after each attempt to persist tracking information
		m.SaveInstanceStates()
	}

	log.Printf("[%s] Failed to complete tutorial after %d attempts", instance.ID, maxTutorialAttempts)
}

// Pause pauses all automation
func (m *Manager) Pause() {
	m.pauseLock.Lock()
	defer m.pauseLock.Unlock()
	m.paused = true
}

// Resume resumes all automation
func (m *Manager) Resume() {
	m.pauseLock.Lock()
	defer m.pauseLock.Unlock()
	m.paused = false
}

// IsPaused returns the current pause state
func (m *Manager) IsPaused() bool {
	m.pauseLock.Lock()
	defer m.pauseLock.Unlock()
	return m.paused
}

// Modified RunGameplayIteration function to integrate building state tracking and prerequisites
func (m *Manager) RunGameplayIteration(instance *Instance) {
	// Check if automation is paused
	m.pauseLock.Lock()
	paused := m.paused
	m.pauseLock.Unlock()

	if paused {
		// Sleep briefly then return without doing anything
		time.Sleep(500 * time.Millisecond)
		return
	}

	log.Printf("[%s] Running gameplay iteration", instance.ID)

	// Run one-time startup tasks first (no vision required)
	if !instance.State.StartupTasksCompleted {
		if startupComplete := m.runStartupTasks(instance); startupComplete {
			// All startup tasks are complete, mark it in the state
			instance.State.StartupTasksCompleted = true
			m.SaveInstanceStates()
			log.Printf("[%s] All startup tasks completed", instance.ID)
		} else {
			// If any startup task executed successfully, return to get fresh state
			return
		}
	}

	// Take screenshot for analysis
	screenshot, err := vision.CaptureScreenshot(instance.DeviceID, m.Config.Gameplay.ADBPath)
	if err != nil {
		log.Printf("[%s] Error capturing screenshot: %v", instance.ID, err)
		return
	}

	// Determine if we need to take periodic screenshot for reporting
	timeSinceLastReport := time.Since(instance.State.LastReportTime)
	if timeSinceLastReport > time.Duration(m.Config.Global.ReportingIntervalS)*time.Second {
		screenshotPath := fmt.Sprintf("screenshots/%s_%s.png",
			instance.ID, time.Now().Format("20060102_150405"))
		if err := vision.SaveScreenshot(screenshot, screenshotPath); err != nil {
			log.Printf("[%s] Error saving screenshot: %v", instance.ID, err)
		} else {
			instance.State.LastScreenshotPath = screenshotPath
			instance.State.LastReportTime = time.Now()
			m.Reporter.ReportScreenshot(instance.ID, screenshotPath, map[string]interface{}{
				"city_hall_level": instance.State.CityHallLevel,
			})
		}
	}

	// Get current game state and view (city or map)
	gameView, detections, err := vision.AnalyzeGameState(
		screenshot,
		m.Config.Global.RoboflowAPIKey,
		m.Config.Global.RoboflowGameplayModel,
	)
	if err != nil {
		log.Printf("[%s] Error analyzing game state: %v", instance.ID, err)
		return
	}

	for _, det := range detections {
		if det.Class == "in_build" && det.Confidence > common.MinConfidence {
			log.Printf("[%s] Detected we're in build menu, using escape key to exit...", instance.ID)

			// Use Android back button (escape key) for build menu
			if err := utils.PressKey(instance.DeviceID, m.Config.Gameplay.ADBPath, "4"); err != nil {
				log.Printf("[%s] Error pressing escape key: %v", instance.ID, err)
			} else {
				// Wait briefly and press again just to be sure
				time.Sleep(800 * time.Millisecond)
				utils.PressKey(instance.DeviceID, m.Config.Gameplay.ADBPath, "4")

				// Wait for menu to close
				time.Sleep(800 * time.Millisecond)
				log.Printf("[%s] Exited build menu, restarting iteration", instance.ID)
				return // Return to restart iteration with fresh state
			}
			break
		}
	}

	// First attempt field-specific actions if we're in the field
	if gameView == "field" || gameView == "map" {
		log.Printf("[%s] Currently in %s view", instance.ID, gameView)
		fieldTaskExecuted := false

		// Try field-specific tasks first
		for i := range instance.Tasks {
			task := &instance.Tasks[i]
			requirement, hasRequirement := taskRequirements[task.Name]

			// Skip if task is on cooldown - BUT make exception for build_order when builders are idle
			if time.Since(task.LastExecuted) < time.Duration(task.CooldownSec)*time.Second {
				// Check if this is the build order task and a builder is available and last build was successful
				if task.Name == "process_build_order" {
					builderAvailable := false
					for _, det := range detections {
						if det.Class == "builders_hut_idle" && det.Confidence > common.MinConfidence {
							builderAvailable = true
							break
						}
					}

					// Only bypass cooldown if last build was successful AND builder is available
					if builderAvailable && m.lastBuildSuccessful[instance.ID] {
						log.Printf("[%s] Builder is idle and last build was successful, running build order despite cooldown", instance.ID)
						// Continue with task execution
					} else {
						// No idle builder or last build failed, honor the cooldown
						continue
					}
				} else {
					// Not a build order task, honor the cooldown
					continue
				}
			}

			// Skip if task requires city view
			if hasRequirement {
				// Skip city-specific tasks
				isFieldTask := false

				// Tasks that work in field view:
				switch task.Name {
				case "manage_scouts", "farm_barbarians", "challenge_barbarians", "return_to_city":
					isFieldTask = true
				}

				if !isFieldTask {
					continue
				}

				// Check other requirements
				if !requirement.IsMet(detections) {
					continue
				}
			}

			// Execute field-appropriate task
			if executed := task.Handler(
				instance.DeviceID,
				gameView,
				detections,
				m.Config.Gameplay.ADBPath,
				task.Config,
				instance.State, // Pass state for tracking building levels
			); executed {
				log.Printf("[%s] Executed field task: %s", instance.ID, task.Name)
				task.LastExecuted = time.Now()
				fieldTaskExecuted = true
				return // Return to get fresh state
			}
		}

		// If no field tasks were executed, try to return to city
		if !fieldTaskExecuted {
			// Find return to city button
			for _, det := range detections {
				if det.Class == "on_field" && det.Confidence > common.MinConfidence {
					log.Printf("[%s] No field tasks to execute, returning to city", instance.ID)

					if err := utils.TapScreen(instance.DeviceID, m.Config.Gameplay.ADBPath,
						int(det.X), int(det.Y)); err != nil {
						log.Printf("[%s] Error tapping return to city button: %v", instance.ID, err)
					} else {
						log.Printf("[%s] Sleeping 1 second to return to city", instance.ID)
						time.Sleep(1 * time.Second)
						return // Return to get fresh state after navigation
					}
					break
				}
			}

			log.Printf("[%s] In field view but couldn't find return button", instance.ID)
			return
		}
	}

	// If we're in city view or couldn't handle field view, proceed with city tasks
	log.Printf("[%s] Processing city tasks", instance.ID)

	// Execute tasks in priority order
	for i := range instance.Tasks {
		task := &instance.Tasks[i]

		// Skip if task is on cooldown - BUT make exception for build_order when builders are idle
		if time.Since(task.LastExecuted) < time.Duration(task.CooldownSec)*time.Second {
			// Check if this is the build order task and a builder is available and last build was successful
			if task.Name == "process_build_order" {
				builderAvailable := false
				for _, det := range detections {
					if det.Class == "builders_hut_idle" && det.Confidence > common.MinConfidence {
						builderAvailable = true
						break
					}
				}

				// Only bypass cooldown if last build was successful AND builder is available
				if builderAvailable && m.lastBuildSuccessful[instance.ID] {
					log.Printf("[%s] Builder is idle and last build was successful, running build order despite cooldown", instance.ID)
					// Continue with task execution
				} else {
					// No idle builder or last build failed, honor the cooldown
					continue
				}
			} else {
				// Not a build order task, honor the cooldown
				continue
			}
		}

		// Check if detection requirements are met for this task
		requirement, hasRequirement := taskRequirements[task.Name]
		if hasRequirement && !requirement.IsMet(detections) {
			// Skip this task as its detection requirements aren't met
			continue
		}

		log.Printf("[%s] Executing task: %s", instance.ID, task.Name)

		// Execute task with state parameter
		if executed := task.Handler(
			instance.DeviceID,
			gameView,
			detections,
			m.Config.Gameplay.ADBPath,
			task.Config,
			instance.State, // Pass state for tracking building levels
		); executed {
			log.Printf("[%s] Executed task: %s", instance.ID, task.Name)
			task.LastExecuted = time.Now()

			// Save state immediately after executing a building-related task
			if task.Name == "process_build_order" {
				m.lastBuildSuccessful[instance.ID] = true

				if err := m.SaveInstanceStates(); err != nil {
					log.Printf("[%s] Error saving state after building task: %v", instance.ID, err)
				}
			}

			return // Return to get fresh state
		} else {
			// If this was a build task that failed, update our tracking
			if task.Name == "process_build_order" {
				m.lastBuildSuccessful[instance.ID] = false
			}
		}
	}
}

// runStartupTasks handles one-time startup tasks that don't require vision
func (m *Manager) runStartupTasks(instance *Instance) bool {
	// Get startup tasks from config
	startupTasks := m.Config.Gameplay.StartupTasks
	log.Printf("[%s] Running startup tasks: %v", instance.ID, startupTasks)

	// Create empty config for tasks that don't need specific config
	emptyConfig := common.TaskConfig{}

	// Process the tasks SEQUENTIALLY - focus on one task until it's complete
	for _, taskName := range startupTasks {
		switch taskName {
		case "clear_trees":
			// Tree clearing is implemented elsewhere and works fine
			if !instance.State.TreeClearingCompleted {
				if executed := actions.ClearTrees(
					instance.DeviceID,
					"",
					nil,
					m.Config.Gameplay.ADBPath,
					emptyConfig,
					instance.State,
				); executed {
					log.Printf("[%s] Executed startup task: clear_trees", instance.ID)
					return false
				}

				if actions.IsTreeClearingComplete() {
					instance.State.TreeClearingCompleted = true
					log.Printf("[%s] Startup task completed: clear_trees", instance.ID)
				} else {
					return false
				}
			}

		case "recruit_second_builder":
			log.Printf("[%s] Checking recruit_second_builder task conditions", instance.ID)

			// Only start this task if tree clearing is complete and second builder not added
			if instance.State.TreeClearingCompleted && !instance.State.SecondBuilderAdded {
				log.Printf("[%s] Conditions met for recruit_second_builder task", instance.ID)

				// Take a fresh screenshot for second builder task
				log.Printf("[%s] Taking fresh screenshot for second builder task", instance.ID)
				screenshot, err := vision.CaptureScreenshot(instance.DeviceID, m.Config.Gameplay.ADBPath)
				if err != nil {
					log.Printf("[%s] Error capturing screenshot for second builder: %v", instance.ID, err)
					return false
				}

				// Save screenshot for debugging
				debugPath := fmt.Sprintf("screenshots/debug_builder_%s.png", time.Now().Format("20060102_150405"))
				if err := vision.SaveScreenshot(screenshot, debugPath); err != nil {
					log.Printf("[%s] Failed to save debug screenshot: %v", instance.ID, err)
				} else {
					log.Printf("[%s] Saved debug screenshot to %s", instance.ID, debugPath)
				}

				// Get current game state and view
				log.Printf("[%s] Analyzing game state for second builder", instance.ID)
				gameView, detections, err := vision.AnalyzeGameState(
					screenshot,
					m.Config.Global.RoboflowAPIKey,
					m.Config.Global.RoboflowGameplayModel,
				)
				if err != nil {
					log.Printf("[%s] Error analyzing game state for second builder: %v", instance.ID, err)
					return false
				}

				// Log detailed detection info
				log.Printf("[%s] Game view detected: '%s' with %d objects for second builder",
					instance.ID, gameView, len(detections))

				for i, det := range detections {
					if det.Confidence > common.MinConfidence {
						log.Printf("[%s] Detection %d: %s (%.2f) at (%.1f, %.1f) %.0fx%.0f",
							instance.ID, i+1, det.Class, det.Confidence, det.X, det.Y, det.Width, det.Height)
					}
				}

				// Check if we have proper detections
				if len(detections) == 0 || gameView == "" {
					log.Printf("[%s] No detections found, cannot proceed with second builder task", instance.ID)
					return false
				}

				// Use the existing RecruitSecondBuilder function with the fresh detections
				if executed := actions.RecruitSecondBuilder(
					instance.DeviceID,
					gameView,
					detections,
					m.Config.Gameplay.ADBPath,
					emptyConfig,
					instance.State,
				); executed {
					log.Printf("[%s] Executed startup task: recruit_second_builder", instance.ID)
					return false
				}

				// Check if task is now complete via global state variable
				if actions.IsSecondBuilderAdded() {
					instance.State.SecondBuilderAdded = true
					log.Printf("[%s] Startup task completed: recruit_second_builder", instance.ID)
				} else {
					return false
				}
			} else {
				log.Printf("[%s] Conditions not met for recruit_second_builder: treesCleared=%v, builderAdded=%v",
					instance.ID, instance.State.TreeClearingCompleted, instance.State.SecondBuilderAdded)
			}

		default:
			log.Printf("[%s] Unknown startup task in config: %s", instance.ID, taskName)
		}
	}

	// All tasks have been completed if we reach here
	return instance.State.TreeClearingCompleted && instance.State.SecondBuilderAdded
}
