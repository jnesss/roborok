# roborok/tasks.py
"""Task system for game automation"""

import logging
import time
from typing import Dict, List, Optional, Any, Callable, Tuple
from dataclasses import dataclass, field
from datetime import datetime

from roborok.models import Detection, InstanceState, GameState, GameResources

@dataclass
class TaskConfig:
    """Configuration for a gameplay task"""
    max_level_desired: int = 0
    claim_only_main_quest: bool = False
    troop_level_desired: int = 0
    research_path: List[str] = field(default_factory=list)


@dataclass
class Task:
    """Represents an automated task with priority and cooldown"""
    name: str
    priority: int  # Higher number = higher priority
    cooldown_sec: int  # Minimum seconds between executions
    handler: Callable  # Function to execute the task
    config: TaskConfig = field(default_factory=TaskConfig)
    last_executed: datetime = field(default_factory=lambda: datetime.min)
    
    @property
    def is_on_cooldown(self) -> bool:
        """Check if this task is currently on cooldown"""
        elapsed = datetime.now() - self.last_executed
        return elapsed.total_seconds() < self.cooldown_sec
    
    def execute(self, *args, **kwargs) -> bool:
        """
        Execute the task if not on cooldown
        
        Returns:
            True if task was executed, False otherwise
        """
        if self.is_on_cooldown:
            return False
        
        # Execute task
        result = self.handler(*args, **kwargs)
        
        # Update last executed time if successful
        if result:
            self.last_executed = datetime.now()
            
        return result


class TaskManager:
    """Manages and executes tasks based on priority and conditions"""
    
    def __init__(self):
        """Initialize task manager"""
        self.tasks: List[Task] = []
        self.logger = logging.getLogger("task_manager")
    
    def add_task(self, task: Task) -> None:
        """
        Add a task to the manager
        
        Args:
            task: Task to add
        """
        self.tasks.append(task)
        # Sort tasks by priority (highest first)
        self.tasks.sort(key=lambda t: t.priority, reverse=True)
        
    def remove_task(self, task_name: str) -> bool:
        """
        Remove a task by name
        
        Args:
            task_name: Name of task to remove
            
        Returns:
            True if task was removed, False if not found
        """
        initial_count = len(self.tasks)
        self.tasks = [t for t in self.tasks if t.name != task_name]
        return len(self.tasks) < initial_count
    
    
    def execute_highest_priority_task(self, 
                                       device_id: str, 
                                       game_view: str,
                                       detections: List[Detection],
                                       adb_path: str,
                                       instance_state: InstanceState) -> bool:
        """
        Execute the highest priority task that isn't on cooldown
        
        Args:
            device_id: Device ID
            game_view: Current game view (city, map, etc.)
            detections: List of detections from current screen
            adb_path: Path to ADB executable
            instance_state: Current instance state
            
        Returns:
            True if a task was executed, False otherwise
        """
        for task in self.tasks:
            if task.is_on_cooldown:
                self.logger.debug(f"Task {task.name} is on cooldown, skipping")
                continue
                
            self.logger.info(f"Executing task: {task.name}")
            result = task.execute(
                device_id=device_id,
                game_view=game_view,
                detections=detections,
                adb_path=adb_path,
                config=task.config,
                instance_state=instance_state
            )
            
            if result:
                self.logger.info(f"Task {task.name} executed successfully")
                return True
            else:
                self.logger.debug(f"Task {task.name} returned no action taken")
        
        self.logger.info("No tasks were executed")
        return False
        
# Move this function outside the TaskManager class
def create_default_tasks() -> List[Task]:
    """
    Create a default set of tasks
    
    Returns:
        List of Task objects
    """
    # Import the handlers here to avoid circular imports
    from roborok.actions.actions import handle_collect_quests, handle_build_order
    
    return [
        Task(
            name="collect_quests",
            priority=100,  # Highest priority
            cooldown_sec=0,  # No cooldown
            handler=handle_collect_quests,
            config=TaskConfig(claim_only_main_quest=False)
        ),
        Task(
            name="build_order",
            priority=90,  # High priority
            cooldown_sec=5,  # 5 second cooldown
            handler=handle_build_order,
            config=TaskConfig(max_level_desired=0)  # 0 = no limit
        )
    ]