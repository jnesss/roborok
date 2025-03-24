import torch
import torch.nn as nn
import torch.nn.functional as F

class SecondsAwareTimeDigitCNN(nn.Module):
    def __init__(self):
        super(SecondsAwareTimeDigitCNN, self).__init__()
        
        # Input size: [batch_size, 1, 30, 150]
        # 1 channel (grayscale), 30 height, 150 width
        
        # Base convolutional layers - shared for all digits
        self.conv1 = nn.Conv2d(1, 32, kernel_size=3, padding=1)
        self.bn1 = nn.BatchNorm2d(32)
        self.pool1 = nn.MaxPool2d(2, 2)
        
        self.conv2 = nn.Conv2d(32, 64, kernel_size=3, padding=1)
        self.bn2 = nn.BatchNorm2d(64)
        self.pool2 = nn.MaxPool2d(2, 2)
        
        self.conv3 = nn.Conv2d(64, 128, kernel_size=3, padding=1)
        self.bn3 = nn.BatchNorm2d(128)
        self.pool3 = nn.MaxPool2d(2, 2)
        
        # Use regular dropout instead of dropout2d
        self.dropout1 = nn.Dropout(0.25)
        self.dropout2 = nn.Dropout(0.25)
        
        # Calculate feature size after convolutions
        self.fc_input_size = 128 * 3 * 18
        
        # Fully connected layers - shared base
        self.fc1 = nn.Linear(self.fc_input_size, 512)
        self.fc2 = nn.Linear(512, 256)
        
        # Specialized branches for different digit groups
        # Hours and minutes tens (easier digits)
        self.easy_branch = nn.Sequential(
            nn.Linear(256, 128),
            nn.ReLU(),
            nn.Dropout(0.2)
        )
        
        # Minutes ones and seconds (harder digits)
        self.hard_branch = nn.Sequential(
            nn.Linear(256, 192),  # More capacity
            nn.ReLU(),
            nn.Dropout(0.3),
            nn.Linear(192, 128),
            nn.ReLU(),
            nn.Dropout(0.2)
        )
        
        # Output layers - separate for easy and hard digits
        self.easy_outputs = nn.ModuleList([nn.Linear(128, 10) for _ in range(3)])  # Hours + minutes tens
        self.hard_outputs = nn.ModuleList([nn.Linear(128, 10) for _ in range(3)])  # Minutes ones + seconds
        
    def forward(self, x):
        # Convolutional feature extraction
        x = self.pool1(F.relu(self.bn1(self.conv1(x))))
        x = self.dropout1(x)
        
        x = self.pool2(F.relu(self.bn2(self.conv2(x))))
        x = self.dropout1(x)
        
        x = self.pool3(F.relu(self.bn3(self.conv3(x))))
        x = self.dropout2(x)
        
        # Flatten
        x = x.view(-1, self.fc_input_size)
        
        # Shared fully connected processing
        x = F.relu(self.fc1(x))
        x = self.dropout2(x)
        x = F.relu(self.fc2(x))
        
        # Process through specialized branches
        easy_features = self.easy_branch(x)
        hard_features = self.hard_branch(x)
        
        # Output layers for each digit
        digit_outputs = []
        
        # Easy digits (hours + minutes tens)
        for i in range(3):
            digit_outputs.append(self.easy_outputs[i](easy_features))
        
        # Hard digits (minutes ones + seconds)
        for i in range(3):
            digit_outputs.append(self.hard_outputs[i](hard_features))
        
        return digit_outputs
        
class TimeDigitCNN(nn.Module):
    def __init__(self):
        super(TimeDigitCNN, self).__init__()
        
        # Input size: [batch_size, 1, 30, 150]
        # 1 channel (grayscale), 30 height, 150 width
        
        # Feature extraction layers
        # Conv1: (1, 30, 150) -> (32, 15, 75)
        self.conv1 = nn.Conv2d(1, 32, kernel_size=3, padding=1)
        self.bn1 = nn.BatchNorm2d(32)
        self.pool1 = nn.MaxPool2d(2, 2)
        
        # Conv2: (32, 15, 75) -> (64, 7, 37)
        self.conv2 = nn.Conv2d(32, 64, kernel_size=3, padding=1)
        self.bn2 = nn.BatchNorm2d(64)
        self.pool2 = nn.MaxPool2d(2, 2)
        
        # Conv3: (64, 7, 37) -> (128, 3, 18)
        self.conv3 = nn.Conv2d(64, 128, kernel_size=3, padding=1)
        self.bn3 = nn.BatchNorm2d(128)
        self.pool3 = nn.MaxPool2d(2, 2)
        
        # Dropout for regularization
        self.dropout1 = nn.Dropout2d(0.25)
        self.dropout2 = nn.Dropout2d(0.25)
        
        # Calculate the flattened size after convolutions and pooling
        # After 3 pooling layers: [batch_size, 128, 3, 18]
        self.fc_input_size = 128 * 3 * 18
        
        # Fully connected layers for shared feature extraction
        self.fc1 = nn.Linear(self.fc_input_size, 512)
        self.fc2 = nn.Linear(512, 256)
        
        # Output layers - one for each digit position (6 positions, 10 classes each)
        self.digit_layers = nn.ModuleList([nn.Linear(256, 10) for _ in range(6)])
        
    def forward(self, x):
        # Convolutional layers with batch normalization, pooling, and ReLU activation
        x = self.pool1(F.relu(self.bn1(self.conv1(x))))
        x = self.dropout1(x)
        
        x = self.pool2(F.relu(self.bn2(self.conv2(x))))
        x = self.dropout1(x)
        
        x = self.pool3(F.relu(self.bn3(self.conv3(x))))
        x = self.dropout2(x)
        
        # Flatten
        x = x.view(-1, self.fc_input_size)
        
        # Fully connected layers with ReLU activation
        x = F.relu(self.fc1(x))
        x = self.dropout2(x)
        x = F.relu(self.fc2(x))
        
        # Output layers for each digit
        digit_outputs = [digit_layer(x) for digit_layer in self.digit_layers]
        
        return digit_outputs

def create_model(device=None):
    """Create a new instance of the TimeDigitCNN model."""
    model = TimeDigitCNN()
    if device is not None:
        model = model.to(device)
    return model

def load_model(checkpoint_path, device=None, model_class=None):
    """Load a saved model from a checkpoint file."""
    if model_class is None:
        model_class = TimeDigitCNN
    model = model_class()
    if device is not None:
        model = model.to(device)
    checkpoint = torch.load(checkpoint_path, map_location=device)
    model.load_state_dict(checkpoint['model_state_dict'])
    return model
    
def save_model(model, optimizer=None, epoch=None, loss=None, accuracy=None, filepath='model.pth'):
    """Save model checkpoint with optional metadata."""
    checkpoint = {
        'model_state_dict': model.state_dict(),
    }
    
    # Add optional elements if provided
    if optimizer is not None:
        checkpoint['optimizer_state_dict'] = optimizer.state_dict()
    if epoch is not None:
        checkpoint['epoch'] = epoch
    if loss is not None:
        checkpoint['loss'] = loss
    if accuracy is not None:
        checkpoint['accuracy'] = accuracy
        
    torch.save(checkpoint, filepath)
    return filepath

def count_parameters(model):
    """Count the number of trainable parameters in the model."""
    return sum(p.numel() for p in model.parameters() if p.requires_grad)

def get_model_summary(model, input_size=(1, 30, 150)):
    """Generate a string summary of the model architecture."""
    # Create a dummy input to trace through the model
    batch_size = 1
    x = torch.zeros(batch_size, *input_size)
    
    # Build summary string
    summary = [f"TimeDigitCNN Model Summary:"]
    summary.append(f"Input shape: {input_size}")
    summary.append(f"Total parameters: {count_parameters(model):,}")
    summary.append(f"\nLayer structure:")
    summary.append(str(model))
    
    return "\n".join(summary)