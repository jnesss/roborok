# Rise of Kingdoms Automation with Roboflow Vision AI

A sophisticated automation system for Rise of Kingdoms that uses Roboflow's computer vision capabilities to intelligently play the start of the game. I created this project to experiment with computer vision AI integrated with game automation.  This initial version automates the laborious ROK tutorial at the beginning and builds the first few buildings.  

## Demo Video
[Watch the demonstration video here](https://youtu.be/your-video-id)

## Features
- **Computer Vision Integration**: Uses Roboflow's vision AI to detect game elements and make decisions
- **Multi-Instance Management**: Control multiple game instances with a single application
- **Tutorial Automation**: Automatically completes the tutorial with your preferred civilization
- **Building Optimization**: Smart building placement and upgrade sequencing
- **Resource Management**: Efficiently harvests resources and completes tasks

## Technical Overview
This project is built using:
- Go programming language
- Roboflow Vision AI
- Android Debug Bridge (ADB) for device control
- BlueStacks for game emulation

The architecture follows a modular design with these key components:
- **Instance Manager**: Coordinates multiple game instances
- **Vision System**: Integrates with Roboflow for visual recognition
- **Action System**: Implements game actions (upgrading, training, etc.)
- **State Management**: Tracks and persists game state

## Getting Started

### Prerequisites
- Go 1.16 or higher
- BlueStacks or another Android emulator
- ADB tools
- Roboflow account

### Configuration
1. Clone this repository
2. Copy `config.example.json` to `config.json` and update with your settings
3. Build the application with `go build`

### Running the Automation
```bash
./roborok