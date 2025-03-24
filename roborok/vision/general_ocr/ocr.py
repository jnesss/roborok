# roborok/ocr.py
"""OCR utilities using EasyOCR"""

import easyocr
import numpy as np
from PIL import Image, ImageEnhance, ImageOps, ImageFilter
from typing import List, Dict, Any, Optional

from roborok.models import Detection, OCRResult

# Initialize the OCR reader (lazy loading - only created when needed)
_reader = None

def get_reader():
    """Get the EasyOCR reader instance (initialize if needed)"""
    global _reader
    if _reader is None:
        _reader = easyocr.Reader(['en'])
    return _reader

def process_text_region(image, detection):
    """
    Try both original and processed images and use the result with higher confidence
    """
    # Get crop coordinates
    crop_box = detection.get_crop_coordinates()
    
    # Crop the image
    original_img = image.crop(crop_box)
    
    # Save original cropped image
    original_img.save(f"original_{detection.class_name}.png")
    
    # Get OCR reader
    reader = get_reader()
    
    # Process original image with EasyOCR
    original_results = reader.readtext(
        np.array(original_img), 
        detail=1, 
        paragraph=False,
        allowlist='0123456789,.kKmM'
    )
    
    # If original image produces results, use them
    if original_results:
        best_original = max(original_results, key=lambda x: x[2])
        original_confidence = best_original[2]
        original_text = best_original[1]
        print(f"Original image OCR: '{original_text}' (conf: {original_confidence:.2f})")
    else:
        original_confidence = 0.0
        original_text = ""
    
    # Try enhanced version for comparison
    enhanced_img = original_img.copy()
    
    # Different enhancement based on counter type
    if detection.class_name == "power_counter":
        # Just enhance contrast slightly
        enhancer = ImageEnhance.Contrast(enhanced_img)
        enhanced_img = enhancer.enhance(1.5)
    else:
        # For resource counters (light text on dark background)
        # Convert to RGB first
        enhanced_img = enhanced_img.convert('RGB')
        
        # Apply contrast and brightness enhancements without inversion
        enhancer = ImageEnhance.Contrast(enhanced_img)
        enhanced_img = enhancer.enhance(2.5)
        
        brightness = ImageEnhance.Brightness(enhanced_img)
        enhanced_img = brightness.enhance(1.3)
    
    # Save enhanced image
    enhanced_img.save(f"enhanced_{detection.class_name}.png")
    
    # Process enhanced image
    enhanced_results = reader.readtext(
        np.array(enhanced_img), 
        detail=1, 
        paragraph=False,
        allowlist='0123456789,.kKmM'
    )
    
    # If enhanced image produces results, check them
    if enhanced_results:
        best_enhanced = max(enhanced_results, key=lambda x: x[2])
        enhanced_confidence = best_enhanced[2]
        enhanced_text = best_enhanced[1]
        print(f"Enhanced image OCR: '{enhanced_text}' (conf: {enhanced_confidence:.2f})")
    else:
        enhanced_confidence = 0.0
        enhanced_text = ""
    
    # Choose the better result
    if original_confidence >= enhanced_confidence:
        print(f"Using original image result for {detection.class_name}")
        result_text = original_text
        result_confidence = original_confidence
    else:
        print(f"Using enhanced image result for {detection.class_name}")
        result_text = enhanced_text
        result_confidence = enhanced_confidence
    
    # Return the OCR result
    return OCRResult(
        text=result_text,
        confidence=result_confidence,
        region_type=detection.class_name
    )

def process_game_text(image, detections):
    """
    Process all text regions in the game screenshot
    
    Args:
        image: PIL Image of the full screenshot
        detections: List of Detection objects
        
    Returns:
        List of OCRResult objects
    """
    results = []
    
    # Find text regions
    text_regions = [d for d in detections if d.is_text_region()]
    
    # Process each text region
    for region in text_regions:
        result = process_text_region(image, region)
        results.append(result)
        print(f"OCR Result for {region.class_name}: '{result.text}' (conf: {result.confidence:.2f})")
    
    return results