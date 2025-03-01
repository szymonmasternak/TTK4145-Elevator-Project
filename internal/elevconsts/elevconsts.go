package elevconsts

const (
	N_FLOORS  = 4
	N_BUTTONS = 3
)

type Dirn int

func (d Dirn) String() string {
	switch d {
	case Up:
		return "Up"
	case Down:
		return "Down"
	case Stop:
		return "Stop"
	default:
		return "Undefined"
	}
}

const (
	Down Dirn = -1
	Stop Dirn = 0
	Up   Dirn = 1
)

type Button int

const (
	HallUp Button = iota
	HallDown
	Cab
)

func (b Button) String() string {
	switch b {
	case HallUp:
		return "B_HallUp"
	case HallDown:
		return "B_HallDown"
	case Cab:
		return "B_Cab"
	default:
		return "B_UNDEFINED"
	}
}

type ElevatorBehaviour int

const (
	Idle ElevatorBehaviour = iota // 0
	DoorOpen
	Moving
)

func (eb ElevatorBehaviour) String() string {
	switch eb {
	case Idle:
		return "EB_Idle"
	case DoorOpen:
		return "EB_DoorOpen"
	case Moving:
		return "EB_Moving"
	default:
		return "EB_UNDEFINED"
	}
}

type ClearRequestVariant int

const (
	All ClearRequestVariant = iota
	InDirn
)
