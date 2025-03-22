# RoboRok Python Setup Guide

This guide walks you through the complete setup process for RoboRok Python, from installing prerequisites to running your first automation session.

## Prerequisites

- Python 3.7 or higher
- BlueStacks or another Android emulator
- ADB tools (included with BlueStacks)
- Roboflow account (free tier works fine)
- Rise of Kingdoms installed in your emulator

## Step 1: Install BlueStacks

1. Download BlueStacks from [bluestacks.com](https://www.bluestacks.com)
2. Follow the installation instructions for your operating system
3. Once installed, launch BlueStacks and install Rise of Kingdoms from the Play Store

## Step 2: Configure ADB Connection

1. Locate the ADB executable in your BlueStacks installation:
   - macOS: `/Applications/BlueStacks.app/Contents/MacOS/hd-adb`
   - Windows: `C:\Program Files\BlueStacks_nxt\HD-Adb.exe`

2. Connect to the BlueStacks instance:
   ```bash
   /Applications/BlueStacks.app/Contents/MacOS/hd-adb connect 127.0.0.1:5555
   ```

3. Verify the connection:
   ```bash
   /Applications/BlueStacks.app/Contents/MacOS/hd-adb devices
   ```
   
   You should see output like this:
   ```
   List of devices attached
   127.0.0.1:5555 device
   ```

4. Test that you can capture screenshots:
   ```bash
   # Save screenshot to file
   /Applications/BlueStacks.app/Contents/MacOS/hd-adb -s 127.0.0.1:5555 exec-out screencap -p > screenshot.png
   
   # Or view directly on macOS
   /Applications/BlueStacks.app/Contents/MacOS/hd-adb -s 127.0.0.1:5555 exec-out screencap -p | open -a Preview.app -f
   ```

## Step 3: Setup Roboflow and Get API Key

1. Create a Roboflow account at [roboflow.com](https://roboflow.com) if you haven't already
2. Create and train your models as described in [TRAINING_DATA.md](./TRAINING_DATA.md)
3. To find your API key, go to the workspace settings or API Keys section
4. Copy your API key (you'll need it for the config file)

## Step 4: Set Up Python Environment

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/roborok.git
   cd roborok
   ```

2. Set up a virtual environment:
   ```bash
   # Create a virtual environment
   python -m venv venv
   
   # Activate the virtual environment
   # On macOS/Linux:
   source venv/bin/activate
   # On Windows:
   venv\Scripts\activate
   ```

3. Install dependencies:
   ```bash
   pip install -r requirements.txt
   ```

4. Install the package in development mode:
   ```bash
   pip install -e .
   ```

## Step 5: Configure RoboRok

1. Create your configuration file:
   ```bash
   cp example_configs/basic_config.json config.json
   ```

2. Open and edit `config.json` with your details:
   ```json
   {
     "global": {
       "roboflow_api_key": "YOUR_API_KEY_HERE",
       "roboflow_tutorial_model_id": "rok_tutorial/1",
       "roboflow_gameplay_model_id": "rok_gameplay/1",
       "refresh_interval_ms": 1000,
       "report_endpoint": "http://localhost:3000/api/stats",
       "reporting_interval_s": 300
     },
     "instances": {
       "instance1": {
         "device_id": "127.0.0.1:5555",
         "preferred_civilization": "china"
       }
     },
     "gameplay": {
       "adb_path": "/Applications/BlueStacks.app/Contents/MacOS/hd-adb",
       "startup_tasks": [
         "clear_trees",
         "recruit_second_builder"
       ]
     }
   }
   ```

## Step 6: Run RoboRok

1. Run the main module:
   ```bash
   python -m roborok.main --config config.json
   ```

2. Additional command-line options:
   ```bash
   # Specify a state file for persistence
   python -m roborok.main --config config.json --state instance_states.json
   
   # Specify a specific instance to run
   python -m roborok.main --config config.json --instance instance1
   
   # Skip tutorial
   python -m roborok.main --config config.json --skip-tutorial
   
   # Skip tree clearing
   python -m roborok.main --config config.json --skip-tree-clearing
   
   # Skip second builder recruitment
   python -m roborok.main --config config.json --skip-second-builder
   
   # Run a specific number of cycles
   python -m roborok.main --config config.json --cycles 10
   ```

## Troubleshooting

### ADB Connection Issues

If you have trouble connecting to BlueStacks:

1. Make sure BlueStacks is running
2. Try restarting the ADB server:
   ```bash
   /Applications/BlueStacks.app/Contents/MacOS/hd-adb kill-server
   /Applications/BlueStacks.app/Contents/MacOS/hd-adb start-server
   ```
3. Reconnect to the device:
   ```bash
   /Applications/BlueStacks.app/Contents/MacOS/hd-adb connect 127.0.0.1:5555
   ```

### Python Environment Issues

If you encounter Python environment issues:

1. Make sure you've activated your virtual environment
2. Verify that all dependencies are installed:
   ```bash
   pip install -r requirements.txt
   ```
3. Check for error messages in the console output

### Model Detection Issues

If the computer vision detection isn't working correctly:

1. Verify your API key is correct
2. Check that your model IDs are entered correctly
3. Make sure you have internet connectivity
4. Test your models manually in the Roboflow interface
5. Check console logs for any errors

### Common Errors

1. **ImportError**: Make sure all imports are correct and point to the right modules
2. **Missing Dependencies**: Run `pip install -r requirements.txt` to ensure all dependencies are installed
3. **ADB Connection Failure**: Verify that BlueStacks is running and properly connected
4. **Roboflow API Errors**: Check your API key and internet connection
5. **OCR Issues**: EasyOCR may require additional dependencies on some systems

## Understanding the Code

The RoboRok system is built around several key components:

1. **State Machine**: The central brain of the system, controlling transitions between different game states
2. **Vision System**: Integrates with Roboflow to detect game elements
3. **Action System**: Implements specific game actions based on detected state
4. **Task Management**: Prioritizes and executes tasks based on game state
5. **Models**: Data structures for representing game state and elements

To understand the codebase better:

1. Start with `main.py` to see the overall execution flow
2. Examine `state_machine/state_machine.py` to understand state transitions
3. Look at `actions/actions.py` to see how game actions are implemented
4. Check out `vision/roboflow.py` to understand the computer vision integration
