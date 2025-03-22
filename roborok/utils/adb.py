# roborok/utils/adb.py
import subprocess
import logging
from typing import Optional

def tap_screen(device_id: str, adb_path: str, x: int, y: int) -> bool:
    """
    Simulates a tap on the screen
    
    Args:
        device_id: The device ID
        adb_path: Path to the ADB executable
        x: X coordinate
        y: Y coordinate
        
    Returns:
        True if successful, False otherwise
    """
    try:
        subprocess.run(
            [adb_path, "-s", device_id, "shell", "input", "tap", str(x), str(y)],
            check=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE
        )
        return True
    except subprocess.SubprocessError as e:
        logging.error(f"Error tapping screen at ({x}, {y}): {e}")
        return False

def swipe_screen(device_id: str, adb_path: str, x1: int, y1: int, x2: int, y2: int, duration_ms: int) -> bool:
    """
    Simulates a swipe from (x1, y1) to (x2, y2) with the given duration
    
    Args:
        device_id: The device ID
        adb_path: Path to the ADB executable
        x1: Starting X coordinate
        y1: Starting Y coordinate
        x2: Ending X coordinate
        y2: Ending Y coordinate
        duration_ms: Duration of the swipe in milliseconds
        
    Returns:
        True if successful, False otherwise
    """
    try:
        subprocess.run(
            [
                adb_path, "-s", device_id, "shell", "input", "swipe",
                str(x1), str(y1), str(x2), str(y2), str(duration_ms)
            ],
            check=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE
        )
        return True
    except subprocess.SubprocessError as e:
        logging.error(f"Error swiping screen from ({x1}, {y1}) to ({x2}, {y2}): {e}")
        return False