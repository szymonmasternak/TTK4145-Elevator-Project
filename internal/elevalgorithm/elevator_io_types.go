package elevalgorithm

const (
	N_FLOORS  = 4
	N_BUTTONS = 3
)

type Dirn int

const (
	D_Down Dirn = -1
	D_Stop Dirn = 0
	D_Up   Dirn = 1
)

type Button int

const (
	B_HallUp Button = iota
	B_HallDown
	B_Cab
)

// ElevOutputDevice struct replacing function pointers with function fields
type ElevOutputDevice struct {
	FloorIndicator     func(floor int)
	RequestButtonLight func(floor int, button Button, value int)
	DoorLight          func(value int)
	StopButtonLight    func(value int)
	MotorDirection     func(direction Dirn)
}
