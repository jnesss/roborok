# roborok/constants.py
"""Constants for the RoboRok application"""

# Default API key placeholder - will be overridden by config
DEFAULT_ROBOFLOW_API_KEY = "YOUR_ROBOFLOW_API_KEY"

# Model IDs for different phases
TUTORIAL_MODEL_ID = "rok_tutorial/7"
GAMEPLAY_MODEL_ID = "rok_gameplay/1"

# Confidence thresholds
MIN_CONFIDENCE = 0.7

# File paths
DEFAULT_STATE_FILE_PATH = "instance_states.json"
DEFAULT_CONFIG_FILE_PATH = "config.json"
SCREENSHOTS_DIR_PATH = "screenshots"

# Game positions
HOME_BUTTON_X = 31
HOME_BUTTON_Y = 450