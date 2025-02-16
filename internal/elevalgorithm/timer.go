package elevalgorithm

import (
	"time"
)

// Timer state variables

var timerActive bool
var timerEndTime time.Time

// Start the timer with a given duration in seconds
func Timer_start(duration time.Duration) {
	timerEndTime = time.Now().Add(duration)
	//fmt.Println("Timer started with endtime", timerEndTime)
	timerActive = true
}

// Stop the timer
func Timer_stop() {
	timerActive = false
}

// Check if the timer has timed out
func Timer_timedOut() bool {
	//fmt.Println(getWallTime(), timerEndTime)
	return timerActive && time.Now().After(timerEndTime)
}
