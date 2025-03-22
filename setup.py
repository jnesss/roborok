# setup.py
from setuptools import setup, find_packages

setup(
    name="roborok",
    version="0.1.0",
    packages=find_packages(),
    install_requires=[
        "requests>=2.28.0",
        "pillow>=9.0.0",
        "easyocr>=1.6.0",
        "numpy>=1.20.0",
    ],
    entry_points={
        "console_scripts": [
            "roborok=roborok.main:main_cli",
        ],
    },
)