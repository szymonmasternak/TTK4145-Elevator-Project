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
