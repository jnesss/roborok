# roborok/config.py
"""Configuration utilities for RoboRok"""

import json
import os

from roborok.utils.constants import DEFAULT_ROBOFLOW_API_KEY


# Global config variable
_CONFIG = None

def load_config(filepath):
    """
    Load configuration from a JSON file
    
    Args:
        filepath: Path to the configuration file
        
    Returns:
        The configuration as a dictionary
    """
    global _CONFIG
    
    try:
        with open(filepath, 'r') as f:
            _CONFIG = json.load(f)
        return _CONFIG
    except Exception as e:
        print(f"Error loading configuration: {e}")
        return None

def get_config():
    """
    Get the current configuration
    
    Returns:
        The current configuration
    """
    global _CONFIG
    if _CONFIG is None:
        raise RuntimeError("Configuration not loaded. Call load_config() first.")
    return _CONFIG

def get_roboflow_api_key():
    """Get the Roboflow API key from the config"""
    return get_config().get("global", {}).get("roboflow_api_key", DEFAULT_ROBOFLOW_API_KEY)