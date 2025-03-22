# roborok/tutorial.py
"""Tutorial automation for first-time game setup"""

import logging
import time
import random
from typing import List, Dict, Optional, Any

from roborok.models import Detection, InstanceState
from roborok.utils.adb import tap_screen, swipe_screen

# Constants
MIN_CONFIDENCE = 0.7

# Global variables for tracking tutorial state
civilization_scroll_attempts = 0

# Tutorial state constants
TUTORIAL_STATE_UNKNOWN = "unknown"
TUTORIAL_STATE_SKIP_BUTTON = "skip_button"
TUTORIAL_STATE_COUNSELOR_TEXT = "counselor_text"
TUTORIAL_STATE_CIVILIZATION_SELECT = "civilization_select"
TUTORIAL_STATE_CONFIRM_BUTTON = "confirm_button"
TUTORIAL_STATE_ARROW_AND_TARGET = "arrow_and_target"
TUTORIAL_STATE_ARROW_ONLY = "arrow_only"
TUTORIAL_STATE_UPGRADE_COMPLETE = "upgrade_complete"
TUTORIAL_STATE_FINAL_ARROW = "final_arrow"

def is_tutorial_complete(instance_state: InstanceState) -> bool:
    """
    Check if the tutorial is completed
    
    Args:
        instance_state: Current instance state
        
    Returns:
        True if tutorial is completed, False otherwise
    """
    # If we've already marked tutorial as completed, don't re-check
    if instance_state.tutorial_completed:
        return True
        
    # Check if we've tracked both steps in the completion sequence
    if instance_state.tutorial_upgrade_complete_clicked and instance_state.tutorial_final_arrow_clicked:
        # Tutorial is complete if both steps have been done
        instance_state.tutorial_completed = True
        return True
        
    # Tutorial is not complete yet
    return False

def determine_tutorial_state(
    detections: List[Detection], 
    preferred_civilization: str,
    instance_state: InstanceState
) -> str:
    """
    Determine the current tutorial state based on detections
    
    Args:
        detections: List of detections
        preferred_civilization: Preferred civilization
        instance_state: Current instance state
        
    Returns:
        Tutorial state as string
    """
    # Look for any arrow and target combination
    has_arrow = False
    has_target = False
    
    for detection in detections:
        if detection.class_name == "click_arrow" and detection.is_confident():
            has_arrow = True
        if detection.class_name == "click_target" and detection.is_confident():
            has_target = True
    
    # If we've already clicked upgrade_complete but not the final arrow,
    # prioritize looking for ANY click_arrow + click_target combination
    if instance_state.tutorial_upgrade_complete_clicked and not instance_state.tutorial_final_arrow_clicked:
        if has_arrow and has_target:
            return TUTORIAL_STATE_FINAL_ARROW
    
    # If we haven't yet clicked upgrade_complete, prioritize finding it
    if not instance_state.tutorial_upgrade_complete_clicked:
        for detection in detections:
            if detection.class_name == "upgrade_complete" and detection.is_confident():
                return TUTORIAL_STATE_UPGRADE_COMPLETE
    
    # Standard tutorial state detection follows below
    if has_arrow and has_target:
        return TUTORIAL_STATE_ARROW_AND_TARGET
    
    # Check for skip button
    for detection in detections:
        if detection.class_name == "skip button" and detection.is_confident():
            return TUTORIAL_STATE_SKIP_BUTTON
    
    # Check for counselor text
    for detection in detections:
        if detection.class_name == "counselor text bubble" and detection.is_confident():
            return TUTORIAL_STATE_COUNSELOR_TEXT
    
    # Check for civilization selection
    # Look for civilizations to determine if we're on that screen
    civ_count = 0
    for detection in detections:
        if is_civilization(detection.class_name):
            civ_count += 1
    
    # If we see multiple civilizations, we're likely on the selection screen
    if civ_count >= 3:
        return TUTORIAL_STATE_CIVILIZATION_SELECT
    
    # Check for confirm button
    for detection in detections:
        if detection.class_name == "confirm_button" and detection.is_confident():
            return TUTORIAL_STATE_CONFIRM_BUTTON
    
    if has_arrow:
        return TUTORIAL_STATE_ARROW_ONLY
    
    return TUTORIAL_STATE_UNKNOWN

def handle_tutorial_state(
    device_id: str,
    adb_path: str,
    detections: List[Detection],
    tutorial_state: str,
    preferred_civilization: str,
    instance_state: InstanceState
) -> bool:
    """
    Handle the current tutorial state
    
    Args:
        device_id: Device ID
        adb_path: Path to ADB executable
        detections: List of detections
        tutorial_state: Current tutorial state
        preferred_civilization: Preferred civilization
        instance_state: Current instance state
        
    Returns:
        True if an action was taken, False otherwise
    """
    if tutorial_state == TUTORIAL_STATE_UPGRADE_COMPLETE:
        if handle_upgrade_complete(device_id, adb_path, detections, instance_state):
            # Mark that we've clicked on upgrade complete
            instance_state.tutorial_upgrade_complete_clicked = True
            logging.info("Marked 'upgrade_complete' as clicked - looking for final arrow next")
            return True
        return False
        
    elif tutorial_state == TUTORIAL_STATE_FINAL_ARROW:
        if handle_final_arrow(device_id, adb_path, detections, instance_state):
            # Mark that we've clicked on the final arrow
            instance_state.tutorial_final_arrow_clicked = True
            logging.info("Marked final arrow as clicked - tutorial sequence complete!")
            instance_state.tutorial_completed = True
            return True
        return False
        
    elif tutorial_state == TUTORIAL_STATE_SKIP_BUTTON:
        return handle_skip_button(device_id, adb_path, detections)
        
    elif tutorial_state == TUTORIAL_STATE_COUNSELOR_TEXT:
        return handle_counselor_text(device_id, adb_path, detections)
        
    elif tutorial_state == TUTORIAL_STATE_CIVILIZATION_SELECT:
        return handle_civilization_selection(device_id, adb_path, detections, preferred_civilization)
        
    elif tutorial_state == TUTORIAL_STATE_CONFIRM_BUTTON:
        return handle_confirm_button(device_id, adb_path, detections)
        
    elif tutorial_state == TUTORIAL_STATE_ARROW_AND_TARGET:
        return handle_arrow_and_target(device_id, adb_path, detections)
        
    elif tutorial_state == TUTORIAL_STATE_ARROW_ONLY:
        return handle_arrow_only(device_id, adb_path, detections)
        
    # In unknown state, just return False
    return False

# Individual state handlers

def handle_skip_button(device_id: str, adb_path: str, detections: List[Detection]) -> bool:
    """Handle skip button state"""
    for detection in detections:
        if detection.class_name == "skip button" and detection.is_confident():
            logging.info("Found skip button - clicking...")
            if tap_screen(device_id, adb_path, int(detection.x), int(detection.y)):
                return True
    return False

def handle_counselor_text(device_id: str, adb_path: str, detections: List[Detection]) -> bool:
    """Handle counselor text state"""
    for detection in detections:
        if detection.class_name == "counselor text bubble" and detection.is_confident():
            logging.info("Found counselor text - clicking...")
            if tap_screen(device_id, adb_path, int(detection.x), int(detection.y)):
                return True
    return False

def handle_civilization_selection(
    device_id: str, 
    adb_path: str, 
    detections: List[Detection],
    preferred_civilization: str
) -> bool:
    """Handle civilization selection state"""
    global civilization_scroll_attempts
    
    # Check if our preferred civilization is selected
    selected_civ_class = preferred_civilization.lower() + "_selected"
    for detection in detections:
        if detection.class_name.lower() == selected_civ_class and detection.is_confident():
            logging.info(f"Found {detection.class_name} - preferred civilization selected")
            
            # Find and click the confirm button
            for btn in detections:
                if btn.class_name == "confirm_button" and btn.is_confident():
                    logging.info("Found confirm button - clicking...")
                    if tap_screen(device_id, adb_path, int(btn.x), int(btn.y)):
                        # Wait for confirmation
                        time.sleep(1)
                        return True
            return False
    
    # Count civilizations
    detected_civs = 0
    civ_detections = []
    
    for detection in detections:
        if is_civilization(detection.class_name):
            detected_civs += 1
            civ_detections.append(detection)
    
    logging.info(f"Counted {detected_civs} civilizations")
    
    # Make sure we have enough civilizations visible
    expected_min_civs = 6
    if detected_civs < expected_min_civs:
        logging.info(f"Only detected {detected_civs} civilizations, waiting for better view")
        return False
    
    # Look for the preferred civilization
    for detection in civ_detections:
        if detection.class_name.lower() == preferred_civilization.lower() and detection.confidence > 0.5:
            logging.info(f"Found {preferred_civilization} - clicking...")
            if tap_screen(device_id, adb_path, int(detection.x), int(detection.y)):
                # Wait for selection to take effect
                time.sleep(1)
                # Reset scroll attempts on success
                civilization_scroll_attempts = 0
                return True
    
    # If preferred civilization not found, try to scroll right
    max_scroll_attempts = 5
    
    # After reaching max scroll attempts, reset and start over
    if civilization_scroll_attempts >= max_scroll_attempts:
        logging.info(f"Reached maximum scroll attempts ({max_scroll_attempts}), resetting")
        civilization_scroll_attempts = 0
        time.sleep(1)
        return False
    
    logging.info(f"Preferred civilization '{preferred_civilization}' not found, scrolling")
    
    # Find rightmost and leftmost civilizations
    rightmost = None
    leftmost = None
    rightmost_x = 0
    leftmost_x = 9999
    
    for civ in civ_detections:
        if civ.x > rightmost_x:
            rightmost_x = civ.x
            rightmost = civ
        if civ.x < leftmost_x:
            leftmost_x = civ.x
            leftmost = civ
    
    if rightmost is not None and leftmost is not None:
        # Perform swipe from rightmost to leftmost civilization
        start_x = int(rightmost.x)
        start_y = int(rightmost.y)
        end_x = int(leftmost.x)
        end_y = int(leftmost.y)
        
        logging.info(f"Swiping from ({start_x},{start_y}) to ({end_x},{end_y})")
        if swipe_screen(device_id, adb_path, start_x, start_y, end_x, end_y, 300):
            civilization_scroll_attempts += 1
            # Wait after scrolling
            time.sleep(1)
            return True
    
    logging.info(f"Could not find suitable points to scroll")
    return False

def handle_confirm_button(device_id: str, adb_path: str, detections: List[Detection]) -> bool:
    """Handle confirm button state"""
    for detection in detections:
        if detection.class_name == "confirm_button" and detection.is_confident():
            logging.info("Found confirm button - clicking...")
            if tap_screen(device_id, adb_path, int(detection.x), int(detection.y)):
                return True
    return False

def handle_arrow_and_target(device_id: str, adb_path: str, detections: List[Detection]) -> bool:
    """Handle arrow and target state"""
    target = None
    
    for detection in detections:
        if detection.class_name == "click_target" and detection.is_confident():
            target = detection
            break
    
    if target is not None:
        logging.info("Found arrow and target - clicking target...")
        if tap_screen(device_id, adb_path, int(target.x), int(target.y)):
            return True
    
    return False

def handle_arrow_only(device_id: str, adb_path: str, detections: List[Detection]) -> bool:
    """Handle arrow only state"""
    arrow = None
    
    for detection in detections:
        if detection.class_name == "click_arrow" and detection.is_confident():
            arrow = detection
            break
    
    if arrow is not None:
        # For now, just tap center of screen when we see an arrow
        logging.info("Found arrow - tapping center of screen...")
        if tap_screen(device_id, adb_path, 320, 240):
            return True
    
    return False

def handle_upgrade_complete(
    device_id: str, 
    adb_path: str, 
    detections: List[Detection],
    instance_state: InstanceState
) -> bool:
    """Handle upgrade complete state"""
    for detection in detections:
        if detection.class_name == "upgrade_complete" and detection.is_confident():
            logging.info("Found 'upgrade_complete' notification - clicking center of screen")
            
            if tap_screen(device_id, adb_path, 240, 400):
                # Mark as clicked
                instance_state.tutorial_upgrade_complete_clicked = True
                logging.info("Marked 'upgrade_complete' as clicked - looking for final arrow next")
                
                # Wait for UI to update
                time.sleep(1)
                return True
    return False

def handle_final_arrow(
    device_id: str, 
    adb_path: str, 
    detections: List[Detection],
    instance_state: InstanceState
) -> bool:
    """Handle final arrow state"""
    # Find the best target to click (highest confidence)
    best_target = None
    best_confidence = 0
    
    # Look through all click_targets
    for detection in detections:
        if detection.class_name == "click_target" and detection.is_confident():
            if detection.confidence > best_confidence:
                best_target = detection
                best_confidence = detection.confidence
    
    # If we found a target, click it
    if best_target is not None:
        logging.info("Found final arrow/target - clicking to complete tutorial")
        if tap_screen(device_id, adb_path, int(best_target.x), int(best_target.y)):
            # Wait for the tutorial to fully complete
            time.sleep(1)
            return True
    
    return False

def is_civilization(class_name: str) -> bool:
    """Check if a class name is a civilization"""
    civilizations = [
        "arabia", "britain", "china", "egypt", "france",
        "germany", "greece", "japan", "korea", "maya",
        "rome", "spain", "vikings"
    ]
    return class_name.lower() in civilizations

def run_tutorial_automation(
    device_id: str,
    roboflow_api_key: str,
    roboflow_model_id: str,
    adb_path: str,
    preferred_civilization: str,
    instance_state: InstanceState
) -> bool:
    """
    Main tutorial automation function
    
    Args:
        device_id: Device ID
        roboflow_api_key: Roboflow API key
        roboflow_model_id: Roboflow model ID for tutorial
        adb_path: Path to ADB executable
        preferred_civilization: Preferred civilization
        instance_state: Current instance state
        
    Returns:
        True if tutorial was completed, False otherwise
    """
    from roborok.vision.screenshot import capture_screenshot
    from roborok.vision.roboflow import send_to_roboflow, parse_detections
    
    logging.info(f"Starting tutorial automation for device {device_id}")
    logging.info(f"Using civilization: {preferred_civilization}")
    
    # Initialize random for unsticking
    random.seed()
    
    # Tutorial timeout (10 minutes should be more than enough)
    tutorial_timeout = time.time() + (10 * 60)
    
    # Counters for tracking progress and detecting stuck states
    iteration_count = 0
    stuck_iteration_count = 0
    last_state = TUTORIAL_STATE_UNKNOWN
    stuck_state_count = 0
    
    # If we're in the same state for too many iterations, we might be stuck
    max_stuck_iterations = 20
    
    # Main tutorial automation loop - run until timeout or completion
    while time.time() < tutorial_timeout:
        iteration_count += 1
        
        # Every 50 iterations, check if tutorial is complete
        if iteration_count % 50 == 0:
            if is_tutorial_complete(instance_state):
                logging.info("Tutorial completed!")
                return True
        
        # Capture screenshot
        screenshot = capture_screenshot(device_id, adb_path)
        if screenshot is None:
            logging.error("Error capturing screenshot")
            time.sleep(0.5)
            continue
        
        # Send to Roboflow for analysis
        response = send_to_roboflow(screenshot, roboflow_api_key, roboflow_model_id)
        if response is None:
            logging.error("Error sending to Roboflow")
            time.sleep(0.5)
            continue
        
        # Parse detections
        detections = parse_detections(response)
        
        # Log occasional detection information
        if len(detections) > 0 and iteration_count % 10 == 0:
            logging.info(f"Detected {len(detections)} objects")
        
        # Determine the tutorial state
        tutorial_state = determine_tutorial_state(detections, preferred_civilization, instance_state)
        
        # Check if we're stuck in the same state
        if tutorial_state == last_state:
            stuck_state_count += 1
        else:
            stuck_state_count = 0
            last_state = tutorial_state
        
        # If we're stuck in the same state for too long, try a random tap
        if stuck_state_count > max_stuck_iterations:
            logging.info(f"Stuck in state {tutorial_state} for {stuck_state_count} iterations, trying to unstick...")
            
            # Try a random tap in the center area
            center_x = 200 + random.randint(0, 200)  # 200-400
            center_y = 200 + random.randint(0, 200)  # 200-400
            tap_screen(device_id, adb_path, center_x, center_y)
            
            # Reset stuck counter
            stuck_state_count = 0
            time.sleep(1)
            continue
        
        if tutorial_state != TUTORIAL_STATE_UNKNOWN:
            logging.info(f"Tutorial state: {tutorial_state}")
        
        # Handle the current state
        action_taken = handle_tutorial_state(
            device_id,
            adb_path,
            detections,
            tutorial_state,
            preferred_civilization,
            instance_state
        )
        
        if not action_taken:
            stuck_iteration_count += 1
            
            # If no action was taken for many iterations, try a different approach
            if stuck_iteration_count > 30:
                logging.info("No action taken for many iterations, checking for tutorial completion")
                
                # Check if tutorial is actually complete
                if is_tutorial_complete(instance_state):
                    logging.info("Tutorial was already completed!")
                    return True
                
                # Try tapping center of screen to dismiss any dialogs
                tap_screen(device_id, adb_path, 240, 400)
                stuck_iteration_count = 0
                time.sleep(1)
            else:
                # Only sleep if no action was taken
                time.sleep(0.5)
        else:
            # Reset stuck counter when action is taken
            stuck_iteration_count = 0
        
        # Check if we've completed both necessary steps in the sequence
        if instance_state.tutorial_upgrade_complete_clicked and instance_state.tutorial_final_arrow_clicked:
            logging.info("Detected complete tutorial sequence!")
            instance_state.tutorial_completed = True
            return True
    
    logging.info("Tutorial automation timed out")
    return False