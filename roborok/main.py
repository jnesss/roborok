# roborok/main.py
"""Main entry point for RoboRok"""

import argparse
import time
import logging
import json
import os
from typing import Dict, Any

from roborok.utils.config import load_config
from roborok.state_machine.state_machine import create_state_machine, TutorialState, CityViewState
from roborok.tasks import TaskManager, create_default_tasks
from roborok.models import InstanceState, GameState, GameResources
from roborok.utils.adb import tap_screen
from roborok.actions.actions import analyze_game_state

def setup_logging():
    """Configure logging"""
    logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
        handlers=[
            logging.StreamHandler(),
            logging.FileHandler("roborok.log")
        ]
    )

def load_instance_state(state_file: str, instance_id: str, device_id: str) -> InstanceState:
    """
    Load instance state from file or create new
    
    Args:
        state_file: Path to state file
        instance_id: Instance ID
        device_id: Device ID
        
    Returns:
        InstanceState object
    """
    logger = logging.getLogger("main.load_instance_state")
    
    # Try to load existing state
    if os.path.exists(state_file):
        try:
            with open(state_file, 'r') as f:
                states_data = json.load(f)
                
            if instance_id in states_data:
                logger.info(f"Found existing state data for instance {instance_id}")
                try:
                    state_data = states_data[instance_id]
                    # Convert from dict to InstanceState
                    state = InstanceState.from_dict(state_data)
                    # Ensure device ID is current
                    state.device_id = device_id
                    
                    # Make sure we have a building queue
                    if not hasattr(state, 'building_tasks') or not state.building_tasks:
                        logger.info("Initializing building queue for existing instance state")
                        state.initialize_build_queue()
                        
                    # Make sure current_task_index is initialized
                    if not hasattr(state, 'current_task_index'):
                        logger.info("Initializing current_task_index for existing instance state")
                        state.current_task_index = 0
                        
                    logger.info(f"Loaded state with current_task_index: {state.current_task_index}")
                    return state
                except Exception as e:
                    logger.error(f"Error converting state data: {e}")
                    # Continue to create new state
        except Exception as e:
            logger.error(f"Error loading state: {e}")
    
    # Create new state if loading failed
    logger.info("Creating new instance state with initialized building queue")
    new_state = InstanceState(id=instance_id, device_id=device_id)
    new_state.initialize_build_queue()
    return new_state

def save_instance_state(state_file: str, instance_states: Dict[str, InstanceState]):
    """
    Save instance states to file
    
    Args:
        state_file: Path to state file
        instance_states: Dictionary of instance states
    """
    # Convert to dict
    states_data = {}
    for instance_id, state in instance_states.items():
        states_data[instance_id] = state.to_dict()
        
    # Save to file
    try:
        with open(state_file, 'w') as f:
            json.dump(states_data, f, indent=2)
    except Exception as e:
        logging.error(f"Error saving state: {e}")

def run_tutorial(device_id: str, adb_path: str, api_key: str, tutorial_model_id: str, preferred_civilization: str, instance_state: InstanceState) -> bool:
    """
    Run the tutorial automation
    
    Args:
        device_id: Device ID
        adb_path: Path to ADB executable
        api_key: Roboflow API key
        tutorial_model_id: Roboflow tutorial model ID
        preferred_civilization: Preferred civilization
        instance_state: Instance state to update
        
    Returns:
        True if tutorial was completed, False otherwise
    """
    logger = logging.getLogger("main.tutorial")
    logger.info(f"Starting tutorial automation for device {device_id}")
    
    # Create state machine for tutorial with tutorial model
    state_machine = create_state_machine(device_id, adb_path, api_key, tutorial_model_id)
    state_machine.instance_state = instance_state
    state_machine.current_state = TutorialState()
    
    # Set preferred civilization in instance state
    instance_state.preferred_civilization = preferred_civilization
    
    # Run tutorial for a maximum number of iterations
    max_iterations = 500
    iteration_count = 0
    
    while iteration_count < max_iterations:
        iteration_count += 1
        logger.info(f"Tutorial iteration {iteration_count}/{max_iterations}")
        
        # Update state machine
        action_taken = state_machine.update()
        
        # Check if tutorial is completed
        if instance_state.tutorial_completed:
            logger.info("Tutorial completed successfully!")
            return True
            
        # Sleep between iterations
        sleep_time = 1  # 1 second between iterations
        logger.info(f"Tutorial iteration {iteration_count} completed, sleeping for {sleep_time} seconds")
        time.sleep(sleep_time)
    
    logger.warning(f"Tutorial did not complete after {max_iterations} iterations")
    return False
    
def run_startup_tasks(device_id: str, adb_path: str, api_key: str, gameplay_model_id: str, instance_state: InstanceState, skip_tree_clearing=False, skip_second_builder=False) -> bool:
    """
    Run one-time startup tasks after tutorial but before main gameplay
    
    Args:
        device_id: Device ID
        adb_path: Path to ADB executable  
        api_key: Roboflow API key
        gameplay_model_id: Roboflow gameplay model ID
        instance_state: Instance state to update
        skip_tree_clearing: Skip tree clearing task
        skip_second_builder: Skip second builder recruitment
        
    Returns:
        True if all startup tasks were completed, False otherwise
    """
    logger = logging.getLogger("main.startup")
    logger.info("Running startup tasks after tutorial completion")
    
    # 1. Check if tree clearing is already completed
    if not instance_state.tree_clearing_completed and not skip_tree_clearing:
        logger.info("Starting tree clearing task")
        
        # Import here to avoid circular imports
        from roborok.actions.actions import clear_trees, is_tree_clearing_complete, reset_tree_clearing
        
        # Create a fresh state machine just once for initial state detection
        state_machine = create_state_machine(device_id, adb_path, api_key, gameplay_model_id)
        state_machine.instance_state = instance_state
        state_machine.update()  # Get fresh game state and detections initially
        
        # Make sure we're using the game view from the state machine
        game_view = state_machine.game_view
        detections = state_machine.detections
        
        # Keep clearing trees until the function returns False (all done)
        # But do NOT update the state machine each time to avoid the builder's hut clicks
        tree_clearing_in_progress = True
        max_tree_attempts = 30  # Safety limit
        tree_attempt_count = 0
        
        # Reset the tree clearing state to start fresh
        reset_tree_clearing()
        
        while tree_clearing_in_progress and tree_attempt_count < max_tree_attempts:
            tree_attempt_count += 1
            logger.info(f"Tree clearing attempt {tree_attempt_count}/{max_tree_attempts}")
            
            # Run tree clearing task
            tree_clearing_success = clear_trees(
                device_id=device_id,
                game_view=game_view,
                detections=detections,
                adb_path=adb_path,
                config=None,
                instance_state=instance_state
            )
            
            # Check if all trees are cleared
            all_trees_cleared = is_tree_clearing_complete() or instance_state.tree_clearing_completed
            
            if all_trees_cleared:
                logger.info("All trees successfully cleared")
                tree_clearing_in_progress = False
            elif not tree_clearing_success:
                # If clear_trees returns False but trees aren't cleared, we need a fresh state
                logger.info("Tree clearing action returned False, updating game state")
                state_machine.update()
                game_view = state_machine.game_view
                detections = state_machine.detections
            
            # Small delay between attempts
            if tree_clearing_in_progress:
                time.sleep(0.5)
        
        # Final check to see if we succeeded or timed out
        if is_tree_clearing_complete() or instance_state.tree_clearing_completed:
            logger.info("Tree clearing task completed successfully")
        else:
            logger.error(f"Tree clearing task failed after {tree_attempt_count} attempts")
            return False
            
        # Save state after tree clearing
        instance_id = instance_state.id
        all_states = {instance_id: instance_state}
        save_instance_state("instance_states.json", all_states)
    else:
        if skip_tree_clearing and not instance_state.tree_clearing_completed:
            logger.info("Tree clearing skipped due to --skip-tree-clearing flag")
            # Mark as completed to avoid doing it in the future
            instance_state.tree_clearing_completed = True
        else:
            logger.info("Tree clearing already completed, skipping")
    
    # Only proceed to second builder after trees are fully cleared
    if not instance_state.second_builder_added and not skip_second_builder:
        logger.info("Starting second builder task")
        
        # Import here to avoid circular imports
        from roborok.actions.actions import recruit_second_builder
        
        # Create a fresh state machine to get detections
        state_machine = create_state_machine(device_id, adb_path, api_key, gameplay_model_id)
        state_machine.instance_state = instance_state
        state_machine.update()  # Get fresh game state and detections
        
        # Run second builder task
        success = recruit_second_builder(
            device_id=device_id,
            game_view=state_machine.game_view,
            detections=state_machine.detections,
            adb_path=adb_path,
            config=None,
            instance_state=instance_state,
            api_key=api_key,
            model_id=gameplay_model_id
        )
        
        if success:
            logger.info("Second builder task completed successfully")
            
            # Save state after adding second builder
            instance_id = instance_state.id
            all_states = {instance_id: instance_state}
            save_instance_state("instance_states.json", all_states)
        else:
            logger.error("Second builder task failed")
            return False
    else:
        if skip_second_builder and not instance_state.second_builder_added:
            logger.info("Second builder recruitment skipped due to --skip-second-builder flag")
            # Mark as completed to avoid doing it in the future
            instance_state.second_builder_added = True
        else:
            logger.info("Second builder already added, skipping")
    
    logger.info("All startup tasks completed successfully")
    return True


def main():
    """Main entry point"""
    # Setup logging
    setup_logging()
    logger = logging.getLogger(__name__)
    
    # Parse command line arguments
    parser = argparse.ArgumentParser(description="RoboRok - Rise of Kingdoms Automation")
    parser.add_argument("--config", default="config.json", help="Configuration file")
    parser.add_argument("--state", default="instance_states.json", help="State file")
    parser.add_argument("--instance", default="instance1", help="Instance ID to run")
    parser.add_argument("--cycles", type=int, default=0, help="Number of cycles to run (0 = infinite)")
    parser.add_argument("--skip-tutorial", action="store_true", help="Skip tutorial even if not completed")
    parser.add_argument("--skip-tree-clearing", action="store_true", help="Skip tree clearing")
    parser.add_argument("--skip-second-builder", action="store_true", help="Skip second builder recruitment")
    args = parser.parse_args()
    
    # Load configuration
    config = load_config(args.config)
    if not config:
        logger.error("Failed to load configuration")
        return
    
    # Check if instance exists in config
    if args.instance not in config["instances"]:
        logger.error(f"Instance {args.instance} not found in configuration")
        return
    
    # Get instance config
    instance_config = config["instances"][args.instance]
    
    # Get API keys and model IDs
    api_key = config["global"]["roboflow_api_key"]
    gameplay_model_id = config["global"]["roboflow_gameplay_model_id"]
    tutorial_model_id = config["global"]["roboflow_tutorial_model_id"]
    
    # Get device ID and ADB path
    device_id = instance_config["device_id"]
    adb_path = config["gameplay"]["adb_path"]
    
    # Get preferred civilization
    preferred_civilization = instance_config.get("preferred_civilization", "china")
    
    # Load or create instance state
    instance_state = load_instance_state(args.state, args.instance, device_id)
    
    # If tutorial not completed and not skipped, run it first
    if not instance_state.tutorial_completed and not args.skip_tutorial:
        logger.info("Tutorial not completed, running tutorial automation")
        tutorial_completed = run_tutorial(
            device_id, 
            adb_path, 
            api_key, 
            tutorial_model_id, 
            preferred_civilization, 
            instance_state
        )
        
        if tutorial_completed:
            logger.info("Tutorial completed successfully, moving to startup tasks")
            # Save state after tutorial
            all_states = {args.instance: instance_state}
            save_instance_state(args.state, all_states)
            
            # Run startup tasks
            startup_completed = run_startup_tasks(
                device_id,
                adb_path,
                api_key,
                gameplay_model_id,
                instance_state,
                skip_tree_clearing=args.skip_tree_clearing,
                skip_second_builder=args.skip_second_builder
            )
            
            if not startup_completed:
                logger.error("Startup tasks failed, exiting")
                return
                
            logger.info("Startup tasks completed, moving to main gameplay")
        else:
            logger.error("Tutorial automation failed, exiting")
            return
            
    elif not instance_state.tutorial_completed and args.skip_tutorial:
        logger.info("Tutorial not completed but --skip-tutorial flag set, skipping tutorial")
        # Mark tutorial as completed to proceed with gameplay
        instance_state.tutorial_completed = True
        
        # Run startup tasks if they're not already completed
        if not instance_state.tree_clearing_completed or not instance_state.second_builder_added:
            logger.info("Running startup tasks after skipping tutorial")
            
            startup_completed = run_startup_tasks(
                device_id,
                adb_path,
                api_key,
                gameplay_model_id,
                instance_state,
                skip_tree_clearing=args.skip_tree_clearing,
                skip_second_builder=args.skip_second_builder
            )
            
            if not startup_completed:
                logger.error("Startup tasks failed, exiting")
                return
                
            logger.info("Startup tasks completed, moving to main gameplay")
        
    else:        
        # Tutorial already completed
        logger.info("Tutorial already completed")
        
        # Check if startup tasks are completed
        if not instance_state.tree_clearing_completed or not instance_state.second_builder_added:
            logger.info("Running pending startup tasks")
            
            startup_completed = run_startup_tasks(
                device_id,
                adb_path,
                api_key,
                gameplay_model_id,
                instance_state,
                skip_tree_clearing=args.skip_tree_clearing,
                skip_second_builder=args.skip_second_builder
            )
            
            if not startup_completed:
                logger.error("Startup tasks failed, exiting")
                return
                
            logger.info("Startup tasks completed, moving to main gameplay")
        else:
            logger.info("Startup tasks already completed, proceeding with gameplay")
    
        
    
    # Create state machine for normal gameplay
    state_machine = create_state_machine(device_id, adb_path, api_key, gameplay_model_id)
    state_machine.instance_state = instance_state
    
    # Create task manager and add default tasks
    task_manager = TaskManager()
    for task in create_default_tasks():
        task_manager.add_task(task)
    
    # Run automation loop
    cycle_count = 0
    try:
        while args.cycles == 0 or cycle_count < args.cycles:
            cycle_count += 1
            logger.info(f"Starting cycle {cycle_count}")
            
            # Update state machine
            state_updated = state_machine.update()
            
            if state_updated:
                logger.info("State machine executed an action")
            else:
                # If state machine didn't act, try executing a task
                logger.info("State machine did not act, trying tasks")
                task_manager.execute_highest_priority_task(
                    device_id=device_id,
                    game_view=state_machine.game_view,
                    detections=state_machine.detections,
                    adb_path=adb_path,
                    instance_state=state_machine.instance_state
                )
            
            # Save state after each cycle
            all_states = {args.instance: state_machine.instance_state}
            save_instance_state(args.state, all_states)
            
            # Sleep between cycles
            sleep_time = config["global"].get("refresh_interval_ms", 1000) / 1000.0
            logger.info(f"Cycle {cycle_count} completed, sleeping for {sleep_time:.2f} seconds")
            time.sleep(sleep_time)
            
    except KeyboardInterrupt:
        logger.info("Interrupted by user")
    except Exception as e:
        logger.error(f"Error in automation loop: {e}", exc_info=True)
    finally:
        # Save state before exiting
        all_states = {args.instance: state_machine.instance_state}
        save_instance_state(args.state, all_states)
        logger.info("Saved instance state")

def main_cli():
    """Entry point for command-line interface"""
    try:
        main()
    except Exception as e:
        logging.error(f"Unhandled exception: {e}", exc_info=True)

if __name__ == "__main__":
    main_cli()