# SnapFit
NOTE: This currently only works on Windows

## Basic Description
This program resizes the currently focused window to dimensions specified in a configuration file when the "alt" + "shift" + "0" keys are pressed. Created out of necessity, this program was produced as I found myself wasting quite a lot of time resizing windows on my Ultrawide monitor (e.g. opening a 2nd VS Code would open a very small window).

## Requirements
- Windows 10/11
- Go 1.21.4 (or sooner)

## How it Works
This program works by checking every 25ms for the pressed shortcut keys. If the required shortcut keys are pressed, the program then resizes the currently focused window to match the settings detailed in the config file.

## Installation
Download the source code.
Open a terminal in the directory containing the source code.
Run the following command to build the program:
```bash
go build -ldflags -H=windowsgui
```

### Running from Startup
To ensure that the program is always running in the background (checking for key presses), open "Run" from the Windows Start Menu or press WindowsKey+r. Once this "Run" window opens, enter in the following command to open the Startup folder:
```bash
shell:startup
```
Paste a shortcut created from the compiled executable in this Startup folder. The program will now run at startup (the program starts after all other necessary processes boot).

## Configuration
All lines must follow the convention as shown below:
```bash
key:value
```
For the program to function, "startHeight", "windowX" and "windowY" keys are required. All other keys will be required.
- startHeight defines the resized window's top distance from the top of the working area
- windowX defines the width of the new window
- windowY defines the height of the new window