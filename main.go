package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"roborok/internal/manager"
	"roborok/internal/report"
	"roborok/internal/utils"
	"strings"
	"syscall"
	"time"
)

func main() {
	log.Println("Starting Rise of Kingdoms Automation...")

	// Ensure screenshots directory exists
	os.MkdirAll("screenshots", 0755)

	// Initialize global configuration
	configPath := "config.json"
	if err := utils.InitGlobalConfig(configPath); err != nil {
		log.Fatalf("Error initializing global configuration: %v", err)
	}

	// Get the initialized config
	config := utils.GetConfig()

	// Initialize reporter (placeholder for now)
	reporter := report.NewReporter(config.Global.ReportEndpoint)
	go reporter.Start()

	// Initialize instance manager
	mgr := manager.NewManager(config, reporter)

	// Load saved instance states
	if err := mgr.LoadInstanceStates(filepath.Join(".", "instance_states.json")); err != nil {
		log.Printf("Warning: Could not load instance states: %v", err)
		log.Println("Initializing with default states...")
	}

	// Setup signal handler for graceful shutdown
	setupSignalHandler(mgr)

	// Start console command monitor in a separate goroutine
	go monitorCommands(mgr)

	// Start all instances in parallel
	for id, instance := range mgr.Instances {
		go mgr.RunInstanceLoop(id, instance)
	}

	// Log success
	log.Printf("Started automation for %d instances", len(mgr.Instances))

	// Keep the main process alive
	for {
		time.Sleep(time.Hour)
	}
}

// setupSignalHandler creates a handler for graceful shutdown
func setupSignalHandler(mgr *manager.Manager) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("Shutdown signal received, saving state...")
		if err := mgr.SaveInstanceStates(); err != nil {
			log.Printf("Error saving state on shutdown: %v", err)
		}
		os.Exit(0)
	}()
}

// monitorCommands processes user input commands for controlling the automation
func monitorCommands(mgr *manager.Manager) {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("\n=== Command Interface ===")
	fmt.Println("Available commands:")
	fmt.Println("  p - Pause automation")
	fmt.Println("  r - Resume automation")
	fmt.Println("  s - Show status")
	fmt.Println("  t60 - Pause for 60 seconds (change number as needed)")
	fmt.Println("  q - Quit")
	fmt.Println("  h - Show this help message")

	for scanner.Scan() {
		cmd := strings.TrimSpace(scanner.Text())

		switch {
		case cmd == "p":
			mgr.Pause()
			fmt.Println("Automation paused. Type 'r' to resume.")

		case cmd == "r":
			mgr.Resume()
			fmt.Println("Automation resumed.")

		case cmd == "s":
			printStatus(mgr)

		case cmd == "h":
			printHelp()

		case cmd == "q":
			fmt.Println("Saving state and shutting down...")
			if err := mgr.SaveInstanceStates(); err != nil {
				log.Printf("Error saving state on quit: %v", err)
			}
			os.Exit(0)

		case strings.HasPrefix(cmd, "t") && len(cmd) > 1:
			// Parse time in seconds
			var seconds int
			_, err := fmt.Sscanf(cmd[1:], "%d", &seconds)
			if err != nil || seconds <= 0 {
				fmt.Println("Invalid time format. Use tXX where XX is seconds, e.g., t60")
				continue
			}

			fmt.Printf("Pausing automation for %d seconds...\n", seconds)
			mgr.Pause()

			// Start a goroutine to resume after the specified time
			go func() {
				time.Sleep(time.Duration(seconds) * time.Second)
				mgr.Resume()
				fmt.Printf("Time's up! Automation resumed after %d seconds.\n", seconds)
			}()

		default:
			fmt.Println("Unknown command. Type 'h' for help.")
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading command input: %v", err)
	}
}

// printStatus shows the current automation status
func printStatus(mgr *manager.Manager) {
	fmt.Println("\n=== Automation Status ===")
	fmt.Printf("Running: %v\n", !mgr.IsPaused())

	for id, instance := range mgr.Instances {
		fmt.Printf("\nInstance: %s\n", id)
		fmt.Printf("  Device ID: %s\n", instance.DeviceID)
		fmt.Printf("  City Hall Level: %d\n", instance.State.CityHallLevel)
		fmt.Printf("  Tutorial Completed: %v\n", instance.State.TutorialCompleted)
		fmt.Printf("  Startup Tasks Completed: %v\n", instance.State.StartupTasksCompleted)

		if !instance.State.StartupTasksCompleted {
			fmt.Printf("  Tree Clearing Completed: %v\n", instance.State.TreeClearingCompleted)
			fmt.Printf("  Second Builder Added: %v\n", instance.State.SecondBuilderAdded)
		}
	}

	fmt.Println("\nType 'h' for available commands.")
}

// printHelp displays the help message
func printHelp() {
	fmt.Println("\n=== Command Interface Help ===")
	fmt.Println("Available commands:")
	fmt.Println("  p - Pause automation")
	fmt.Println("  r - Resume automation")
	fmt.Println("  s - Show status")
	fmt.Println("  t60 - Pause for 60 seconds (change number as needed)")
	fmt.Println("  q - Quit")
	fmt.Println("  h - Show this help message")
	fmt.Println("\nWhile automation is running, you can use these commands")
	fmt.Println("to temporarily take control of the game.")
}
