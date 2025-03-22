# roborok/roboflow.py
"""Roboflow API integration"""

import requests
import json
import logging
import time
from typing import List, Dict, Any, Optional

from roborok.models import Detection
from roborok.vision.screenshot import capture_screenshot
from roborok.utils.adb import tap_screen, swipe_screen

def send_to_roboflow(image_bytes, api_key, model_id):
    """
    Send an image to Roboflow for detection
    
    Args:
        image_bytes: Screenshot image data as bytes
        api_key: Roboflow API key
        model_id: Roboflow model ID (format: project_name/version_number)
        
    Returns:
        Roboflow API response JSON if successful, None if failed
    """
    url = f"https://detect.roboflow.com/{model_id}?api_key={api_key}"
    
    try:
        # Create files dict for the multipart request
        files = {'file': ('screenshot.jpg', image_bytes, 'image/jpeg')}
        
        # Send the request
        response = requests.post(url, files=files, timeout=10)
        
        # Raise an exception for bad status codes
        response.raise_for_status()
        
        # Parse JSON response
        return response.json()
    except Exception as e:
        print(f"Error sending image to Roboflow: {e}")
        return None

def parse_detections(roboflow_response):
    """
    Parse Roboflow API response into Detection objects
    
    Args:
        roboflow_response: JSON response from Roboflow API
        
    Returns:
        List of Detection objects
    """
    detections = []
    
    if not roboflow_response or 'predictions' not in roboflow_response:
        return detections
    
    # Parse each prediction into a Detection object
    for pred in roboflow_response['predictions']:
        detection = Detection(
            class_name=pred['class'],
            x=pred['x'],
            y=pred['y'],
            width=pred['width'],
            height=pred['height'],
            confidence=pred['confidence']
        )
        detections.append(detection)
    
    return detections

def detect_game_elements(screenshot_bytes, api_key, model_id):
    """
    Process a screenshot and detect game elements
    
    Args:
        screenshot_bytes: Screenshot image data
        api_key: Roboflow API key
        model_id: Roboflow model ID
        
    Returns:
        Tuple of (List[Detection], error_message)
    """
    # Send to Roboflow
    response = send_to_roboflow(screenshot_bytes, api_key, model_id)
    
    if not response:
        return [], "Failed to get response from Roboflow"
    
    # Parse detections
    detections = parse_detections(response)
    
    # Log detection count
    print(f"Detected {len(detections)} game elements")
    
    return detections, None

def determine_game_view(detections):
    """
    Determine if we're in city view, map view, etc.
    
    Args:
        detections: List of Detection objects
        
    Returns:
        String indicating the view ("city", "map", "unknown")
    """
    # Check for explicit view indicators
    for detection in detections:
        if detection.class_name == "in_city" and detection.is_confident():
            return "city"
        if detection.class_name == "on_map" and detection.is_confident():
            return "map"
    
    # Check for view-specific elements
    city_indicators = 0
    map_indicators = 0
    
    for detection in detections:
        # City view indicators
        if detection.class_name in [
            "city_hall", "barracks", "farm", "storehouse"
        ] and detection.is_confident():
            city_indicators += 1
        
        # Map view indicators
        if detection.class_name in [
            "return_to_city_button", "map_button", "barbarian"
        ] and detection.is_confident():
            map_indicators += 1
    
    # Determine view based on indicators
    if city_indicators > map_indicators:
        return "city"
    elif map_indicators > 0:
        return "map"
    else:
        return "unknown"

def capture_and_detect(device_id: str, adb_path: str, api_key: str, model_id: str, max_attempts: int = 6) -> List[Detection]:
    """
    Take a screenshot, handle all high-priority clickables in order of priority, and return clean detections.
    
    Args:
        device_id: Device ID
        adb_path: Path to ADB executable
        api_key: Roboflow API key
        model_id: Roboflow model ID
        max_attempts: Maximum number of attempts to handle high-priority elements
        
    Returns:
        List of detections after handling all high-priority elements
    """
    logger = logging.getLogger("actions.capture_and_detect")
    
    # Define high-priority clickable elements (in order of priority)
    high_priority_elements = [
        "help_chat_bubble",  # Handle help bubbles first as they block interaction
        "farm_clickable", 
        "lumber_mill_clickable",
        "alliance_help_request",
        # "alliance_invite_join",  # Removed from direct click list - will handle specially
        # Add more elements here in order of priority
    ]
    
    for attempt in range(max_attempts):
        # Take a screenshot
        screenshot_bytes = capture_screenshot(device_id, adb_path)
        if not screenshot_bytes:
            logger.error(f"Failed to capture screenshot (attempt {attempt+1}/{max_attempts})")
            time.sleep(0.5)
            continue
        
        # Get detections
        detections, error = detect_game_elements(screenshot_bytes, api_key, model_id)
        if error:
            logger.error(f"Error getting detections (attempt {attempt+1}/{max_attempts}): {error}")
            time.sleep(0.5)
            continue
        
        # Special case: Check for alliance invite dialog
        has_alliance_invite = False
        has_exit_button = False
        exit_button = None
        
        for det in detections:
            if det.class_name == "alliance_invite_join" and det.is_confident(0.4):
                has_alliance_invite = True
            elif det.class_name == "exit_dialog_button" and det.is_confident(0.4):
                has_exit_button = True
                exit_button = det
        
        # If we have both alliance invite and exit button, click the exit button
        if has_alliance_invite and has_exit_button and exit_button is not None:
            logger.info(f"Found alliance invite dialog, clicking exit button at ({exit_button.x}, {exit_button.y})")
            tap_screen(device_id, adb_path, int(exit_button.x), int(exit_button.y))
            time.sleep(0.5)  # Wait for action to register
            continue  # Get fresh detections
            
        # Check for other high-priority clickables in order of priority
        high_priority_found = False
        
        # First, collect all high-priority elements present
        priority_detections = []
        for priority_class in high_priority_elements:
            for det in detections:
                if det.class_name == priority_class and det.is_confident(0.4):
                    priority_detections.append(det)
        
        # If we found any high-priority elements, click the highest priority one
        if priority_detections:
            # Sort by priority (order in the high_priority_elements list)
            sorted_priorities = sorted(
                priority_detections, 
                key=lambda det: high_priority_elements.index(det.class_name)
            )
            
            # Click the highest priority element
            top_priority = sorted_priorities[0]
            logger.info(f"Found high-priority element {top_priority.class_name} at ({top_priority.x}, {top_priority.y}), clicking")
            tap_screen(device_id, adb_path, int(top_priority.x), int(top_priority.y))
            time.sleep(0.5)  # Wait for action to register
            
            # If we found and handled a high-priority element, try again to get fresh detections
            high_priority_found = True
        
        if high_priority_found:
            logger.info("Handled high-priority element, getting fresh detections")
            continue
            
        # If no high-priority elements found, return the detections
        return detections
    
    # If we've exhausted all attempts
    logger.info(f"Completed {max_attempts} rounds of checking for high-priority elements")
    return detections