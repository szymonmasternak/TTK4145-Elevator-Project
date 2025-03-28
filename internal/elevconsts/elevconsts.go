package elevconsts

const (
	N_FLOORS  = 4
	N_BUTTONS = 3
)

type Dirn int

func (d Dirn) String() string {
	switch d {
	case Up:
		return "up"
	case Down:
		return "down"
	case Stop:
		return "stop"
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
		return "idle"
	case DoorOpen:
		return "doorOpen"
	case Moving:
		return "moving"
	default:
		return "EB_UNDEFINED"
	}
}

type ClearRequestVariant int

const (
	All ClearRequestVariant = iota
	InDirn
)

// RequestState represents the state of a request.
type RequestState int

const (
	REQ_Initial     RequestState = -1
	REQ_None        RequestState = 0
	REQ_Unconfirmed RequestState = 1
	REQ_Confirmed   RequestState = 2
	REQ_Completed   RequestState = 3
)

// Request holds the state and list of nodes that have confirmed the request.
type Request struct {
	State          RequestState `json:"state"`
	ConsensusPeers []string     `json:"consensus"`
}

// RequestArray is a two-dimensional array of requests.
type RequestArray [N_FLOORS][N_BUTTONS]Request

// RequestConfirmationMap maps a node identifier to its RequestArray.
type RequestConfirmationMap map[string]RequestArray

// RequestMessage is used for local button press or state changes.
type RequestMessage struct {
	Floor  int
	Button Button
	State  RequestState
}

// RequestArrayMessage is used to exchange the entire RequestArray between nodes.
type RequestArrayMessage struct {
	Identifier   string       `json:"id"`
	RequestArray RequestArray `json:"reqArray"`
}
