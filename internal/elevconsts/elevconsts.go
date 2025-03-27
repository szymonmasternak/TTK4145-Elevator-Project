package elevconsts

const (
	N_FLOORS  = 4
	N_BUTTONS = 3
)

type Dirn int

func (d Dirn) String() string {
	switch d {
	case UP:
		return "up"
	case DOWN:
		return "down"
	case STOP:
		return "stop"
	default:
		return "Undefined"
	}
}

const (
	DOWN Dirn = -1
	STOP Dirn = 0
	UP   Dirn = 1
)

type Button int

const (
	HALL_UP Button = iota
	HALL_DOWN
	CAB
)

func (b Button) String() string {
	switch b {
	case HALL_UP:
		return "B_HallUp"
	case HALL_DOWN:
		return "B_HallDown"
	case CAB:
		return "B_Cab"
	default:
		return "B_UNDEFINED"
	}
}

type ElevatorBehaviour int

const (
	IDLE ElevatorBehaviour = iota // 0
	DOOR_OPEN
	MOVING
)

func (eb ElevatorBehaviour) String() string {
	switch eb {
	case IDLE:
		return "idle"
	case DOOR_OPEN:
		return "doorOpen"
	case MOVING:
		return "moving"
	default:
		return "EB_UNDEFINED"
	}
}

type ClearRequestVariant int

const (
	ALL ClearRequestVariant = iota
	IN_DIRN
)
