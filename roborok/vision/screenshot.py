# roborok/screenshot.py
"""Screenshot capture and processing utilities"""

import os
import subprocess
import time
import logging
import numpy as np

from datetime import datetime
from pathlib import Path
from PIL import Image
import io

def capture_screenshot(device_id, adb_path):
    """
    Capture a screenshot from the device using ADB
    
    Args:
        device_id: The device ID to capture from
        adb_path: Path to the ADB executable
        
    Returns:
        Image bytes if successful, None if failed
    """
    try:
        # Ensure the ADB command works properly with the device
        cmd = [adb_path, "-s", device_id, "exec-out", "screencap", "-p"]
        result = subprocess.run(cmd, capture_output=True, check=True)
        return result.stdout
    except subprocess.SubprocessError as e:
        print(f"Error capturing screenshot: {e}")
        return None

def save_screenshot(screenshot_bytes, filename=None):
    """
    Save a screenshot to disk
    
    Args:
        screenshot_bytes: The screenshot data as bytes
        filename: Optional filename, if None will generate based on timestamp
        
    Returns:
        Path to the saved screenshot
    """
    if filename is None:
        # Create a filename based on timestamp
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        filename = f"screenshot_{timestamp}.png"
    
    # Ensure the screenshots directory exists
    screenshots_dir = "screenshots"
    os.makedirs(screenshots_dir, exist_ok=True)
    
    # Full path to the screenshot
    filepath = os.path.join(screenshots_dir, filename)
    
    try:
        with open(filepath, "wb") as f:
            f.write(screenshot_bytes)
        return filepath
    except Exception as e:
        print(f"Error saving screenshot: {e}")
        return None

def get_image_from_bytes(screenshot_bytes):
    """
    Convert screenshot bytes to a PIL Image
    
    Args:
        screenshot_bytes: The screenshot data as bytes
        
    Returns:
        PIL Image if successful, None if failed
    """
    try:
        return Image.open(io.BytesIO(screenshot_bytes))
    except Exception as e:
        print(f"Error converting screenshot to image: {e}")
        return None

def crop_image(image, crop_box):
    """
    Crop an image to the specified box
    
    Args:
        image: PIL Image to crop
        crop_box: Tuple of (left, top, right, bottom)
        
    Returns:
        Cropped PIL Image
    """
    return image.crop(crop_box)
    
def ocr_region(image, x1, y1, x2, y2):
    """
    Run OCR on a specific region of the image
    
    Args:
        image: PIL Image object
        x1, y1, x2, y2: Bounding box coordinates
        
    Returns:
        Text found in the region
    """
    try:
        import easyocr
        reader = easyocr.Reader(['en'])
        
        # Crop the image to the specified region
        cropped_img = image.crop((x1, y1, x2, y2))
        
        # For debugging
        # cropped_img.save(f"ocr_region_{x1}_{y1}.png")
        
        # Run OCR on the cropped image
        results = reader.readtext(np.array(cropped_img))
        
        # Combine all found text
        text = " ".join([r[1] for r in results])
        return text
    except Exception as e:
        import logging
        logging.error(f"Error during OCR: {e}")
        return ""
