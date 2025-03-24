# roborok/vision/time_ocr/ocr.py
import torch
from .model import SecondsAwareTimeDigitCNN
import cv2
import numpy as np

class TimeOCR:
    
    def __init__(self, model_path=None):
        self.device = torch.device('cuda' if torch.cuda.is_available() 
                                  else 'mps' if torch.backends.mps.is_available() 
                                  else 'cpu')
        self.model = SecondsAwareTimeDigitCNN().to(self.device)
        
        if model_path is None:
            # Get the directory where this file is located
            import os
            current_dir = os.path.dirname(os.path.abspath(__file__))
            model_path = os.path.join(current_dir, 'time_cnn_best_20250318_092526.pth')
        
        # Load the wrapped model state
        checkpoint = torch.load(model_path, map_location=self.device)
        # Extract just the model state from the checkpoint
        if 'model_state_dict' in checkpoint:
            self.model.load_state_dict(checkpoint['model_state_dict'])
        else:
            # If for some reason it's not wrapped
            self.model.load_state_dict(checkpoint)
        
        self.model.eval()
        self.last_confidences = []
        
    def preprocess_for_debug(self, image):
        """
        Preprocess an image for debugging visualization (without converting to tensor)
    
        Args:
            image: NumPy array of the cropped time region
        
        Returns:
            Preprocessed image as a NumPy array (before tensor conversion)
        """
        # Convert to grayscale if needed
        if len(image.shape) == 3:
            image = cv2.cvtColor(image, cv2.COLOR_BGR2GRAY)
    
        # Keep original aspect ratio when resizing
        h, w = image.shape[:2]
        aspect = w / h
    
        # Calculate new width while preserving aspect ratio
        new_w = int(30 * aspect)
    
        # If new width is too large, we need to cap it
        if new_w > 150:
            print(f"Warning: Image aspect ratio is very wide ({aspect:.2f}). Capping width at 150px.")
            new_w = 150
    
        resized = cv2.resize(image, (new_w, 30))
    
        # Create empty canvas of target size
        result = np.zeros((30, 150), dtype=np.float32)
    
        # Center the resized image
        x_offset = max(0, (150 - new_w) // 2)
        paste_width = min(new_w, 150)
        result[:, x_offset:x_offset+paste_width] = resized[:, :paste_width]
    
        # Normalize pixel values
        result = result.astype(np.float32) / 255.0
    
        return result

    def preprocess_image(self, image):
        # Convert to grayscale if needed
        if len(image.shape) == 3:
            image = cv2.cvtColor(image, cv2.COLOR_BGR2GRAY)
    
        # Add debug logging for image shape
        print(f"Original image shape: {image.shape}")
    
        # Keep original aspect ratio when resizing
        h, w = image.shape[:2]
        aspect = w / h
    
        # Calculate new width while preserving aspect ratio
        new_w = int(30 * aspect)
    
        # If new width is too large, we need to cap it
        if new_w > 150:
            print(f"Warning: Image aspect ratio is very wide ({aspect:.2f}). Capping width at 150px.")
            new_w = 150
    
        resized = cv2.resize(image, (new_w, 30))
    
        # Create empty canvas of target size
        result = np.zeros((30, 150), dtype=np.float32)
    
        # Center the resized image
        x_offset = max(0, (150 - new_w) // 2)
        paste_width = min(new_w, 150)
        result[:, x_offset:x_offset+paste_width] = resized[:, :paste_width]
    
        # Normalize pixel values
        result = result.astype(np.float32) / 255.0
    
        # Convert to tensor
        image_tensor = torch.tensor(result).unsqueeze(0).unsqueeze(0)
        return image_tensor.to(self.device)
        
    
    def predict(self, image):
        """
        Recognize time from an image
        
        Args:
            image: NumPy array of the cropped time region
            
        Returns:
            time_str: String in format "HH:MM:SS"
        """
        with torch.no_grad():
            # Preprocess
            x = self.preprocess_image(image)
            
            # Forward pass
            outputs = self.model(x)
            
            # Get predictions and confidences
            digits = []
            confidences = []
            for output in outputs:
                # Get the probabilities using softmax
                probs = torch.nn.functional.softmax(output, dim=1)
                # Get the predicted digit and its confidence
                confidence, digit_idx = torch.max(probs, dim=1)
                
                digits.append(str(digit_idx.item()))
                confidences.append(confidence.item())
            
            # Store confidences for later retrieval
            self.last_confidences = confidences
            
            # Format as time
            time_str = f"{digits[0]}{digits[1]}:{digits[2]}{digits[3]}:{digits[4]}{digits[5]}"
            return time_str
            
    def get_digit_confidences(self):
        """Return the confidence scores for the last prediction"""
        return self.last_confidences