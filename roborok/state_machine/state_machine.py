# roborok/state_machine.py
"""State machine for game automation"""

import logging
from abc import ABC, abstractmethod
from typing import List, Optional

from roborok.actions.actions import analyze_game_state
from roborok.utils.adb import tap_screen
from roborok.models import Detection, InstanceState, GameResources 


class GameState(ABC):
    """Abstract base class for game states"""
    
    @abstractmethod
    def execute(self, context) -> bool:
        """
        Execute actions for this state
        
        Args:
            context: The game state context
            
        Returns:
            True if an action was taken, False otherwise
        """
        pass


class GameStateContext:
    """
    Context class for the state machine that manages the current state
    and transitions between states.
    """
    
    def __init__(self, device_id: str, adb_path: str, api_key: str, model_id: str):
        """
        Initialize the state machine context
    
        Args:
            device_id: Device ID
            adb_path: Path to ADB executable
            api_key: Roboflow API key
            model_id: Roboflow model ID
        """
        self.device_id = device_id
        self.adb_path = adb_path
        self.api_key = api_key
        self.model_id = model_id
    
        # Initialize with Unknown state
        self.current_state = UnknownState()
    
        # Instance state for persistent data
        self.instance_state = InstanceState(id=device_id, device_id=device_id)
    
        # Current game resources
        self.resources = GameResources()
    
        # Current detections from last analysis
        self.detections: List[Detection] = []
    
        # Last detected game view (city, map, etc.)
        self.game_view: str = "unknown"
    
        # Logger
        self.logger = logging.getLogger("state_machine")

    def update(self) -> bool:
        """
        Update the state machine by analyzing game state and executing current state

        Returns:
            True if an action was taken, False otherwise
        """
        # Capture screenshot and analyze game state
        game_view, resources, detections, error = analyze_game_state(
            self.device_id, 
            self.adb_path, 
            self.api_key, 
            self.model_id
        )

        if error:
            self.logger.error(f"Error analyzing game state: {error}")
            return False

        # Update context with new information
        self.game_view = game_view
        if resources:
            self.resources = resources
        self.detections = detections if detections else []
    
        # If we're in tutorial and it's not completed, transition to tutorial state
        if not self.instance_state.tutorial_completed:
            tutorial_elements = False
            for detection in self.detections:
                # Note: using "counselor text bubble" which is the correct detection class
                if detection.class_name in ["click_arrow", "click_target", "counselor text bubble", "upgrade_complete"]:
                    tutorial_elements = True
                    break

            if tutorial_elements:
                if not isinstance(self.current_state, TutorialState):
                    self.transition_to(TutorialState())
                return self.current_state.execute(self)
    
        # Handle special popup states that can occur in any game state (only if not in tutorial)
        for detection in self.detections:
            # Handle new feature unlock popup
            if detection.class_name == "new_feature_unlock" and detection.is_confident():
                self.logger.info("Found new feature unlock popup, tapping home button area of screen")
                if tap_screen(self.device_id, self.adb_path, 31, 450):  # Home button coordinates
                    return True
            
            # Add other popup handlers here if needed
    
        # Determine appropriate state based on game view if we're in unknown state
        if isinstance(self.current_state, UnknownState):
            self._determine_state_from_view(game_view)

        # Execute current state
        return self.current_state.execute(self)

    def _determine_state_from_view(self, game_view: str):
        """
        Determine the appropriate state based on game view
    
        Args:
            game_view: Current game view (city, map, etc.)
        """
        if game_view == "city":
            self.transition_to(CityViewState())
        elif game_view == "map":
            self.transition_to(MapViewState())
        else:
            # Stay in unknown state if we can't determine
            pass
    
    def transition_to(self, new_state):
        """
        Transition to a new state
        
        Args:
            new_state: The new state to transition to
        """
        self.logger.info(f"Transitioning from {self.current_state.__class__.__name__} to {new_state.__class__.__name__}")
        self.current_state = new_state

class UnknownState(GameState):
    """State when the game state is unknown"""
    
    def execute(self, context: GameStateContext) -> bool:
        """
        Try to determine the current state by analyzing the screen
    
        Args:
            context: The game state context
        
        Returns:
            True if an action was taken, False otherwise
        """
        context.logger.info("In unknown state, attempting to determine current state")
    
        # Check if we have any detections that indicate a specific state
        for detection in context.detections:
            if detection.class_name == "in_city" and detection.is_confident():
                context.transition_to(CityViewState())
                return True
            elif detection.class_name == "on_map" and detection.is_confident():
                context.transition_to(MapViewState())
                return True
    
        # Try clicking the home button to get to a known state
        context.logger.info("Clicking home button to get to known state")
        if tap_screen(context.device_id, context.adb_path, 31, 450):  # Home button coordinates
            return True
        
        return False
        
class CityViewState(GameState):
    """State for when in city view"""
    
    def execute(self, context: GameStateContext) -> bool:
        """
        Execute city view actions
        
        Args:
            context: The game state context
            
        Returns:
            True if an action was taken, False otherwise
        """
        context.logger.info("Executing city view actions")
                
        return False # Let the task system handle more complex actions


class MapViewState(GameState):
    """State for when in map view"""
    
    def execute(self, context: GameStateContext) -> bool:
        """
        Execute map view actions
        
        Args:
            context: The game state context
            
        Returns:
            True if an action was taken, False otherwise
        """
        context.logger.info("Executing map view actions")
        
        # Example: Return to city if no specific map tasks
        context.logger.info("No map tasks, returning to city")
        
        # Look for return to city button
        for detection in context.detections:
            if detection.class_name == "return_to_city_button" and detection.is_confident():
                context.logger.info("Found return to city button, tapping it")
                if tap_screen(context.device_id, context.adb_path, int(detection.x), int(detection.y)):
                    context.transition_to(CityViewState())
                    return True
        
        return False
        
class TutorialState(GameState):
    """State for when in tutorial"""
    
    def execute(self, context: GameStateContext) -> bool:
        """
        Execute tutorial actions
        
        Args:
            context: The game state context
            
        Returns:
            True if an action was taken, False otherwise
        """
        context.logger.info("Executing tutorial actions")
        
        # Check tutorial completion flags
        if context.instance_state.tutorial_upgrade_complete_clicked and context.instance_state.tutorial_final_arrow_clicked:
            context.logger.info("Tutorial completion sequence detected")
            context.instance_state.tutorial_completed = True
            context.transition_to(CityViewState())
            return True
        
        # Keep track of special conditions
        has_arrow = False
        has_target = False
        best_target = None
        
        civ_count = 0
        found_preferred_civ = False
        preferred_civ_detection = None
        preferred_civ = getattr(context.instance_state, "preferred_civilization", "china")
        
        # end conditions
        has_build_archery_range = False
        has_build_confirm_button = False
        has_build_reject_button = False
        build_confirm_button_detection = None
        
        # Look for important detections first
        for detection in context.detections:
            # Count civilizations for later
            if is_civilization(detection.class_name):
                civ_count += 1
                # Check if it's our preferred civ
                if detection.class_name.lower() == preferred_civ.lower() and detection.is_confident():
                    found_preferred_civ = True
                    preferred_civ_detection = detection
            
            # Track arrow and target for final arrow handling
            if detection.class_name == "click_arrow" and detection.is_confident():
                has_arrow = True
            if detection.class_name == "click_target" and detection.is_confident():
                has_target = True
                if best_target is None or detection.confidence > best_target.confidence:
                    best_target = detection
                    
            if detection.class_name == "build_archery_range" and detection.is_confident():
                has_build_archery_range = True
            elif detection.class_name == "build_confirm_button" and detection.is_confident():
                has_build_confirm_button = True
                build_confirm_button_detection = detection
            elif detection.class_name == "build_reject_button" and detection.is_confident():
                has_build_reject_button = True
            
        
        # Now process detections one by one with priority
        for detection in context.detections:
            # Highest priority: upgrade complete notification
            if (detection.class_name == "upgrade_complete" or detection.class_name == "new_feature_unlock")  and detection.is_confident():
                context.logger.info("Found upgrade notification, tapping center of screen")
                if tap_screen(context.device_id, context.adb_path, 240, 400):
                    # Mark this step as complete
                    context.instance_state.tutorial_upgrade_complete_clicked = True
                    return True
            
            # Skip button (with both possible formats)
            elif (detection.class_name == "skip button" or detection.class_name == "skip_button") and detection.is_confident():
                context.logger.info("Found skip button, tapping it")
                if tap_screen(context.device_id, context.adb_path, int(detection.x), int(detection.y)):
                    return True
            
            # Tutorial completion indication
            elif detection.class_name == "tutorial_complete" and detection.is_confident():
                context.logger.info("Tutorial completed, transitioning to city view")
                context.instance_state.tutorial_completed = True
                context.transition_to(CityViewState())
                return True
            
            # Confirmation button
            elif detection.class_name == "confirm_button" and detection.is_confident():
                context.logger.info("Found confirm button, tapping it")
                if tap_screen(context.device_id, context.adb_path, int(detection.x), int(detection.y)):
                    return True
            
            # Counselor text bubble
            elif detection.class_name == "counselor text bubble" and detection.is_confident():
                context.logger.info("Found counselor text bubble, tapping it")
                if tap_screen(context.device_id, context.adb_path, int(detection.x), int(detection.y)):
                    return True
            
            # Click target (if not handling final arrow)
            elif detection.class_name == "click_target" and detection.is_confident() and not context.instance_state.tutorial_upgrade_complete_clicked:
                context.logger.info("Found tutorial click target, tapping it")
                if tap_screen(context.device_id, context.adb_path, int(detection.x), int(detection.y)):
                    return True
        
        # After checking individual detections, handle more complex state cases
        
        # Handle final arrow case
        if context.instance_state.tutorial_upgrade_complete_clicked and not context.instance_state.tutorial_final_arrow_clicked:
            if has_arrow and has_target and best_target is not None:
                context.logger.info("Found final arrow and target, tapping target to complete tutorial")
                if tap_screen(context.device_id, context.adb_path, int(best_target.x), int(best_target.y)):
                    # Mark this step as complete
                    context.instance_state.tutorial_final_arrow_clicked = True
                    return True
        
        # Handle civilization selection screen
        if civ_count >= 3:
            context.logger.info("Detected civilization selection screen")
            
            # If we found our preferred civilization, tap it
            if found_preferred_civ and preferred_civ_detection is not None:
                context.logger.info(f"Found preferred civilization {preferred_civ}, tapping it")
                if tap_screen(context.device_id, context.adb_path, int(preferred_civ_detection.x), int(preferred_civ_detection.y)):
                    return True
                    
        # If we have all build elements and no tutorial arrows/targets, handle as tutorial end
        if has_build_archery_range and has_build_confirm_button and has_build_reject_button and not has_arrow and not has_target:
            context.logger.info("Detected archery range build screen - end of tutorial")
            if build_confirm_button_detection is not None:
                context.logger.info("Clicking confirm button to end tutorial")
                if tap_screen(context.device_id, context.adb_path, int(build_confirm_button_detection.x), int(build_confirm_button_detection.y)):
                    # Mark tutorial as completed
                    context.instance_state.tutorial_completed = True
                    context.logger.info("Tutorial marked as completed via archery range build")
                    context.transition_to(CityViewState())
                    return True
        
                
        return False

# Helper function to check if a class name is a civilization
def is_civilization(class_name: str) -> bool:
    """Check if a class name is a civilization"""
    civilizations = [
        "arabia", "britain", "china", "egypt", "france",
        "germany", "greece", "japan", "korea", "maya",
        "rome", "spain", "vikings"
    ]
    return class_name.lower() in civilizations

def create_state_machine(device_id: str, adb_path: str, api_key: str, model_id: str) -> GameStateContext:
    """
    Create a new state machine instance
    
    Args:
        device_id: Device ID
        adb_path: Path to ADB executable
        api_key: Roboflow API key
        model_id: Roboflow model ID
        
    Returns:
        GameStateContext instance
    """
    return GameStateContext(device_id, adb_path, api_key, model_id)