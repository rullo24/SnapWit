package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
)

// global variable mutex to avoid race conditions
var sync_mu sync.Mutex

// function for readability in code
func accept_and_clean_user_input(p_avail_resp_keys *[]rune) (rune, error) {
	// converting runes to bytes (if available) --> throw error if there are any UNICODE chars
	var avail_resp_keys_ascii []byte
	sync_mu.Lock()
	for _, key := range *p_avail_resp_keys {
		if !rune_is_ascii_compat(key) { // fails if rune cannot be converted to ASCII --> should not happen by keyboard presses
			return 0x0, errors.New("could not convert rune to ASCII")
		}
		avail_resp_keys_ascii = append(avail_resp_keys_ascii, byte(key)) // adding the byte (ASCII) char to the list of available keys
	}
	sync_mu.Unlock()

	// change terminal mode to raw to read single bytes
	old_state, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return 0x0, err
	}
	defer term.Restore(int(os.Stdin.Fd()), old_state) // Ensure terminal mode is restored after reading

	// runs infinitely until a valid ASCII key is provided
	for {
		var curr_char byte
		var temp_buf []byte = make([]byte, 1)      // single byte slice buffer
		_, char_get_err := os.Stdin.Read(temp_buf) // read a single byte into curr_char
		if char_get_err != nil {
			return 0x0, char_get_err // return because of error
		}
		curr_char = temp_buf[0] // assigning curr char as the only byte in the buffer

		for _, key := range avail_resp_keys_ascii {
			if curr_char == key {
				return rune(key), nil // return the found key
			}
		}
	}
}

func rune_is_ascii_compat(target_r rune) bool {
	return target_r >= 0 && target_r <= 127
}

// updating the current time --> infinite run until program close
func constant_update_current_time(ctx context.Context, p_current_time_obj *time.Time) { // runs in a separate goroutine
	for {
		select {
		case <-ctx.Done(): // stop goroutine (called at end of program)
			return
		default: // running as usual unless stop is asked
			time.Sleep(10 * time.Millisecond) // updating every certain length of time
			sync_mu.Lock()
			*p_current_time_obj = time.Now()
			sync_mu.Unlock()
		}
	}
}

// run in the background in a goroutine --> printing time if available every so often
func print_time_since_stopwatch_start(ctx context.Context, p_curr_time *time.Time, p_timer_start_time *time.Time, p_should_print_flag *bool) {
	var time_diff time.Duration
	for {
		select {
		case <-ctx.Done():
			return
		default:
			time.Sleep(5 * time.Millisecond) // print every 500ms
			sync_mu.Lock()
			if !(*p_timer_start_time).Equal(time.Unix(0, 0)) && *p_should_print_flag { // only continue if the timer is set (UNIX == 0 means not set)
				time_diff = (*p_curr_time).Sub(*p_timer_start_time) // finding time diff between current time and timer start
				var diff_hours uint64 = uint64(time_diff.Hours())
				var diff_mins uint64 = uint64(time_diff.Minutes()) % 60      // modulus used to only look at the remainder since last full hour
				var diff_secs uint64 = uint64(time_diff.Seconds()) % 60      // modulus used to only look at the remainder since last full minute
				var diff_ms uint64 = uint64(time_diff.Milliseconds()) % 1000 // modulus used to only look at the remainder since last full minute

				fmt.Printf("\rSince Start: %02d:%02d:%02d.%03d", diff_hours, diff_mins, diff_secs, diff_ms) // printing in correct format (overwriting prev line)
			}
			sync_mu.Unlock()
		}
	}
}

// run in the background in a goroutine --> printing remaining time if available every so often
func print_remaining_timer(ctx context.Context, p_curr_time *time.Time, p_user_set_timer_duration *time.Duration, p_timer_end_time *time.Time, p_should_print_flag *bool, beep_flag bool) {
	var time_diff time.Duration
	for {
		select {
		case <-ctx.Done():
			return
		default:
			time.Sleep(5 * time.Millisecond) // print every 500ms
			sync_mu.Lock()
			if !((*p_user_set_timer_duration) == 0) && (*p_curr_time).Before(*p_timer_end_time) && *p_should_print_flag { // only continue if the timer is set (UNIX == 0 means not set)
				time_diff = (*p_timer_end_time).Sub(*p_curr_time) // finding remaining time in timer (length until)
				var remaining_hours uint64 = uint64(time_diff.Hours())
				var remaining_mins uint64 = uint64(time_diff.Minutes()) % 60                               // modulus used to only look at the remainder since last full hour
				var remaining_secs uint64 = uint64(time_diff.Seconds()) % 60                               // modulus used to only look at the remainder since last full minute
				fmt.Printf("\rRemaining: %02d:%02d:%02d", remaining_hours, remaining_mins, remaining_secs) // printing in correct format (overwriting prev line)
			} else if (*p_curr_time).After(*p_timer_end_time) && *p_should_print_flag && beep_flag { // beeping sound at the end of the timer
				*p_should_print_flag = false // Stop printing

				// playing a beep through the computer speakers to symbolise the beep
				var cmd *exec.Cmd // init executable command obj

				switch runtime.GOOS { // checking if the current operating system for beep
				case "windows":
					cmd = exec.Command("powershell", "-c", "for ($i=0; $i -lt 3; $i++) { [console]::beep(750, 300); Start-Sleep -Milliseconds 100 }") // 1000Hz sound 3x (500ms breaks)
				case "darwin":
					// Use the afplay command to play a built-in sound on macOS
					cmd = exec.Command("afplay", "/System/Library/Sounds/Glass.aiff")
				case "linux":
					// Use the aplay command to play a built-in sound on Linux
					cmd = exec.Command("aplay", "/usr/share/sounds/alsa/Front_Center.wav") // Change this to your preferred sound file
				default:
					return
				}
				if err := cmd.Run(); err != nil {
					log.Println("ERROR: could not play the beep sound")
					panic(err)
				}
			}
			sync_mu.Unlock()
		}
	}
}

func main() {
	// init vars
	var beep_flag bool = true
	log.SetOutput(os.Stderr)                                    // pretty sure it writes to this by default
	ctx, ctx_cancel := context.WithCancel(context.Background()) // cancel at end of program to close all goroutines
	defer ctx_cancel()                                          // closing all goroutines on program end
	var current_time_obj time.Time                              // init current time

	// asking user what is to occur (stopwatch or timer)
	fmt.Print("=============================\n")
	fmt.Println("s --> Choose Stopwatch")
	fmt.Println("t --> Choose Timer")
	fmt.Println("q --> Quit")
	fmt.Print("=============================\n")

	var avail_base_keys []rune = []rune{'s', 't', 'q'}
	stop_or_timer_choice, user_input_err := accept_and_clean_user_input(&avail_base_keys)
	if user_input_err != nil {
		log.Println("ERROR:", user_input_err.Error())
		return
	}

	// going into selections
	if stop_or_timer_choice == 's' {
		var stopwatch_start_time time.Time = time.Unix(0, 0) // init as UNIX==0 so that the goroutines can see this as "not set"
		var should_print_flag bool = false

		// print help information
		fmt.Print("\n")
		fmt.Print("################\n")
		fmt.Print("$$$ STOPWATCH $$$\n")
		fmt.Println("s --> Start Stopwatch")
		fmt.Println("e --> End Stopwatch")
		fmt.Println("r --> Reset Stopwatch")
		fmt.Println("q --> Quit")
		fmt.Print("################\n\n")

		// goroutine to consistently print the current time
		go constant_update_current_time(ctx, &current_time_obj)
		go print_time_since_stopwatch_start(ctx, &current_time_obj, &stopwatch_start_time, &should_print_flag) // only prints if the stopwatch_start_time exists

		// infinite loop --> waiting for user input
		var avail_resp_keys []rune = []rune{'s', 'e', 'r', 'q'}
		for {
			valid_r, char_get_err := accept_and_clean_user_input(&avail_resp_keys)
			if char_get_err != nil {
				log.Println("ERROR:", char_get_err.Error())
				return
			}

			switch valid_r {
			case 's':
				should_print_flag = true
				if stopwatch_start_time.Equal(time.Unix(0, 0)) { // checking if the stopwatch is not currently active
					stopwatch_start_time = current_time_obj // setting the stopwatch start time to the current time
				}
			case 'e':
				should_print_flag = false
			case 'r':
				stopwatch_start_time = time.Unix(0, 0)                       // setting to UNIX time --> tells the funcs that the stopwatch hasn't started
				fmt.Printf("\rSince Start: %02d:%02d:%02d.%03d", 0, 0, 0, 0) // printing in correct format (overwriting prev line)
			case 'q':
				return
			default:
				continue // do nothing if byte is not one of these keys
			}
		}

	} else if stop_or_timer_choice == 't' {
		// init timer-related variables
		var timer_end_time time.Time = time.Unix(0, 0)
		var user_set_timer_duration time.Duration = 0
		var remaining_timer_duration time.Duration = 0
		var should_print_flag bool = false

		// print help information
		fmt.Print("\n")
		fmt.Print("################\n")
		fmt.Print("$$$ TIMER $$$\n")
		fmt.Println("u --> Update Timer")
		fmt.Println("s --> Start Timer")
		fmt.Println("e --> End Timer")
		fmt.Println("r --> Reset Timer")
		fmt.Println("q --> Quit")
		fmt.Print("################\n\n")

		// goroutine to consistently print the current time
		go constant_update_current_time(ctx, &current_time_obj)
		go print_remaining_timer(ctx, &current_time_obj, &user_set_timer_duration, &timer_end_time, &should_print_flag, beep_flag) // only prints if the stopwatch_start_time exists

		// infinite loop --> waiting for user input
		var avail_resp_keys []rune = []rune{'u', 's', 'e', 'r', 'q'}
		for {
			valid_r, char_get_err := accept_and_clean_user_input(&avail_resp_keys)
			if char_get_err != nil {
				log.Println("ERROR:", char_get_err.Error())
				return
			}

			switch valid_r {
			case 'u':
				// getting user input
				fmt.Print("\033[A\033[K")
				fmt.Print("\rEnter time in format HH:mm:SS and press enter (e.g., 02:15:30): ")
				var new_timer_string string
				_, scan_err := fmt.Scanf("%s", &new_timer_string)
				if scan_err != nil {
					log.Println("ERROR:", scan_err.Error())
					return
				}

				// checking user input --> print message if poor format parsed
				new_timer_string = strings.TrimSpace(new_timer_string)                 // removing all spaces on the ends
				var timer_string_parts []string = strings.Split(new_timer_string, ":") // splitting on the colons
				if len(timer_string_parts) != 3 {
					log.Println("ERROR: Invalid string format. Must adhere to HH:mm:SS")
					continue
				}

				// grabbing the time as valid integers from the parsed user string
				hours, err := strconv.Atoi(timer_string_parts[0])
				if err != nil {
					log.Println("Invalid hours value:", err)
					continue
				}
				minutes, err := strconv.Atoi(timer_string_parts[1])
				if err != nil {
					log.Println("Invalid minutes value:", err)
					continue
				}
				seconds, err := strconv.Atoi(timer_string_parts[2])
				if err != nil {
					log.Println("Invalid seconds value:", err)
					continue
				}

				// creating a duration object for the timer from the user parsed string
				user_set_timer_duration = time.Duration(hours)*time.Hour +
					time.Duration(minutes)*time.Minute +
					time.Duration(seconds)*time.Second

				// printing the new duration (even though its stopped)
				fmt.Printf("\rRemaining: %02d:%02d:%02d",
					int64(user_set_timer_duration.Hours()), int64(user_set_timer_duration.Minutes())%60,
					int64(user_set_timer_duration.Seconds())%60) // printing in correct format (overwriting prev line)

			case 's':
				// check remaining time --> add to the current time to find the new stopping time
				should_print_flag = true
				if remaining_timer_duration > 0 {
					timer_end_time = current_time_obj.Add(remaining_timer_duration)
					remaining_timer_duration = 0
				}

				// starting a new timer
				if timer_end_time.Equal(time.Unix(0, 0)) || timer_end_time.Before(current_time_obj) { // checking if the stopwatch is not currently active
					timer_end_time = current_time_obj.Add(user_set_timer_duration) // setting the stopwatch start time to the current time
				}
			case 'e':
				should_print_flag = false
				remaining_timer_duration = timer_end_time.Sub(current_time_obj)

			case 'r':
				should_print_flag = false
				timer_end_time = time.Unix(0, 0) // setting to UNIX time --> tells the funcs that the stopwatch hasn't started
				remaining_timer_duration = 0
				fmt.Printf("\rRemaining: %02d:%02d:%02d",
					int64(user_set_timer_duration.Hours()), int64(user_set_timer_duration.Minutes())%60,
					int64(user_set_timer_duration.Seconds())%60) // printing in correct format (overwriting prev line)
			case 'q':
				return
			default:
				continue // do nothing if byte is not one of these keys
			}
		}

	}
}
