package main

import (
    "syscall"
    "os"
    "path/filepath"
    "strings"
    "strconv"
    "time"
    // "fmt" // Used for debugging
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
    VK_SEVEN = 0x37
    VK_EIGHT = 0x38
    VK_NINE = 0x39
    VK_LMENU = 0xA4
)

// Creating return struct for each piece of config data
type configVal struct {
    windowX int
    windowY int
    startHeight int
}

/////////////////////////////////////////////////////////////////
/////////////////////////// MAIN FUNC ///////////////////////////
/////////////////////////////////////////////////////////////////

func main() {
    // Reading the necessary config file for user defined window size and placement   
    var configData []byte = readConfigFile()
    largeConfig, mediumConfig, smallConfig := manipConfigData(configData) // Returns 3x structs (1x struct for each window size)

    // Creating a time ticker (CLK)
    var ticker *time.Ticker = time.NewTicker(30 * time.Millisecond) // Check for keypress every 30ms (should be enough time to avoid using keyboard events whilst minimising resource usage)
    defer ticker.Stop() // Stopping the ticker if the function is stopped (Shouldn't ever occur)

    for range ticker.C {
        var shortcutPressed uint8 = shortcutKeysPressed()
        switch shortcutPressed {
        case 0: // Passing over this iteration (no shortcuts activated)
        case 1: // Activate large screen
            boilerplateScript(&mediumConfig, true) // mediumConfig used as 'fill-in' to avoid another function writeup
        case 2: // Large size wanted
            boilerplateScript(&largeConfig, false)
        case 3: // Medium size wanted
            boilerplateScript(&mediumConfig, false)
        case 4: // Small size wanted
            boilerplateScript(&smallConfig, false)
        }
    }
}

// Reducing boilerplate in the main func
func boilerplateScript(ptr_configSizeData *configVal, maximiseBool bool) {
    // Preprocessing - Windows API calls
    screenWidth, _ := getCurrMonitorRes() // Grabbing screen's working area pixel dimensions
    var currentProgHandle syscall.Handle = getForegroundWindow() // Grabbing the currently focused window ID

    // Calculating startWidth (middle) from current screen
    var startWidth int = int(screenWidth/2) - int((*ptr_configSizeData).windowX/2)

    // Changing the window size and location using a windows API call
    setWindowPos(currentProgHandle, startWidth, (*ptr_configSizeData).startHeight, (*ptr_configSizeData).windowX, (*ptr_configSizeData).windowY, maximiseBool)
}

/////////////////////////////////////////////////////////////////
/////////////////////////// MAIN FUNC ///////////////////////////
/////////////////////////////////////////////////////////////////

// Checks if the keyboard shortcut keys were pressed --> Invalid return 0 | [alt+shift+0:Maximise] return 1 | [alt+shift+9:Large] return 2 | [alt+shift+8:Medium] return 3 | [alt+shift+7:Small] return 4
func shortcutKeysPressed() uint8 {
    leftAltPressed, _, _ := procGetKeyState.Call(VK_LMENU) // Checking Left Alt

    // NOTE: Result 65409 or 65408 == key currently pressed (presumed: high-order bit is 1 & low-order bit is 0/1 --> 16-bit subtract a value?)
    if (leftAltPressed & 0x8000) != 0 { // Left Alt Pressed
        leftShiftPressed, _, _ := procGetKeyState.Call(VK_LSHIFT) // Checking Left Shift 
        if (leftShiftPressed & 0x8000) != 0 { // Left Shift Pressed
            zeroPressed, _, _ := procGetKeyState.Call(VK_ZERO) // Checking Zero
            if (zeroPressed & 0x8000) != 0 { // Zero Pressed --> Maximise Mode Selected
                return 1
            }
            // Nine Pressed --> Large Mode Selected
            ninePressed, _, _ := procGetKeyState.Call(VK_NINE) // Checking Nine,
            if (ninePressed & 0x8000) != 0 { // Nine Pressed --> Large Mode Selected
                return 2
            }
            // Eight Pressed --> Medium Mode Selected
            eightPressed, _, _ := procGetKeyState.Call(VK_EIGHT) // Checking Eight,
            if (eightPressed & 0x8000) != 0 { // Eight Pressed --> Medium Mode Selected
                return 3
            }
            // Seven Pressed --> Small Mode Selected
            sevenPressed, _, _ := procGetKeyState.Call(VK_SEVEN) // Checking Seven,
            if (sevenPressed & 0x8000) != 0 { // Seven Pressed --> Small Mode Selected
                return 4
            }
        }
    }
    return 0
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
func setWindowPos(currentProgHandle syscall.Handle, locX, locY, windowX, windowY int, maximiseBool bool) {
    // Set the current window to a "Normal" window to avoid issues with fullscreen not responding to procSetWindowPos
    const SW_SHOWNORMAL uintptr = uintptr(1) // If the window is minimised, maximised, or arranged, the system restores it to its original size and position.
    const SW_MAXIMISE uintptr = uintptr(3) // Flag for maximise
    if maximiseBool {
        maximiseResult, _, maxErr := procShowWindow.Call(uintptr(currentProgHandle), SW_MAXIMISE)
        if maximiseResult == 0 {
            var panicString string = "ERROR: " + maxErr.Error()
            panic(panicString) 
        }
    } else {
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
func manipConfigData(configData []byte) (configVal, configVal, configVal) {
    // Intialising the 3 size structs to hold necessary window sizing values
    var largeConfig configVal
    var mediumConfig configVal
    var smallConfig configVal

    // Grabbing configData lines
    var configString string = string(configData) // Converting byte array to string
    var configLines []string = strings.Split(configString, "\n") // Splitting string into line by line values
    
    // Viewing each line to extract relevant data
    for _, line := range configLines {
        var configLineSplit []string = strings.Split(line, ":") // Determining size in each line
        var configId string = strings.TrimSpace(configLineSplit[1]) // Obtaining identifier
        var configIntTrim string = strings.TrimSpace(configLineSplit[2]) // Obtaining value

        switch strings.TrimSpace(configLineSplit[0]) { // Determines the 'size' of the config value
        case "L":
            if configId == "startHeight" {
                sHLAsInt, err := strconv.Atoi(configIntTrim) // Converting string to int
                largeConfig.startHeight = sHLAsInt // Setting the struct value for function return
                if err != nil {
                    panic("ERROR: Failed to conv string --> int")
                }
            } else if configId == "windowX" {
                wXLAsInt, err := strconv.Atoi(configIntTrim) // Converting string to int
                largeConfig.windowX = wXLAsInt // Setting the struct value for function return
                if err != nil {
                    panic("ERROR: Failed to conv string --> int")
                }
            } else if configId == "windowY" {
                wYLAsInt, err := strconv.Atoi(configIntTrim) // Converting string to int
                largeConfig.windowY = wYLAsInt // Setting the struct value for function return
                if err != nil {
                    panic("ERROR: Failed to conv string --> int")
                }
            } else {
                var panicString string = "ERROR: Invalid L config recognised"
                panic(panicString)
            }
        case "M":
            if configId == "startHeight" {
                sHMAsInt, err := strconv.Atoi(configIntTrim) // Converting string to int
                mediumConfig.startHeight = sHMAsInt // Setting the struct value for function return
                if err != nil {
                    panic("ERROR: Failed to conv string --> int")
                }
            } else if configId == "windowX" {
                wXMAsInt, err := strconv.Atoi(configIntTrim) // Converting string to int
                mediumConfig.windowX = wXMAsInt // Setting the struct value for func return
                if err != nil {
                    panic("ERROR: Failed to conv string --> int")
                }
            } else if configId == "windowY" {
                wYMAsInt, err := strconv.Atoi(configIntTrim) // Converting string to int
                mediumConfig.windowY = wYMAsInt // Setting struct val for func return
                if err != nil {
                    panic("ERROR: Failed to conv string --> int")
                }
            } else {
                var panicString string = "ERROR: Invalid M config recognised"
                panic(panicString)
            }
        case "S":
            if configId == "startHeight" {
                sHSAsInt, err := strconv.Atoi(configIntTrim) // Converting string to int
                smallConfig.startHeight = sHSAsInt // Setting struct val for func return
                if err != nil {
                    panic("ERROR: Failed to conv string --> int")
                }
            } else if configId == "windowX" {
                wXSAsInt, err := strconv.Atoi(configIntTrim) // Converting string to int
                smallConfig.windowX = wXSAsInt // Setting struct val for func return
                if err != nil {
                    panic("ERROR: Failed to conv string --> int")
                }
            } else if configId == "windowY" {
                wYSAsInt, err := strconv.Atoi(configIntTrim) // Converting string to int
                smallConfig.windowY = wYSAsInt // Setting struct val for func return
                if err != nil {
                    panic("ERROR: Failed to conv string --> int")
                }
            } else {
                var panicString string = "ERROR: Invalid S config recognised"
                panic(panicString)
            }
        }
    }

    // Checking if all values have been successfully retrieved --> Ensure all values are intialised before progressing
    var checkL bool = ((largeConfig.startHeight != 0) && (largeConfig.windowX != 0) && (largeConfig.windowY != 0)) && true 
    var checkM bool = ((mediumConfig.startHeight != 0) && (mediumConfig.windowX != 0) && (mediumConfig.windowY != 0)) && true
    var checkS bool = ((smallConfig.startHeight != 0) && (smallConfig.windowX != 0) && (smallConfig.windowY != 0)) && true
    if !checkL || !checkM || !checkS { // If any values are missing, throw an error
        panic("ERROR: Failed to successfully retrieve all required dimensions")
    }

    return largeConfig, mediumConfig, smallConfig 
}