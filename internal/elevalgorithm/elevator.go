package elevalgorithm

import (
	"time"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/logger"
)

type ElevatorBehaviour int

const (
	EB_Idle ElevatorBehaviour = iota // 0
	EB_DoorOpen
	EB_Moving
)


type ClearRequestVariant int

const (
	CV_All ClearRequestVariant = iota
	CV_InDirn
)


type Elevator struct {
	Floor     int
	Dirn      Dirn
	Requests  [N_FLOORS][N_BUTTONS]int
	Behaviour ElevatorBehaviour
	clearRequestVariant ClearRequestVariant
	doorOpenDuration_s  time.Duration
}

// Method for printing the elevator state using logger
func (es Elevator) Print() {
	logger.Log.Info().Msgf("  +--------------------+")
	logger.Log.Info().Msgf(
		"  |floor = %-2d          |\n"+
			"  |dirn  = %-12s|\n"+
			"  |behav = %-12s|\n",
		es.Floor,
		es.Dirn.ToString(),
		ElevatorBehaviour.toString(es.Behaviour),
	)
	logger.Log.Info().Msgf("  +--------------------+")
	logger.Log.Info().Msgf("  |  | up  | dn  | cab |")

	for f := N_FLOORS - 1; f >= 0; f-- {
		logger.Log.Info().Msgf("  | %d", f)
		for btn := 0; btn < N_BUTTONS; btn++ {
			if (f == N_FLOORS-1 && btn == int(B_HallUp)) || (f == 0 && btn == int(B_HallDown)) {
				logger.Log.Info().Msgf("|     ")
			} else {
				if es.Requests[f][btn] != 0 {
					logger.Log.Info().Msgf("|  #  ")
				} else {
					logger.Log.Info().Msgf("|  -  ")
				}
			}
		}
		logger.Log.Info().Msgf("|")
	}
	logger.Log.Info().Msgf("  +--------------------+")
}

// Method to create an uninitialized elevator
func (e *Elevator) init(doorOpenDuration time.Duration, requestVariant ClearRequestVariant ) Elevator {
	return Elevator{
		Floor:     -1,
		Dirn:      D_Stop,
		Behaviour: EB_Idle,
		clearRequestVariant: requestVariant,
		doorOpenDuration_s:  doorOpenDuration,
	}
}

// Method for converting ElevatorBehaviour to string
func (eb ElevatorBehaviour) toString() string {
	switch eb {
	case EB_Idle:
		return "EB_Idle"
	case EB_DoorOpen:
		return "EB_DoorOpen"
	case EB_Moving:
		return "EB_Moving"
	default:
		return "EB_UNDEFINED"
	}
}
