# roborok/models.py
"""Data models for RoboRok"""

from dataclasses import dataclass, field
from datetime import datetime
from typing import Dict, List, Optional, Any
import time

@dataclass
class Detection:
    """Represents a detected object from Roboflow API"""
    class_name: str
    x: float
    y: float
    width: float
    height: float
    confidence: float
    
    def is_confident(self, min_confidence: float = 0.7) -> bool:
        """Check if the detection meets the confidence threshold"""
        return self.confidence >= min_confidence
        
    def is_text_region(self) -> bool:
        """Check if this detection is a text region that needs OCR"""
        text_region_classes = ['power_counter', 'gold_counter', 'food_counter', 'gems_counter', 'level_indicator', 'text_box']
        return self.class_name in text_region_classes

    def get_crop_coordinates(self) -> tuple:
        """Get the crop coordinates for this detection"""
        # Convert from center coordinates to top-left and bottom-right
        x1 = int(self.x - (self.width / 2))
        y1 = int(self.y - (self.height / 2))
        x2 = int(self.x + (self.width / 2))
        y2 = int(self.y + (self.height / 2))

        # Add a small margin
        margin = 3
        x1 = max(0, x1 - margin)
        y1 = max(0, y1 - margin)
        x2 = x2 + margin
        y2 = y2 + margin

        return (x1, y1, x2, y2)
    

@dataclass
class TaskConfig:
    """Configuration for a gameplay task"""
    max_level_desired: int = 0
    claim_only_main_quest: bool = False
    troop_level_desired: int = 0
    research_path: List[str] = field(default_factory=list)
    
@dataclass
class BuildTask:
    """Represents a building construction or upgrade task"""
    type: str  # 'build_new' or 'upgrade'
    building: str  # Building name (e.g., 'cityhall', 'farm')
    detect_class: str  # Class name for detection
    completed: bool = False
    skipped_attempts: int = 0  # Track how many times we've skipped this task
    last_attempted: Optional[float] = None  # Timestamp of last attempt

def default_build_queue() -> List[BuildTask]:
    """Define the default build queue in order of execution"""
    return [
        # Starting point:  City Hall 2, Farm 1, Wall 1, Tavern 1, Barracks 1, Storehouse 1, Hospital 1, Scout Camp 1
        # 
        BuildTask(type="upgrade", building="cityhall", detect_class="cityhall"), # 3
        
        # Level 2 buildings
        # BuildTask(type="build_new", building="archery_range", detect_class="military:build_archery_range"), # 1
        BuildTask(type="upgrade", building="barracks", detect_class="barracks"), # 2
        BuildTask(type="upgrade", building="scout_camp", detect_class="scout_camp"), # 2
        BuildTask(type="upgrade", building="farm", detect_class="farm"), # 2
        BuildTask(type="upgrade", building="tavern", detect_class="tavern"), # 2
        BuildTask(type="upgrade", building="farm", detect_class="farm"),  # 3
        BuildTask(type="upgrade", building="hospital", detect_class="hospital"), # 2
        BuildTask(type="build_new", building="lumber_mill", detect_class="economic:build_lumber_mill"), # 1
        BuildTask(type="upgrade", building="wall", detect_class="wall"), # 2
        BuildTask(type="upgrade", building="lumber_mill", detect_class="lumber_mill"), # 2
        
        # Probably ready by now for City Hall level 3
        BuildTask(type="upgrade", building="cityhall", detect_class="cityhall"),  # 3
        BuildTask(type="upgrade", building="lumber_mill", detect_class="lumber_mill"),  # 3
        
        # Level 3 buildings
        BuildTask(type="upgrade", building="archery_range", detect_class="archery_range"), # 2
        BuildTask(type="upgrade", building="barracks", detect_class="barracks"), # 3
        BuildTask(type="build_new", building="stable", detect_class="military:build_stable"), # 1
        BuildTask(type="upgrade", building="scout_camp", detect_class="scout_camp"), # 3
        BuildTask(type="upgrade", building="hospital", detect_class="hospital"), # 3
        BuildTask(type="upgrade", building="lumber_mill", detect_class="lumber_mill"), # 4
        
        # Add more tasks as needed for higher levels
    ]

@dataclass
class InstanceState:
    """State of a game instance"""
    id: str
    device_id: str
    tutorial_completed: bool = False
    tutorial_upgrade_complete_clicked: bool = False
    tutorial_final_arrow_clicked: bool = False
    tree_clearing_completed: bool = False
    second_builder_added: bool = False
    city_hall_level: int = 1
    last_screenshot_path: str = ""
    last_report_time: datetime = field(default_factory=lambda: datetime.min)
    
    # Building tracking fields
    building_tasks: List[BuildTask] = field(default_factory=list)  # Ordered list of building tasks
    current_task_index: int = 0  # Index of current task in the queue
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert the instance state to a dictionary"""
        result = {}
        for key, value in self.__dict__.items():
            if isinstance(value, datetime):
                # Convert datetime to ISO string
                result[key] = value.isoformat()
            elif key == 'building_tasks':
                # Convert BuildTask objects to dictionary
                result[key] = [task.__dict__ for task in value]
            elif key == 'building_positions' and hasattr(value, '__dict__'):
                # Handle BuildingPositions object
                positions_dict = {}
                for pos_key, pos_value in value.__dict__.items():
                    if hasattr(pos_value, '__dict__'):
                        positions_dict[pos_key] = pos_value.__dict__
                    else:
                        positions_dict[pos_key] = pos_value
                result[key] = positions_dict
            else:
                result[key] = value
        return result
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'InstanceState':
        """Create an instance state from a dictionary"""
        try:
            # Make a copy to avoid modifying the original
            data_copy = data.copy()
        
            # Handle datetime conversion
            if 'last_report_time' in data_copy and isinstance(data_copy['last_report_time'], str):
                try:
                    data_copy['last_report_time'] = datetime.fromisoformat(data_copy['last_report_time'])
                except ValueError:
                    data_copy['last_report_time'] = datetime.min
        
            # Handle BuildTask conversion
            if 'building_tasks' in data_copy and isinstance(data_copy['building_tasks'], list):
                build_tasks = []
                for task_dict in data_copy['building_tasks']:
                    # Convert dict to BuildTask
                    build_tasks.append(BuildTask(**task_dict))
                data_copy['building_tasks'] = build_tasks
        
            # Create the instance
            instance = cls(**data_copy)
            return instance
        except Exception as e:
            print(f"Error in from_dict: {e}")
            # Return a default instance as fallback
            return cls(id=data.get("id", "unknown"), device_id=data.get("device_id", "unknown"))
        

    def initialize_build_queue(self) -> None:
        """Initialize the build queue if it's empty"""
        if not self.building_tasks:
            self.building_tasks = default_build_queue()
            self.current_task_index = 0
    
    def get_current_task(self) -> Optional[BuildTask]:
        """Get the current building task"""
        if not self.building_tasks or self.current_task_index >= len(self.building_tasks):
            return None
        
        return self.building_tasks[self.current_task_index]
    
    def mark_current_task_completed(self) -> None:
        """Mark the current task as completed and move to the next one"""
        if self.building_tasks and self.current_task_index < len(self.building_tasks):
            self.building_tasks[self.current_task_index].completed = True
            self.current_task_index += 1
    
    def skip_current_task(self, cooldown_minutes: int = 10) -> None:
        """
        Skip the current task temporarily and move to the next one
        
        Args:
            cooldown_minutes: How many minutes to wait before retrying this task
        """
        if self.building_tasks and self.current_task_index < len(self.building_tasks):
            # Increment skip counter and record timestamp
            self.building_tasks[self.current_task_index].skipped_attempts += 1
            self.building_tasks[self.current_task_index].last_attempted = time.time()
            
            # Move to the next task
            self.current_task_index += 1
            
            # If we've reached the end of the list, try to find any skipped tasks that can be retried
            if self.current_task_index >= len(self.building_tasks):
                self._reset_to_skipped_task(cooldown_minutes)
    
    def _reset_to_skipped_task(self, cooldown_minutes: int) -> None:
        """
        Try to find a skipped task that's ready to be retried
        
        Args:
            cooldown_minutes: Minimum minutes before retrying a skipped task
        """
        min_cooldown_sec = cooldown_minutes * 60
        current_time = time.time()
        
        # Find the earliest skipped task that's past its cooldown
        for i, task in enumerate(self.building_tasks):
            if not task.completed and task.skipped_attempts > 0 and task.last_attempted is not None:
                time_elapsed = current_time - task.last_attempted
                if time_elapsed >= min_cooldown_sec:
                    self.current_task_index = i
                    return
        
        # If no tasks can be retried, just keep at the end
        self.current_task_index = len(self.building_tasks)
    
    
@dataclass
class OCRResult:
    """Result from OCR processing"""
    text: str
    confidence: float
    region_type: str
    
    def get_numeric_value(self) -> Optional[int]:
        """Attempt to extract a numeric value from the text"""
        import re
        # Look for numbers in the text
        numeric_match = re.search(r'(\d[\d,]*)', self.text)
        if numeric_match:
            # Remove commas and convert to int
            return int(numeric_match.group(1).replace(',', ''))
        return None
    
    def get_resource_value(self) -> Optional[int]:
        """Get resource value, handling K/M suffixes"""
        import re
        # Look for number with optional K or M suffix
        match = re.search(r'(\d[\d,]*\.?\d*)([KkMm])?', self.text)
        if not match:
            return None
            
        value = float(match.group(1).replace(',', ''))
        suffix = match.group(2)
        
        # Apply multiplier for K or M
        if suffix and suffix.upper() == 'K':
            value *= 1000
        elif suffix and suffix.upper() == 'M':
            value *= 1000000
            
        return int(value)
        
@dataclass
class GameResources:
    """Current resource levels in the game"""
    power: int = 0
    gold: int = 0
    food: int = 0
    wood: int = 0
    stone: int = 0
    gems: int = 0
    
    def can_afford(self, costs: Dict[str, int]) -> bool:
        """Check if we can afford a purchase/upgrade"""
        if costs.get('gold', 0) > self.gold:
            return False
        if costs.get('food', 0) > self.food:
            return False
        if costs.get('wood', 0) > self.wood:
            return False
        if costs.get('stone', 0) > self.stone:
            return False
        if costs.get('gems', 0) > self.gems:
            return False
        return True

@dataclass
class GameState:
    """Overall game state"""
    resources: GameResources = field(default_factory=GameResources)
    power: int = 0
    city_hall_level: int = 1
    in_city: bool = True
    buildings: Dict[str, int] = field(default_factory=dict)  # name -> level
    
    def update_from_ocr(self, ocr_results: List[OCRResult]):
        """Update state based on OCR results"""
        for result in ocr_results:
            if result.region_type == 'gold':
                value = result.get_resource_value()
                if value is not None:
                    self.resources.gold = value
            elif result.region_type == 'food':
                value = result.get_resource_value()
                if value is not None:
                    self.resources.food = value
            # Add similar logic for other resource types
