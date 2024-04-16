package main

import (
    "syscall"
    "os"
    "path/filepath"
    "strings"
    "strconv"
    "time"
)

// Defining windows function and dll pointers
var (
    user32 = syscall.NewLazyDLL("user32.dll")
    procGetForegroundWindow = user32.NewProc("GetForegroundWindow")
    procSetWindowPos = user32.NewProc("SetWindowPos")
    procGetSystemMetrics = user32.NewProc("GetSystemMetrics")
    procShowWindow = user32.NewProc("ShowWindow")
    procGetKeyState = user32.NewProc("GetKeyState")
)

// Virtual Key Codes 
const (
    VK_LSHIFT = 0x10
    VK_ZERO = 0x30
    VK_LMENU = 0xA4
)

/////////////////////////////////////////////////////////////////
/////////////////////////// MAIN FUNC ///////////////////////////
/////////////////////////////////////////////////////////////////

func main() {
    // Reading the necessary config file for user defined window size and placement   
    var configData []byte = readConfigFile()
    windowX, windowY, startHeight := manipConfigData(configData)

    // Creating a time ticker (CLK)
    var ticker *time.Ticker = time.NewTicker(50 * time.Millisecond) // Check for keypress every 50ms (should be enough time to avoid using keyboard events whilst minimising resource usage)
    defer ticker.Stop() // Stopping the ticker if the function is stopped (Shouldn't ever occur)

    for range ticker.C {
        if shortcutKeysPressed() { // Change the window's screen size/placement if all required shortcut keys are pressed
            // Preprocessing - Windows API calls
            screenWidth, _ := getCurrMonitorRes() // Grabbing screen's working area pixel dimensions
            var currentProgHandle syscall.Handle = getForegroundWindow() // Grabbing the currently focused window ID

            // Calculating startWidth (middle) from current screen
            var startWidth int = int(screenWidth/2) - int(windowX/2)

            // Changing the window size and location using a windows API call
            setWindowPos(currentProgHandle, startWidth, startHeight, windowX, windowY)
        }
    }
}

/////////////////////////////////////////////////////////////////
/////////////////////////// MAIN FUNC ///////////////////////////
/////////////////////////////////////////////////////////////////

// Checks if the keyboard shortcut keys were pressed --> alt+shift+0
func shortcutKeysPressed() bool {
    leftAltPressed, _, _ := procGetKeyState.Call(VK_LMENU) // Checking Left Alt

    // NOTE: Result 65409 or 65408 == key currently pressed (presumed: high-order bit is 1 & low-order bit is 0/1 --> 16-bit subtract a value?)
    if (leftAltPressed & 0x8000) != 0 { // Left Alt Pressed
        leftShiftPressed, _, _ := procGetKeyState.Call(VK_LSHIFT) // Checking Left Shift 
        if (leftShiftPressed & 0x8000) != 0 { // Left Shift Pressed
            zeroPressed, _, _ := procGetKeyState.Call(VK_ZERO) // Checking Zero
            if (zeroPressed & 0x8000) != 0 { // Zero Pressed
                return true // All shortcuts are currently pressed
            }
        }
    }
    return false
}

// Returns a string that is the directory of the main executable
func getDirOfMainExe() string {
	exeLoc, _ := os.Executable() // Error not captured because it is useless
	var exeDirLoc string = filepath.Dir(exeLoc)
	
	return exeDirLoc
}

// Returns the window that is currently in the foreground
func getForegroundWindow() syscall.Handle {
    hwnd, _, foregroundErr := procGetForegroundWindow.Call()
    if hwnd == 0 {
        var panicString string = "ERROR: " + foregroundErr.Error()
        panic(panicString)
    }
    return syscall.Handle(hwnd)
}

// Returns the resolution of the current monitor --> Returns "workarea", not total resolution
func getCurrMonitorRes() (int, int) {
    // Defining constants for function calls
    const SM_CXSCREEN uintptr = uintptr(0) // Register val for screen res width
    const SM_CYSCREEN uintptr = uintptr(1) // Register val for screen res height

    screenWidthResult, _, widthErr := procGetSystemMetrics.Call(SM_CXSCREEN)
    if screenWidthResult == 0 { // Resultant outcome if an error occurs
        var panicString string = "ERROR: " + widthErr.Error()
        panic(panicString)
    }
    screenHeightResult, _, heightErr := procGetSystemMetrics.Call(SM_CYSCREEN)
    if screenHeightResult == 0 { // Resultant outcome if an error occurs
        var panicString string = "ERROR: " + heightErr.Error()
        panic(panicString)
    }
    return int(screenWidthResult), int(screenHeightResult) 
}

// Given a window (hwnd), this changes the windows position
func setWindowPos(currentProgHandle syscall.Handle, locX, locY, windowX, windowY int) {
    // Set the current window to a "Normal" window to avoid issues with fullscreen not responding to procSetWindowPos
    const SW_SHOWNORMAL uintptr = uintptr(1) // If the window is minimised, maximised, or arranged, the system restores it to its original size and position.
    showWindowResult, _, showErr := procShowWindow.Call(uintptr(currentProgHandle), SW_SHOWNORMAL)
    if showWindowResult == 0 {
        var panicString string = "ERROR: " + showErr.Error()
        panic(panicString) 
    }

    moveResult, _, moveErr := procSetWindowPos.Call(
        uintptr(currentProgHandle),
        0, // hWndInsertAfter (Optional) --> HWND_TOP = 0
        uintptr(locX),
        uintptr(locY),
        uintptr(windowX),
        uintptr(windowY),
        0x0040, // uFlag: SWP_SHOWWINDOW
    )

    if moveResult == 0 {
        var panicString string = "ERROR: " + moveErr.Error()
        panic(panicString)
    }
}

// Returns the config file bytes
func readConfigFile() []byte {
    // Finding the location of the config file
    var mainExecDir string = getDirOfMainExe() // Returns the main executable's parent directory
    var configLoc string = filepath.Join(mainExecDir, "config.txt") // Creating a string to resemble the config file

    // Reading and manipulating config file
    configData, configErr := os.ReadFile(configLoc)
    if configErr != nil {
        var panicString string = "ERROR: " + configErr.Error()
        panic(panicString)
    }

    return configData
}

// Manipulates config file to extract the necessary integers we want
func manipConfigData(configData []byte) (int, int, int) {
    // Initialising the necessary return variables
    var windowX int = -1
    var windowY int = -1
    var startHeight int = -1

    // Grabbing configData lines
    var configString string = string(configData) // Converting byte array to string
    var configLines []string = strings.Split(configString, "\n") // Splitting string into line by line values
    
    // Viewing each line to extract relevant data
    for _, line := range configLines {
        var idAndValSplit []string = strings.Split(line, ":")
        switch strings.TrimSpace(idAndValSplit[0]) {
            case "startHeight":
                var intStringTrim string = strings.TrimSpace(idAndValSplit[1])
                shResult, atoiErr := strconv.Atoi(intStringTrim)
                if atoiErr != nil {
                    var panicString string = "ERROR: " + atoiErr.Error()
                    panic(panicString)
                }
                startHeight = shResult // Assigning the return value

            case "windowX":
                var intStringTrim string = strings.TrimSpace(idAndValSplit[1])
                wxResult, atoiErr := strconv.Atoi(intStringTrim)
                if atoiErr != nil {
                    var panicString string = "ERROR: " + atoiErr.Error()
                    panic(panicString)
                }
                windowX = wxResult // Assigning the return value

            case "windowY":
                var intStringTrim string = strings.TrimSpace(idAndValSplit[1])
                wyResult, atoiErr := strconv.Atoi(intStringTrim)
                if atoiErr != nil {
                    var panicString string = "ERROR: " + atoiErr.Error()
                    panic(panicString)
                }

                windowY = wyResult // Assigning the return value
        }
    }

    // Checking if all values have been successfully retrieved
    if windowX == -1 || windowY == -1 || startHeight == -1 {
        panic("ERROR: Failed to successfully retrieve all required dimensions")
    }

    return windowX, windowY, startHeight
}

