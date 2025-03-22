# roborok/actions.py
"""Basic game automation actions"""

import time
import logging
from typing import List, Dict, Any, Tuple, Optional
from datetime import datetime, timedelta

from roborok.models import Detection, OCRResult, GameResources, InstanceState, BuildTask
from roborok.vision.screenshot import get_image_from_bytes
from roborok.vision.roboflow import capture_and_detect
from roborok.vision.ocr import process_game_text
from roborok.utils.adb import tap_screen

def extract_resources(ocr_results: List[OCRResult]) -> GameResources:
    """
    Extract game resources from OCR results
    
    Args:
        ocr_results: List of OCR results
        
    Returns:
        GameResources object
    """
    resources = GameResources()
    
    for result in ocr_results:
        if result.region_type == "power_text":
            resource_value = result.get_resource_value()
            if resource_value is not None:
                resources.power = resource_value
        elif result.region_type == "wood_text":
            resource_value = result.get_resource_value()
            if resource_value is not None:
                resources.wood = resource_value
        elif result.region_type == "food_text":
            resource_value = result.get_resource_value()
            if resource_value is not None:
                resources.food = resource_value
    
    return resources

def analyze_game_state(device_id: str, adb_path: str, api_key: str, model_id: str) -> Tuple[Optional[str], Optional[GameResources], Optional[List[Detection]], Optional[str]]:
    """
    Capture screenshot and analyze game state
    
    Args:
        device_id: Device ID
        adb_path: Path to ADB executable
        api_key: Roboflow API key
        model_id: Roboflow model ID
        
    Returns:
        Tuple of (game_view, resources, detections, error_message)
    """    
    
    # Detect game elements
    detections = capture_and_detect(device_id, adb_path, api_key, model_id, 4)
        
    # Process text regions with OCR
    # ocr_results = process_game_text(image, detections)
    
    # Extract resources
    # resources = extract_resources(ocr_results)
    
    # Determine game view (city, map, etc.)
    game_view = "unknown"
    for det in detections:
        if det.class_name == "in_city" and det.is_confident():
            game_view = "city"
            break
        elif det.class_name == "on_map" and det.is_confident():
            game_view = "map"
            break
    
    return game_view, None, detections, None

def perform_basic_action(device_id: str, adb_path: str, action_type: str, x: int, y: int) -> bool:
    """
    Perform a basic action like tapping the screen
    
    Args:
        device_id: Device ID
        adb_path: Path to ADB executable
        action_type: Type of action ("tap", "swipe", etc.)
        x: X coordinate
        y: Y coordinate
        
    Returns:
        True if successful, False otherwise
    """
    if action_type == "tap":
        return tap_screen(device_id, adb_path, x, y)
    
    # Add other action types as needed
    
    return False

def handle_build_order(
    device_id: str, 
    game_view: str,
    detections: List[Detection],
    adb_path: str,
    config: Any,
    instance_state: InstanceState
) -> bool:
    """
    Process the build order tasks
    """
    logger = logging.getLogger("actions.upgrade_building")
    
    # Initialize the build queue if necessary
    if not hasattr(instance_state, 'building_tasks') or not instance_state.building_tasks:
        logger.info("Initializing build queue for the first time")
        instance_state.initialize_build_queue()
    
    # Check if we've completed all building tasks
    if not hasattr(instance_state, 'current_task_index'):
        instance_state.current_task_index = 0
    
    # If we're at the end, check for skipped tasks
    if instance_state.current_task_index >= len(instance_state.building_tasks):
        logger.info("At end of task list, checking for any skipped tasks to retry")
        instance_state._reset_to_skipped_task(10)  # 10-second cooldown
    
    # If still at the end after checking skipped tasks, nothing to do
    if instance_state.current_task_index >= len(instance_state.building_tasks):
        logger.info("All building tasks completed, nothing to do")
        return False
    
    # Rest of the function remains the same
    return handle_upgrade_building(device_id, game_view, detections, adb_path, config, instance_state)
    

def handle_collect_quests(
    device_id: str, 
    game_view: str,
    detections: List[Detection],
    adb_path: str,
    config: Any,
    instance_state: InstanceState
) -> bool:
    """
    Handle collecting quest rewards
    
    Args:
        device_id: Device ID
        game_view: Current game view (city, map, etc.)
        detections: List of detections from current screen
        adb_path: Path to ADB executable
        config: Task configuration
        instance_state: Current instance state
        
    Returns:
        True if quests were collected, False otherwise
    """
    
    logger = logging.getLogger("actions.collect_quests")
    
    # First check if side_quest_claimable is present
    has_side_quest_claimable = False
    for detection in detections:
        if detection.class_name == "side_quest_claimable" and detection.is_confident():
            has_side_quest_claimable = True
            break
    
    # Check for claimable quests
    for detection in detections:
        if detection.class_name == "main_quest_claimable" and detection.is_confident():
            logger.info(f"Found claimable main quest at ({detection.x}, {detection.y})")
            
            # Tap on the quest
            if tap_screen(device_id, adb_path, int(detection.x), int(detection.y)):
                # Wait briefly for animation
                time.sleep(0.5)
                return True
                
        elif detection.class_name == "quests_claimable" and detection.is_confident() and not config.claim_only_main_quest:
            # Only proceed if side_quest_claimable is also present
            if has_side_quest_claimable:
                # For side quests, we need to click 78 pixels below the detection point
                click_x = int(detection.x)
                click_y = int(detection.y + 78)  # Add the offset for the first quest in the list
                
                logger.info(f"Found claimable side quest indicator at ({detection.x}, {detection.y})")
                logger.info(f"Side quest is clickable, tapping the top side quest at ({click_x}, {click_y})")
                
                # Tap on the quest with offset
                if tap_screen(device_id, adb_path, click_x, click_y):
                    # Wait briefly for animation
                    time.sleep(0.5)
                    return True
            else:
                logger.info(f"Found quests_claimable but no side_quest_claimable present, ignoring")
                
    return False


# Update these constants based on the Go implementation
TREE_COORDINATES = [
    (134, 137), (195, 169), (261, 217), (243, 173), (292, 216),
    (325, 235), (333, 204), (371, 225), (378, 196), (415, 215),
    (479, 278), (590, 363), (521, 388), (574, 261), (352, 417),
    (368, 391), (299, 411), (203, 382), (176, 354)
]

# Matching harvest coordinates for each tree
HARVEST_COORDINATES = [
    (158, 256), (218, 256), (286, 256), (268, 257), (319, 269),
    (356, 292), (364, 249), (399, 275), (406, 244), (441, 269),
    (504, 324), (582, 415), (548, 429), (600, 314), (375, 467),
    (393, 439), (327, 462), (228, 426), (203, 406)
]

# Home button coordinates for resetting view
HOME_BUTTON_X = 31
HOME_BUTTON_Y = 450

# Global tracking variables
tree_index = 0
clearing_complete = False
second_builder_added = False
view_reset_done = False

def clear_trees(
    device_id: str, 
    game_view: str,
    detections: List[Detection],
    adb_path: str,
    config: Any,
    instance_state: InstanceState
) -> bool:
    """
    Handle clearing trees in the city using hardcoded coordinates
    
    Args:
        device_id: Device ID
        game_view: Current game view (city, map, etc.)
        detections: List of detections from current screen
        adb_path: Path to ADB executable
        config: Task configuration
        instance_state: Current instance state
        
    Returns:
        True if a tree was cleared, False otherwise
    """
    global tree_index, clearing_complete, view_reset_done
    
    logger = logging.getLogger("actions.clear_trees")
    
    # Skip if tree clearing was already completed
    if clearing_complete or instance_state.tree_clearing_completed:
        logger.info("Tree clearing already completed")
        return False
    
    # Initial view reset when we start tree clearing (only once)
    if tree_index == 0:
        logger.info("Initial view reset before starting tree clearing...")
        
        # First tap on home button (to ensure we get to field view)
        if not tap_screen(device_id, adb_path, HOME_BUTTON_X, HOME_BUTTON_Y):
            logger.error("Error tapping home button (first tap)")
            return False
        
        # Wait for field view
        time.sleep(1.0)
        
        # Second tap on home button (to get back to city view)
        if not tap_screen(device_id, adb_path, HOME_BUTTON_X, HOME_BUTTON_Y):
            logger.error("Error tapping home button (second tap)")
            return False
        
        # Wait for city view
        time.sleep(1.0)
        logger.info("Initial view reset completed, ready to start tree clearing")
    
    logger.info(f"Tree harvesting with pre-set coordinates for device {device_id}")
    
    # Handle view reset between sets of trees
    if tree_index == 13 and not view_reset_done:
        logger.info("Resetting view to get to next set of trees...")
        
        # First tap on home button
        if not tap_screen(device_id, adb_path, HOME_BUTTON_X, HOME_BUTTON_Y):
            logger.error("Error tapping home button (first tap)")
            return False
        
        # Wait briefly
        time.sleep(0.5)
        
        # Second tap on home button
        if not tap_screen(device_id, adb_path, HOME_BUTTON_X, HOME_BUTTON_Y):
            logger.error("Error tapping home button (second tap)")
            return False
        
        # Wait for view to reset
        time.sleep(1.0)
        logger.info("View reset completed, ready for next trees")
        
        # Mark view reset as done to avoid looping
        view_reset_done = True
        
        return False  # Return to get a fresh game state
    
    # If we've gone through all trees, perform final reset and mark as complete
    if tree_index >= len(TREE_COORDINATES):
        logger.info("All tree coordinates have been processed, performing final view reset...")
        
        # First tap on home button
        tap_screen(device_id, adb_path, HOME_BUTTON_X, HOME_BUTTON_Y)
        
        # Wait briefly
        time.sleep(0.5)
        
        # Second tap on home button
        tap_screen(device_id, adb_path, HOME_BUTTON_X, HOME_BUTTON_Y)
        
        # Wait for view to reset
        time.sleep(1.0)
        logger.info("Final view reset completed")
        
        # Mark as complete in both global and instance state
        clearing_complete = True
        instance_state.tree_clearing_completed = True
        
        # Record completion in a file (optional)
        try:
            with open("tree_clearing_complete.txt", "w") as f:
                f.write(f"Tree clearing completed at {time.strftime('%Y-%m-%dT%H:%M:%S')}\n")
        except Exception as e:
            logger.error(f"Error writing tree clearing completion file: {e}")
        
        return False
    
    # Get current tree coordinates
    tree_x, tree_y = TREE_COORDINATES[tree_index]
    logger.info(f"Processing tree {tree_index+1}/{len(TREE_COORDINATES)} at position ({tree_x}, {tree_y})")
    
    # Click on the tree
    if not tap_screen(device_id, adb_path, tree_x, tree_y):
        logger.error(f"Error clicking tree at ({tree_x}, {tree_y})")
        tree_index += 1  # Move to next tree even if this one failed
        return False
    
    # Get corresponding harvest coordinates
    harvest_x, harvest_y = HARVEST_COORDINATES[tree_index]
    logger.info(f"Clicking harvest at ({harvest_x}, {harvest_y})")
    
    # Wait briefly for harvest button to appear
    time.sleep(0.5)
    
    # Click the harvest button
    if not tap_screen(device_id, adb_path, harvest_x, harvest_y):
        logger.error(f"Error clicking harvest at ({harvest_x}, {harvest_y})")
        tree_index += 1  # Move to next tree even if harvest failed
        return False
    
    # Log success
    logger.info(f"Successfully harvested tree {tree_index+1}/{len(TREE_COORDINATES)}")
    
    # Increment tree index for next run
    tree_index += 1
    
    # Wait a bit for animations
    time.sleep(1.0)
    
    return True

def reset_tree_clearing():
    """Reset the tree clearing state"""
    global tree_index, clearing_complete, view_reset_done
    tree_index = 0
    clearing_complete = False
    view_reset_done = False
    logging.getLogger("actions").info("Tree clearing state has been reset")

def is_tree_clearing_complete() -> bool:
    """Check if tree clearing is complete"""
    global clearing_complete
    return clearing_complete


def recruit_second_builder(
    device_id: str, 
    game_view: str,
    detections: List[Detection],
    adb_path: str,
    config: Any,
    instance_state: InstanceState,
    api_key: str = None,
    model_id: str = None
) -> bool:
    """
    Handle recruiting the second builder
    
    Args:
        device_id: Device ID
        game_view: Current game view (city, map, etc.)
        detections: List of detections from current screen
        adb_path: Path to ADB executable
        config: Task configuration
        instance_state: Current instance state
        
    Returns:
        True if second builder was recruited, False otherwise
    """
    global second_builder_added
    
    logger = logging.getLogger("actions.recruit_second_builder")
    
    # Skip if already completed
    if second_builder_added or instance_state.second_builder_added:
        logger.info("Second builder already added")
        return False
    
    logger.info(f"Attempting to recruit second builder on device {device_id}")
    
    # First, we need to make sure we're in city view
    if game_view != "city":
        logger.info("Not in city view, cannot recruit second builder")
        return False
    
    # Step 1: Find and click on the Builder's Hut
    builders_hut = None
    for det in detections:
        if (det.class_name == "builders_hut_idle" or det.class_name == "builders_hut") and det.is_confident():
            builders_hut = det
            break
    
    if builders_hut is None:
        logger.info("Builder's Hut not found in detections")
        return False
    
    # Click on Builder's Hut
    logger.info(f"Step 1: Clicking on Builder's Hut at ({builders_hut.x}, {builders_hut.y})")
    if not tap_screen(device_id, adb_path, int(builders_hut.x), int(builders_hut.y)):
        logger.error("Error clicking on Builder's Hut")
        return False
    
    # Wait longer for menu to appear and any help bubbles to show
    logger.info("Waiting for builder's hut menu to fully appear...")
    time.sleep(1)
    
    # Step 2: Look for builders_hut_button
    logger.info("Step 2: Looking for builders_hut_button...")
    
    # Use provided API key and model ID or try to get them from config
    if api_key is None or model_id is None:
        try:
            from roborok.utils.config import load_config
            config_data = load_config("config.json")
            api_key = api_key or config_data["global"]["roboflow_api_key"]
            model_id = model_id or config_data["global"]["roboflow_gameplay_model_id"]
        except Exception as e:
            logger.error(f"Failed to load API key or model ID: {e}")
            return False
    
    # Take a fresh screenshot and analyze
    hire_button_detections = capture_and_detect(device_id, adb_path, api_key, model_id, 4)
            
    # Log what we see for debugging
    logger.info(f"Detected {len(hire_button_detections)} objects after clicking Builder's Hut:")
    for i, det in enumerate(hire_button_detections):
        if det.is_confident():
            logger.info(f"  {i+1}. {det.class_name} ({det.confidence:.2f}): ({det.x:.1f}, {det.y:.1f}) {det.width:.0f}x{det.height:.0f}")
    
    # Look for builders_hut_button
    builders_hut_button = None
    for det in hire_button_detections:
        if det.class_name == "builders_hut_button" and det.is_confident():
            builders_hut_button = det
            break
    
    # If not found, try alternative buttons
    if builders_hut_button is None:
        logger.info("builders_hut_button not found, returnign False...")
        return False
    else:
        # Click on the builders_hut_button we found
        logger.info(f"Clicking on builders_hut_button at ({builders_hut_button.x}, {builders_hut_button.y})")
        if not tap_screen(device_id, adb_path, int(builders_hut_button.x), int(builders_hut_button.y)):
            logger.error("Error clicking on builders_hut_button")
            reset_view(device_id, adb_path)
            return False
    
    # Wait  for hire dialog and any help bubbles to show
    logger.info("Waiting for hire dialog to fully appear...")
    time.sleep(2)
    
    # Step 3: Look for builders_hut_hire_button
    logger.info("Step 3: Looking for builders_hut_hire_button...")
    hire_confirm_detections = capture_and_detect(device_id, adb_path, api_key, model_id, 4)
        
    # Log what we see for debugging
    logger.info(f"Detected {len(hire_confirm_detections)} objects in hire dialog:")
    for i, det in enumerate(hire_confirm_detections):
        if det.is_confident():
            logger.info(f"  {i+1}. {det.class_name} ({det.confidence:.2f}): ({det.x:.1f}, {det.y:.1f}) {det.width:.0f}x{det.height:.0f}")
    
    # Look for hire button
    hire_button = None
    for det in hire_confirm_detections:
        if det.class_name == "builders_hut_hire_button" and det.is_confident():
            hire_button = det
            break
    
    # If not found, try alternative buttons
    if hire_button is None:
        logger.info("builders_hut_hire_button not found, returning False...")
        return False
    else:
        # Click on the hire button we found
        logger.info(f"Clicking on hire button at ({hire_button.x}, {hire_button.y})")
        if not tap_screen(device_id, adb_path, int(hire_button.x), int(hire_button.y)):
            logger.error("Error clicking on hire button")
            reset_view(device_id, adb_path)
            return False
    
    # Wait longer for confirmation dialog and any help bubbles to show
    logger.info("Waiting for confirmation dialog to fully appear...")
    time.sleep(1)
    
    # Step 4: Look for exit_dialog_button or success indicator
    logger.info("Step 4: Looking for exit_dialog_button or success indicator...")
    success_detections = capture_and_detect(device_id, adb_path, api_key, model_id, 4)
    
    # Log what we see for debugging
    logger.info(f"Detected {len(success_detections)} objects in success/confirmation dialog:")
    for i, det in enumerate(success_detections):
        if det.is_confident():
            logger.info(f"  {i+1}. {det.class_name} ({det.confidence:.2f}): ({det.x:.1f}, {det.y:.1f}) {det.width:.0f}x{det.height:.0f}")
    
    # Check for exit buttons or success indicators
    exit_button = None
    success_dialog = None
    
    for det in success_detections:
        if det.class_name == "exit_dialog_button" and det.is_confident():
            exit_button = det
        if det.class_name == "builders_hut_hire_success" and det.is_confident():
            success_dialog = det
    
    # If we found the success dialog, we know it worked
    if success_dialog is not None:
        logger.info("Found builders_hut_hire_success - recruitment successful")
    
    # If we found an exit button, click it
    if exit_button is not None:
        logger.info(f"Clicking exit_dialog_button at ({exit_button.x}, {exit_button.y})")
        if not tap_screen(device_id, adb_path, int(exit_button.x), int(exit_button.y)):
            logger.error("Error clicking exit_dialog_button")
        else:
            logger.info("Successfully clicked exit button")
    else:
        logger.info("No exit button found, clicking center of screen")
        tap_screen(device_id, adb_path, 320, 240)

    # Wait a moment after dismissing dialogs
    time.sleep(1)
    
    # Mark as complete and update state
    instance_state.second_builder_added = True
    if hasattr(instance_state, 'builder_state'):
        instance_state.builder_state.second_builder_added = True
        instance_state.builder_state.second_builder_end_time = datetime.now() + timedelta(days=3)
    
    second_builder_added = True  # Also update the global tracking variable
    logger.info("Second builder successfully recruited!")
    
    # Reset view to ensure we're back in a known state
    logger.info("Resetting view after successful recruitment")
    reset_view(device_id, adb_path)
    return True

def is_second_builder_added() -> bool:
    """Check if second builder is added"""
    global second_builder_added
    return second_builder_added
    
def handle_upgrade_building(
    device_id: str, 
    game_view: str,
    detections: List[Detection],
    adb_path: str,
    config: Any,
    instance_state: InstanceState
) -> bool:
    """
    Handle upgrading buildings based on the ordered build queue
   
    Args:
        device_id: Device ID
        game_view: Current game view (city, map, etc.)
        detections: List of detections from current screen
        adb_path: Path to ADB executable
        config: Task configuration
        instance_state: Current instance state
        
    Returns:
        True if a building was upgraded, False otherwise
    """
    from roborok.utils.adb import tap_screen
    from roborok.utils.config import load_config
    
    logger = logging.getLogger("actions.upgrade_building")
    
    # Skip if not in city view
    if game_view != "city":
        logger.info("Not in city view, skipping building upgrade")
        return False
    
    # Check if we've completed all building tasks
    if not hasattr(instance_state, 'current_task_index'):
        instance_state.current_task_index = 0
    
    if instance_state.current_task_index >= len(instance_state.building_tasks):
        logger.info("All building tasks completed, nothing to do")
        return False
    
    # Get the current building task
    current_task = instance_state.get_current_task()
    if not current_task:
        logger.info("No current building task available")
        return False
    
    logger.info(f"Current building task: {current_task.type} {current_task.building}")
    
    # Check if builders are available
    logger.info("Checking builder availability by inspecting builders hut")
    
    # Debug log all detections to troubleshoot
    #logger.info(f"Found {len(detections)} detections. Looking for builders_hut or builders_hut_button")
    #for i, det in enumerate(detections):
    #    if det.confidence > 0.5:  # Only log higher confidence detections
    #        logger.info(f"  Detection {i+1}: {det.class_name} (conf: {det.confidence:.2f}) at ({det.x:.1f}, {det.y:.1f})")
    
    # Find the builders hut
    builders_hut = None
    builders_hut_button = None
    
    for detection in detections:
        if detection.class_name == "builders_hut" and detection.is_confident():
            logger.info(f"Found builders_hut with confidence {detection.confidence:.2f}")
            builders_hut = detection
            break
        elif detection.class_name == "builders_hut_idle" and detection.is_confident():
            logger.info(f"Found builders_hut_idle with confidence {detection.confidence:.2f}")
            builders_hut = detection
            break
        elif detection.class_name == "builders_hut_button" and detection.confidence > 0.3:
            logger.info(f"Found builders_hut_button with confidence {detection.confidence:.2f}")
            builders_hut_button = detection
    
    # Path 1: If we found the builder's hut, click on it first
    if builders_hut is not None:
        logger.info(f"Clicking on Builder's Hut at ({builders_hut.x}, {builders_hut.y})")
        if not tap_screen(device_id, adb_path, int(builders_hut.x), int(builders_hut.y)):
            logger.error("Error clicking on Builder's Hut")
            return False
        
        # Wait for builders hut menu to appear
        time.sleep(2)
        
        # Take a fresh screenshot and analyze
        try:
            config_data = load_config("config.json")
            api_key = config_data["global"]["roboflow_api_key"]
            model_id = config_data["global"]["roboflow_gameplay_model_id"]
            
            menu_detections = capture_and_detect(device_id, adb_path, api_key, model_id, 4)
            
            # Debug log all detections to troubleshoot
            #logger.info(f"After clicking builders_hut, found {len(menu_detections)} detections. Looking for builders_hut_button")
            #for i, det in enumerate(menu_detections):
            #    if det.confidence > 0.5:  # Only log higher confidence detections
            #        logger.info(f"  Detection {i+1}: {det.class_name} (conf: {det.confidence:.2f}) at ({det.x:.1f}, {det.y:.1f})")
            
            # Look for builders hut button to go to second screen
            builders_hut_button = None
            for det in menu_detections:
                if det.class_name == "builders_hut_button" and det.confidence > 0.3:
                    logger.info(f"Found builders_hut_button with confidence {det.confidence:.2f}")
                    builders_hut_button = det
                    break
            
            if builders_hut_button is None:
                logger.info("builders_hut_button not found, exiting menu")
                reset_view(device_id, adb_path)
                return False
        
        except Exception as e:
            logger.error(f"Error finding builders_hut_button after clicking builders_hut: {e}")
            reset_view(device_id, adb_path)
            return False
    
    # Path 2: If we didn't find builder's hut but found builders_hut_button directly
    elif builders_hut_button is not None:
        logger.info(f"Builder's Hut not found, but found builders_hut_button - skipping ahead")
        # We don't need to click anything here as we've already found the button
    else:
        logger.info("Neither Builder's Hut nor builders_hut_button found in detections")
        return False
    
    # At this point we should have a valid builders_hut_button, continue with clicking it
    logger.info(f"Clicking on builders_hut_button at ({builders_hut_button.x}, {builders_hut_button.y})")
    if not tap_screen(device_id, adb_path, int(builders_hut_button.x), int(builders_hut_button.y)):
        logger.error("Error clicking on builders_hut_button")
        reset_view(device_id, adb_path)
        return False
    
    # Wait for the queue screen to appear
    time.sleep(1)
    
    # Take another screenshot to check queue status
    try:
        config_data = load_config("config.json")
        api_key = config_data["global"]["roboflow_api_key"]
        model_id = config_data["global"]["roboflow_gameplay_model_id"]
        
        queue_detections = capture_and_detect(device_id, adb_path, api_key, model_id, 4)
        
        # Debug log all detections to troubleshoot
        logger.info(f"After clicking builders_hut_button, found {len(queue_detections)} detections. Looking for builder queue indicators")
        for i, det in enumerate(queue_detections):
            if det.confidence > 0.5:  # Only log higher confidence detections
                logger.info(f"  Detection {i+1}: {det.class_name} (conf: {det.confidence:.2f}) at ({det.x:.1f}, {det.y:.1f})")
    
        # Check for available builder queues
        available_queues = 0
        for det in queue_detections:
            if det.class_name == "builders_hut_two_queues_available" and det.confidence > 0.6:
                logger.info("Found two available builder queues")
                available_queues = 2
                break
            elif det.class_name == "builders_hut_build_button" and det.confidence > 0.3:
                logger.info("Found builder queue available")
                available_queues = 1
                # keep looking through list (don't break) because we might find two queues available
        
        # First exit the builder dialog by clicking the exit button
        exit_button = None
        for det in queue_detections:
            if det.class_name == "exit_dialog_button" and det.is_confident():
                exit_button = det
                break
                
        if exit_button:
            logger.info(f"Clicking exit dialog button at ({exit_button.x}, {exit_button.y})")
            tap_screen(device_id, adb_path, int(exit_button.x), int(exit_button.y))
            time.sleep(0.5)  # Wait for dialog to close
        else:            
            logger.info("No exit dialog button found, attempting to tap home button in lower left")
            # Fallback to tapping home button which should dismiss dialog
            tap_screen(device_id, adb_path, HOME_BUTTON_X, HOME_BUTTON_Y)
            time.sleep(0.5)
            
        # Now reset the view to get back to city view
        logger.info("Exiting builder menu")
        reset_view(device_id, adb_path)
        
        if available_queues == 0:
            logger.info("No builder queues available, skipping building upgrades")
            return False
        
        # If we get here, at least one builder is available
        logger.info(f"{available_queues} builder queue(s) available, proceeding with building upgrades")
    
    except Exception as e:
        logger.error(f"Error checking builder availability: {e}")
        reset_view(device_id, adb_path)
        return False
    
    # Process tasks based on available builder queues
    task_success = False
    tasks_processed = 0
    max_tasks = min(available_queues, 2)  # Limit to at most 2 tasks
    skipped_tasks = []  # Track which tasks we skipped
    
    while tasks_processed < max_tasks and instance_state.current_task_index < len(instance_state.building_tasks):
        # Get the current task
        current_task = instance_state.get_current_task()
        if not current_task:
            break
        
        logger.info(f"Processing task {tasks_processed + 1}/{max_tasks}: {current_task.type} {current_task.building}")
        
        # Get fresh detections for this task
        logger.info("Getting fresh detections for building task")
        fresh_detections = capture_and_detect(device_id, adb_path, api_key, model_id, 4)
        
        # Execute the current task
        success = False
        try:
            if current_task.type == 'build_new':
                success = build_new_building(device_id, game_view, fresh_detections, adb_path, api_key, model_id, current_task, instance_state)
            elif current_task.type == 'upgrade':
                success = upgrade_building(device_id, game_view, fresh_detections, adb_path, api_key, model_id, current_task, instance_state)
        except Exception as e:
            logger.error(f"Error processing task: {e}")
            success = False
                    
        if success:
            # Mark the current task as completed and move to the next one
            logger.info(f"Task completed: {current_task.type} {current_task.building}")
            instance_state.mark_current_task_completed()
            task_success = True
            
            # Update city hall level specifically if this is the city hall
            if current_task.type == 'upgrade' and current_task.building == 'cityhall':
                instance_state.city_hall_level += 1
                logger.info(f"Updated city hall level to {instance_state.city_hall_level}")
        else:
            # Task failed - skip it and look ahead to the next one
            logger.info(f"Task failed/unavailable: {current_task.type} {current_task.building} - skipping for now")
            # Record this for retry later
            skipped_tasks.append(current_task)
            # Skip to the next task
            instance_state.skip_current_task(cooldown_seconds=10)  # Try again in 10 seconds
        
        tasks_processed += 1
        
        # After processing a task, get a fresh view of the city
        if tasks_processed < max_tasks:
            reset_view(device_id, adb_path)
            time.sleep(1)  # Give a moment for the view to reset
    
    # If no tasks succeeded but we skipped some, return true so we don't spam logs
    if not task_success and skipped_tasks:
        logger.info(f"Skipped {len(skipped_tasks)} tasks for later retry")
        
    return task_success

def build_new_building(
    device_id: str, 
    game_view: str,
    detections: List[Detection],
    adb_path: str,
    api_key: str,
    model_id: str,
    task: BuildTask,
    instance_state: InstanceState
) -> bool:
    """Handle building a new structure"""
    logger = logging.getLogger("actions.build_new")
    logger.info(f"Starting new building: {task.building}")
    
    # Find the "build new" button
    build_new_button = None
    for det in detections:
        if det.class_name == "build_available" and det.is_confident():
            build_new_button = det
            break
            
    if build_new_button is None:
        logger.info("No build button found, checking for build_new_button instead")
        for det in detections:
            if det.class_name == "build_new_button" and det.is_confident():
                build_new_button = det
                break
                
    if build_new_button is None:
        logger.info("No build buttons found at all, returning")
        return False
    
    # Click the build button
    logger.info(f"Found build button at ({build_new_button.x}, {build_new_button.y}), clicking...")
    if not tap_screen(device_id, adb_path, int(build_new_button.x), int(build_new_button.y)):
        logger.error("Error tapping build button")
        return False
    
    # Wait for building menu to appear
    logger.info("Waiting for building menu to appear...")
    time.sleep(2)
    
    # Parse the detect class to check for category prefix (economic: or military:)
    detect_class_parts = task.detect_class.split(':')
    if len(detect_class_parts) > 1:
        category = detect_class_parts[0].strip()
        building_class = detect_class_parts[1].strip()
        logger.info(f"Using category '{category}' for building class '{building_class}'")
    else:
        logger.info(f"No category specified in {task.detect_class}, using full string as building class")
        category = ""
        building_class = task.detect_class
    
    # Take new screenshot and detect building options
    logger.info("Taking new screenshot to detect building options...")
    build_menu_detections = capture_and_detect(device_id, adb_path, api_key, model_id, 4)
    
    # Log what we see for debugging
    logger.info(f"Building menu detections ({len(build_menu_detections)} total):")
    for i, det in enumerate(build_menu_detections):
        if det.is_confident():
            logger.info(f"  {i+1}. {det.class_name} ({det.confidence:.2f}): ({det.x:.1f}, {det.y:.1f})")
    
    # Find the category button if category is specified
    if category:
        category_button = None
        if category.lower() == "economic":
            for det in build_menu_detections:
                if det.class_name == "build_economic" and det.is_confident():
                    category_button = det
                    break
            logger.info("Looking for economic buildings tab")
        elif category.lower() == "military":
            for det in build_menu_detections:
                if det.class_name == "build_military" and det.is_confident():
                    category_button = det
                    break
            logger.info("Looking for military buildings tab")
        else:
            logger.info(f"Unknown build interface category: {category}")
            reset_view(device_id, adb_path)
            return False
        
        if category_button is None:
            logger.info(f"Could not find {category} tab button")
            reset_view(device_id, adb_path)
            return False
        
        # Click on the category tab
        logger.info(f"Clicking on {category} buildings tab at ({category_button.x}, {category_button.y})")
        if not tap_screen(device_id, adb_path, int(category_button.x), int(category_button.y)):
            logger.error(f"Error tapping {category} tab")
            reset_view(device_id, adb_path)
            return False
        
        # Wait for tab to activate
        time.sleep(1)
        
        # Get fresh detections after switching tabs
        logger.info("Getting fresh detections after switching tabs...")
        build_menu_detections = capture_and_detect(device_id, adb_path, api_key, model_id, 4)
    
    # Log what we see for debugging
    logger.info(f"Building menu detections after tab switch ({len(build_menu_detections)} total):")
    for i, det in enumerate(build_menu_detections):
        if det.is_confident():
            logger.info(f"  {i+1}. {det.class_name} ({det.confidence:.2f}): ({det.x:.1f}, {det.y:.1f})")
    
    # Look for the specific building type option
    building_button = None
    for det in build_menu_detections:
        if det.class_name == building_class and det.is_confident():
            building_button = det
            break
    
    if building_button is None:
        logger.info(f"Building option for '{building_class}' not found, checking for alternative format...")
        
        # Try with "build_" prefix
        for det in build_menu_detections:
            if det.class_name == f"build_{building_class}" and det.is_confident():
                building_button = det
                logger.info(f"Found alternative format 'build_{building_class}'")
                break
        
        if building_button is None:
            logger.info(f"Building option for '{building_class}' not found with any format")
            reset_view(device_id, adb_path)
            return False
    
    # Click on the building option
    logger.info(f"Clicking on {building_class} building at ({building_button.x}, {building_button.y})")
    if not tap_screen(device_id, adb_path, int(building_button.x), int(building_button.y)):
        logger.error("Error tapping building option")
        reset_view(device_id, adb_path)
        return False
    
    # Wait for placement mode
    logger.info("Waiting for placement mode...")
    time.sleep(1)
    
    # Look for confirm button
    logger.info("Taking screenshot to find confirm button...")

    confirm_detections = capture_and_detect(device_id, adb_path, api_key, model_id, 4)
    
    # Log what we see for debugging
    logger.info(f"Confirm screen detections ({len(confirm_detections)} total):")
    for i, det in enumerate(confirm_detections):
        if det.is_confident():
            logger.info(f"  {i+1}. {det.class_name} ({det.confidence:.2f}): ({det.x:.1f}, {det.y:.1f})")
        
    # Look for confirm button
    confirm_button = None
    for det in confirm_detections:
        if det.class_name == "accept_build_location" and det.is_confident():
            confirm_button = det
            break
    
    if confirm_button is None:
        logger.info("No confirmation buttons found, failing build operation")
        reset_view(device_id, adb_path)
        return False
    
    # Click confirm button
    logger.info(f"Clicking confirm button at ({confirm_button.x}, {confirm_button.y})")
    if not tap_screen(device_id, adb_path, int(confirm_button.x), int(confirm_button.y)):
        logger.error("Error tapping confirm button")
        reset_view(device_id, adb_path)
        return False
    
    # Wait for confirmation
    logger.info("Waiting for confirmation dialog...")
    time.sleep(1)
    
    logger.info("Build operation complete, resetting view...")
    reset_view(device_id, adb_path)
    return True

def upgrade_building(
    device_id: str, 
    game_view: str,
    detections: List[Detection],
    adb_path: str,
    api_key: str,
    model_id: str,
    task: BuildTask,
    instance_state: InstanceState
) -> bool:
    """Handle upgrading an existing building"""
    logger = logging.getLogger("actions.upgrade_building")
    logger.info(f"Attempting to upgrade {task.building}")
    
    # Check if we have a stored position for this building
    use_stored_position = False
    click_x, click_y = 0, 0
    
    if is_multiple_type_building(task.building):
        main_x, main_y, has_position = get_main_building_position(task.building, instance_state)
        if has_position:
            logger.info(f"Using stored position ({main_x}, {main_y}) for main {task.building}")
            click_x, click_y = main_x, main_y
            use_stored_position = True
    
    # If not using stored position, find the building in detections
    if not use_stored_position:
        building = None
        
        # Try each possible detection class (split by comma if multiple)
        detect_classes = task.detect_class.split(',')
        for class_name in detect_classes:
            class_name = class_name.strip()
            logger.info(f"Looking for building with class '{class_name}'")
            
            for det in detections:
                if det.class_name == class_name and det.is_confident():
                    building = det
                    logger.info(f"Found building with class '{class_name}'")
                    break
            
            if building:
                break
        
        if not building:
            logger.info(f"{task.building} not found in detections with any specified class")
            return False
        
        click_x, click_y = int(building.x), int(building.y)
        logger.info(f"Found {task.building} at position ({click_x}, {click_y})")
        
        # If this is a multiple-type building, store the position
        if is_multiple_type_building(task.building):
            update_main_building_position(task.building, click_x, click_y, instance_state)
            logger.info(f"Updated position for multiple-type building {task.building} to ({click_x}, {click_y})")
    
    # Click on the building
    logger.info(f"Clicking on {task.building} at ({click_x}, {click_y})")
    if not tap_screen(device_id, adb_path, click_x, click_y):
        logger.error(f"Error clicking on {task.building}")
        reset_view(device_id, adb_path)
        return False
    
    # Wait for menu to appear
    logger.info("Waiting for building menu to appear...")
    time.sleep(1)
    
    # Take another screenshot to find the upgrade button
    logger.info("Taking screenshot to find upgrade button...")
    upgrade_detections = capture_and_detect(device_id, adb_path, api_key, model_id, 4)
    
    # Log what we see for debugging
    logger.info(f"Upgrade menu detections ({len(upgrade_detections)} total):")
    for i, det in enumerate(upgrade_detections):
        if det.is_confident():
            logger.info(f"  {i+1}. {det.class_name} ({det.confidence:.2f}): ({det.x:.1f}, {det.y:.1f})")
    
    # Look for upgrade button
    upgrade_button = None
    for det in upgrade_detections:
        if det.class_name == "upgrade_button" and det.is_confident():
            upgrade_button = det
            break
    
    # If upgrade button not found, try alternative names
    if not upgrade_button:
        logger.info("Upgrade button not found, checking for alternative button names")
        
        alternative_names = ["upgrade_available", "upgrade_building", "building_upgrade"]
        for alt_name in alternative_names:
            for det in upgrade_detections:
                if det.class_name == alt_name and det.is_confident():
                    upgrade_button = det
                    logger.info(f"Found alternative upgrade button: {alt_name}")
                    break
            
            if upgrade_button:
                break
        
        if not upgrade_button:
            logger.info(f"No upgrade button found for {task.building} with any name")
            reset_view(device_id, adb_path)
            return False
    
    # Click on upgrade button
    logger.info(f"Clicking on upgrade button at ({upgrade_button.x}, {upgrade_button.y})")
    if not tap_screen(device_id, adb_path, int(upgrade_button.x), int(upgrade_button.y)):
        logger.error("Error clicking on upgrade button")
        reset_view(device_id, adb_path)
        return False
    
    # Wait for upgrade dialog
    logger.info("Waiting for upgrade dialog to appear...")
    time.sleep(2)
    
    # Take another screenshot to find confirmation button
    logger.info("Taking screenshot to find confirmation button...")
    confirm_detections = capture_and_detect(device_id, adb_path, api_key, model_id, 4)
    
    # Log what we see for debugging
    logger.info(f"Confirm dialog detections ({len(confirm_detections)} total):")
    for i, det in enumerate(confirm_detections):
        if det.is_confident():
            logger.info(f"  {i+1}. {det.class_name} ({det.confidence:.2f}): ({det.x:.1f}, {det.y:.1f})")
        
    # Check if requirements not met
    for det in confirm_detections:
        if det.class_name == "upgrade_not_available" and det.is_confident():
            logger.info(f"Requirements not met for upgrading {task.building}")
            # Look for exit button
            for exit_det in confirm_detections:
                if exit_det.class_name == "exit_dialog_button" and exit_det.is_confident():
                    logger.info(f"Clicking exit dialog button at ({exit_det.x}, {exit_det.y})")
                    tap_screen(device_id, adb_path, int(exit_det.x), int(exit_det.y))
                    break
            reset_view(device_id, adb_path)
            # This is a valid reason to skip the task - prerequisites not met
            return False
    
    # Look for confirm button
    confirm_button = None
    for det in confirm_detections:
        if det.class_name == "upgrade_available_button" and det.is_confident():
            confirm_button = det
            break
    
    if not confirm_button:
        logger.info("upgrade_available_button not found, checking for confirm_button")
        for det in confirm_detections:
            if det.class_name == "confirm_button" and det.is_confident():
                confirm_button = det
                break
    
    if not confirm_button:
        logger.info("No confirm button found, failing upgrade operation")
        reset_view(device_id, adb_path)
        # This is a valid reason to skip the task - prerequisites not met
        return False
    
    # Click confirm button
    logger.info(f"Clicking confirm button at ({confirm_button.x}, {confirm_button.y})")
    if not tap_screen(device_id, adb_path, int(confirm_button.x), int(confirm_button.y)):
        logger.error("Error clicking on confirm button")
        reset_view(device_id, adb_path)
        return False
    
    # Wait for processing
    logger.info("Waiting for processing...")
    time.sleep(1)
    
    # Take one more screenshot to check for alliance help request
    logger.info("Taking screenshot to check for alliance help button...")
    help_detections = capture_and_detect(device_id, adb_path, api_key, model_id, 4)
    
    # Log what we see for debugging
    logger.info(f"Help request detections ({len(help_detections)} total):")
    for i, det in enumerate(help_detections):
        if det.is_confident():
            logger.info(f"  {i+1}. {det.class_name} ({det.confidence:.2f}): ({det.x:.1f}, {det.y:.1f})")
    
    # Look for alliance help button
    for det in help_detections:
        if det.class_name == "alliance_help_button" and det.is_confident():
            logger.info(f"Clicking alliance help request button at ({det.x}, {det.y})")
            tap_screen(device_id, adb_path, int(det.x), int(det.y))
            time.sleep(0.5)
            break

    # Note: We're not updating building levels here since we're using the simplified approach

    logger.info(f"{task.building} upgrade initiated successfully")
    reset_view(device_id, adb_path)
    return True
    
# Helper functions for building management

def is_multiple_type_building(building_type: str) -> bool:
    """Check if this building type can have multiple instances"""
    multiple_types = {
        "farm": True,
        "quarry": True, 
        "lumber_mill": True,
        "goldmine": True,
        "hospital": True
    }
    return multiple_types.get(building_type, False)

def update_main_building_position(building_type: str, x: int, y: int, instance_state: InstanceState) -> None:
    """Update the position of a main building if not already set"""
    if not hasattr(instance_state, 'building_positions'):
        from dataclasses import field, dataclass
        
        @dataclass
        class BuildingPosition:
            x: int = 0
            y: int = 0
            
        @dataclass
        class BuildingPositions:
            farm: BuildingPosition = field(default_factory=BuildingPosition)
            quarry: BuildingPosition = field(default_factory=BuildingPosition)
            lumber_mill: BuildingPosition = field(default_factory=BuildingPosition)
            goldmine: BuildingPosition = field(default_factory=BuildingPosition)
            hospital: BuildingPosition = field(default_factory=BuildingPosition)
            
        instance_state.building_positions = BuildingPositions()
    
    if building_type == "farm":
        if instance_state.building_positions.farm.x == 0 and instance_state.building_positions.farm.y == 0:
            instance_state.building_positions.farm.x = x
            instance_state.building_positions.farm.y = y
            logging.getLogger("actions").info(f"Set main farm position to ({x}, {y})")
    elif building_type == "quarry":
        if instance_state.building_positions.quarry.x == 0 and instance_state.building_positions.quarry.y == 0:
            instance_state.building_positions.quarry.x = x
            instance_state.building_positions.quarry.y = y
            logging.getLogger("actions").info(f"Set main quarry position to ({x}, {y})")
    elif building_type == "lumber_mill":
        if instance_state.building_positions.lumber_mill.x == 0 and instance_state.building_positions.lumber_mill.y == 0:
            instance_state.building_positions.lumber_mill.x = x
            instance_state.building_positions.lumber_mill.y = y
            logging.getLogger("actions").info(f"Set main lumber mill position to ({x}, {y})")
    elif building_type == "goldmine":
        if instance_state.building_positions.goldmine.x == 0 and instance_state.building_positions.goldmine.y == 0:
            instance_state.building_positions.goldmine.x = x
            instance_state.building_positions.goldmine.y = y
            logging.getLogger("actions").info(f"Set main goldmine position to ({x}, {y})")
    elif building_type == "hospital":
        if instance_state.building_positions.hospital.x == 0 and instance_state.building_positions.hospital.y == 0:
            instance_state.building_positions.hospital.x = x
            instance_state.building_positions.hospital.y = y
            logging.getLogger("actions").info(f"Set main hospital position to ({x}, {y})")

def get_main_building_position(building_type: str, instance_state: InstanceState) -> Tuple[int, int, bool]:
    """Get the position of a main building"""
    if not hasattr(instance_state, 'building_positions'):
        return 0, 0, False
        
    if building_type == "farm":
        return (instance_state.building_positions.farm.x, 
                instance_state.building_positions.farm.y,
                instance_state.building_positions.farm.x != 0 or instance_state.building_positions.farm.y != 0)
    elif building_type == "quarry":
        return (instance_state.building_positions.quarry.x, 
                instance_state.building_positions.quarry.y,
                instance_state.building_positions.quarry.x != 0 or instance_state.building_positions.quarry.y != 0)
    elif building_type == "lumber_mill":
        return (instance_state.building_positions.lumber_mill.x, 
                instance_state.building_positions.lumber_mill.y,
                instance_state.building_positions.lumber_mill.x != 0 or instance_state.building_positions.lumber_mill.y != 0)
    elif building_type == "goldmine":
        return (instance_state.building_positions.goldmine.x, 
                instance_state.building_positions.goldmine.y,
                instance_state.building_positions.goldmine.x != 0 or instance_state.building_positions.goldmine.y != 0)
    elif building_type == "hospital":
        return (instance_state.building_positions.hospital.x, 
                instance_state.building_positions.hospital.y,
                instance_state.building_positions.hospital.x != 0 or instance_state.building_positions.hospital.y != 0)
    else:
        return 0, 0, False

def reset_view(device_id: str, adb_path: str, api_key: str = None, model_id: str = None):
    """
    Reset view to ensure we're back in a known state, automatically dismissing any help bubbles
    
    Args:
        device_id: Device ID
        adb_path: Path to ADB executable
        api_key: Optional Roboflow API key (will be loaded from config if not provided)
        model_id: Optional Roboflow model ID (will be loaded from config if not provided)
    """
    logger = logging.getLogger("actions")
    logger.info("Resetting view...")
    
    # Get API key and model ID if not provided
    if api_key is None or model_id is None:
        try:
            from roborok.utils.config import load_config
            config_data = load_config("config.json")
            api_key = api_key or config_data["global"]["roboflow_api_key"]
            model_id = model_id or config_data["global"]["roboflow_gameplay_model_id"]
        except Exception as e:
            logger.error(f"Failed to load API key or model ID: {e}")
            # Fall back to default reset approach
            logger.info("Using default reset approach without detection")
            tap_screen(device_id, adb_path, HOME_BUTTON_X, HOME_BUTTON_Y)
            time.sleep(0.5)
            tap_screen(device_id, adb_path, HOME_BUTTON_X, HOME_BUTTON_Y)
            time.sleep(0.5)
            return
    
    # Get clean detections with no help bubbles
    detections = capture_and_detect(device_id, adb_path, api_key, model_id, 4)
    
    # First check if we're in build menu
    for det in detections:
        if det.class_name == "in_build" and det.is_confident():
            logger.info("Detected we're in build menu, clicking center of screen to exit")
            tap_screen(device_id, adb_path, 320, 240)  # Click center of screen
            time.sleep(1)  # Wait for menu to close
            
            # Get fresh detections after exiting build menu
            detections = capture_and_detect(device_id, adb_path, api_key, model_id, 4)
            break
    
    # Check if we need to exit a dialog
    exit_button = None
    for det in detections:
        if det.class_name == "exit_dialog_button" and det.is_confident():
            exit_button = det
            break
            
    if exit_button:
        logger.info(f"Clicking exit dialog button at ({exit_button.x}, {exit_button.y})")
        tap_screen(device_id, adb_path, int(exit_button.x), int(exit_button.y))
        time.sleep(1)
        
        # Get fresh detections after exiting dialog
        detections = capture_and_detect(device_id, adb_path, api_key, model_id, 4)
            
    # Determine if we're in city view, field view, or some other view
    in_city = False
    on_map = False
    
    for det in detections:
        if det.class_name == "in_city" and det.is_confident():
            in_city = True
            break
        elif det.class_name == "on_map" and det.is_confident():
            on_map = True
            break
    
    # Reset based on current view
    if on_map:
        logger.info("On map view, clicking home button once to return to city")
        tap_screen(device_id, adb_path, HOME_BUTTON_X, HOME_BUTTON_Y)
        time.sleep(0.5)
    else:
        logger.info("In city or unknown view, clicking home button twice")
        tap_screen(device_id, adb_path, HOME_BUTTON_X, HOME_BUTTON_Y)
        time.sleep(1.0)
        tap_screen(device_id, adb_path, HOME_BUTTON_X, HOME_BUTTON_Y)
        time.sleep(1.0)
    
    # Final verification - make sure we're in city view
    logger.info("Performing final verification that we're in city view")
    final_detections = capture_and_detect(device_id, adb_path, api_key, model_id, 4)
    
    in_city = False
    for det in final_detections:
        if det.class_name == "in_city" and det.is_confident():
            in_city = True
            break
            
    if not in_city:
        logger.info("Still not in city view, clicking home button one more time")
        tap_screen(device_id, adb_path, HOME_BUTTON_X, HOME_BUTTON_Y)
        time.sleep(1.0)
    
    
    logger.info("View reset completed")
    
def parse_time_remaining(text: str) -> Optional[timedelta]:
    """
    Parse time remaining text into a timedelta
    
    Handles formats like:
    - "00:01:19" (HH:MM:SS)
    - "1d 23:52:50" (D days HH:MM:SS)
    
    Args:
        text: The time text to parse
        
    Returns:
        timedelta object if successful, None if parsing failed
    """
    logger = logging.getLogger("ocr.parse_time")
    logger.info(f"Parsing time text: '{text}'")
    
    try:
        # Check if there's a day component
        if 'd' in text.lower():
            # Format like "1d 23:52:50"
            day_pattern = r'(\d+)\s*d\s+(\d{2}:\d{2}:\d{2})'
            match = re.search(day_pattern, text)
            if match:
                days_str, time_str = match.groups()
                logger.info(f"Found days format: {days_str} days and {time_str}")
                
                days = int(days_str)
                hours, minutes, seconds = map(int, time_str.split(':'))
                return timedelta(days=days, hours=hours, minutes=minutes, seconds=seconds)
        
        # Standard format "00:00:00"
        time_pattern = r'(\d{2}:\d{2}:\d{2})'
        match = re.search(time_pattern, text)
        if match:
            time_str = match.group(1)
            logger.info(f"Found standard time format: {time_str}")
            
            hours, minutes, seconds = map(int, time_str.split(':'))
            return timedelta(hours=hours, minutes=minutes, seconds=seconds)
            
        logger.warning(f"Could not parse time from text: '{text}'")
        return None
    except Exception as e:
        logger.error(f"Error parsing time: {e}")
        return None