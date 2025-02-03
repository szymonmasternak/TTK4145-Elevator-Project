package elevator

// Constants replacing #define
const (
	N_FLOORS  = 4
	N_BUTTONS = 3
)

// Enum equivalent for Dirn (Direction)
type Dirn int

const (
	D_Down Dirn = -1
	D_Stop Dirn = 0
	D_Up   Dirn = 1
)

// Enum equivalent for Button
type Button int

const (
	B_HallUp Button = iota
	B_HallDown
	B_Cab
)

// ElevInputDevice struct replacing function pointers with function fields
type ElevInputDevice struct {
	FloorSensor   func() int
	RequestButton func(floor int, button Button) int
	StopButton    func() int
	Obstruction   func() int
}

// ElevOutputDevice struct replacing function pointers with function fields
type ElevOutputDevice struct {
	FloorIndicator     func(floor int)
	RequestButtonLight func(floor int, button Button, value int)
	DoorLight          func(value int)
	StopButtonLight    func(value int)
	MotorDirection     func(direction Dirn)
}

/* REDUNDANT
// Function to convert Dirn to string
func elevio_dirn_toString(d Dirn) string {
	switch d {
	case D_Down:
		return "Down"
	case D_Stop:
		return "Stop"
	case D_Up:
		return "Up"
	default:
		return "Unknown"
	}
}

// Function to convert Button to string
func elevio_button_toString(b Button) string {
	switch b {
	case B_HallUp:
		return "Hall Up"
	case B_HallDown:
		return "Hall Down"
	case B_Cab:
		return "Cab"
	default:
		return "Unknown"
	}
}
*/
