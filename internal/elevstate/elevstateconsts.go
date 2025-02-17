package elevstate

const (
	N_FLOORS  = 4
	N_BUTTONS = 3
)

type Dirn int

func (d Dirn) ToString() string {
	switch d {
	case D_Up:
		return "D_Up"
	case D_Down:
		return "D_Down"
	case D_Stop:
		return "D_Stop"
	default:
		return "D_UNDEFINED"
	}
}

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

func (b Button) ToString() string {
	switch b {
	case B_HallUp:
		return "B_HallUp"
	case B_HallDown:
		return "B_HallDown"
	case B_Cab:
		return "B_Cab"
	default:
		return "B_UNDEFINED"
	}
}

type ElevatorBehaviour int

const (
	EB_Idle ElevatorBehaviour = iota // 0
	EB_DoorOpen
	EB_Moving
)

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

type ClearRequestVariant int

const (
	CV_All ClearRequestVariant = iota
	CV_InDirn
)
