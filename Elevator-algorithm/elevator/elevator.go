package elevator

import "fmt"

// Enum equivalent for ElevatorBehaviour
type ElevatorBehaviour int

const (
	EB_Idle ElevatorBehaviour = iota // 0
	EB_DoorOpen
	EB_Moving
)

// Enum equivalent for ClearRequestVariant
type ClearRequestVariant int

const (
	CV_All ClearRequestVariant = iota
	CV_InDirn
)

// Elevator struct equivalent
type Elevator struct {
	Floor     int
	Dirn      Dirn
	Requests  [N_FLOORS][N_BUTTONS]int
	Behaviour ElevatorBehaviour
	Config    struct {
		clearRequestVariant ClearRequestVariant
		doorOpenDuration_s  float64
	}
}

// Function equivalent to elevator_print
func elevator_print(es Elevator) {
	fmt.Println("  +--------------------+")
	fmt.Printf(
		"  |floor = %-2d          |\n"+
			"  |dirn  = %-12s|\n"+
			"  |behav = %-12s|\n",
		es.Floor,
		Elevio_dirn_toString(es.Dirn),
		eb_toString(es.Behaviour),
	)
	fmt.Println("  +--------------------+")
	fmt.Println("  |  | up  | dn  | cab |")

	for f := N_FLOORS - 1; f >= 0; f-- {
		fmt.Printf("  | %d", f)
		for btn := 0; btn < N_BUTTONS; btn++ {
			if (f == N_FLOORS-1 && btn == int(B_HallUp)) || (f == 0 && btn == int(B_HallDown)) {
				fmt.Print("|     ")
			} else {
				if es.Requests[f][btn] != 0 {
					fmt.Print("|  #  ")
				} else {
					fmt.Print("|  -  ")
				}
			}
		}
		fmt.Println("|")
	}
	fmt.Println("  +--------------------+")
}

// Function equivalent to elevator_uninitialized
func elevator_uninitialized() Elevator {
	return Elevator{
		Floor:     -1,
		Dirn:      D_Stop,
		Behaviour: EB_Idle,
		Config: struct {
			clearRequestVariant ClearRequestVariant
			doorOpenDuration_s  float64
		}{
			clearRequestVariant: CV_InDirn,
			doorOpenDuration_s:  3.0,
		},
	}
}

// Function equivalent to eb_toString
func eb_toString(eb ElevatorBehaviour) string {
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
