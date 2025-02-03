package elevator

import (
	"time"
)

// Timer state variables
var timerEndTime float64
var timerActive bool

// Equivalent to `get_wall_time()`: Get current time in seconds
func getWallTime() float64 {
	return float64(time.Now().UnixNano()) / 1e9
}

// Start the timer with a given duration in seconds
func Timer_start(duration float64) {
	timerEndTime = getWallTime() + duration
	//fmt.Println("Timer started with endtime", timerEndTime)
	timerActive = true
}

// Stop the timer
func Timer_stop() {
	timerActive = false
}

// Check if the timer has timed out
func Timer_timedOut() bool {
	//sfmt.Println(getWallTime(), timerEndTime)
	return timerActive && getWallTime() > timerEndTime
}
