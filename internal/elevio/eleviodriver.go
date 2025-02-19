package elevio

import (
	"errors"
	"net"
	"sync"
	"time"
)

const pollRate = 20 * time.Millisecond

type MotorDirection int
type ButtonType int

type ButtonEvent struct {
	Floor  int
	Button ButtonType
}

type ElevIODriver struct {
	conn        net.Conn
	mtx         sync.Mutex
	numFloors   int
	initialized bool
}

const (
	MDUp   MotorDirection = 1
	MDDown MotorDirection = -1
	MDStop MotorDirection = 0
)

const (
	BTHallUp   ButtonType = 0
	BTHallDown ButtonType = 1
	BTCab      ButtonType = 2
)

func NewElevIODriver(addr string, numFloors int) (*ElevIODriver, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, errors.New("failed to connect to elevator")
	}

	//Goroutine to stop the program when disconnected from elevator
	go func() {
		buffer := make([]byte, 1)
		for {
			_, err := conn.Read(buffer)
			if err != nil {
				panic("Elevator connection lost: " + err.Error())
			}
			time.Sleep(time.Millisecond) //Lets not overload the processor by pinging too fast
		}
	}()

	return &ElevIODriver{
		conn:      conn,
		numFloors: numFloors,
	}, nil
}

func (e *ElevIODriver) SetMotorDirection(dir MotorDirection) {
	e.write([4]byte{1, byte(dir), 0, 0})
}

func (e *ElevIODriver) SetButtonLamp(button ButtonType, floor int, value bool) {
	e.write([4]byte{2, byte(button), byte(floor), toByte(value)})
}

func (e *ElevIODriver) SetFloorIndicator(floor int) {
	e.write([4]byte{3, byte(floor), 0, 0})
}

func (e *ElevIODriver) SetDoorOpenLamp(value bool) {
	e.write([4]byte{4, toByte(value), 0, 0})
}

func (e *ElevIODriver) SetStopLamp(value bool) {
	e.write([4]byte{5, toByte(value), 0, 0})
}

func (e *ElevIODriver) PollButtons(receiver chan<- ButtonEvent) {
	prev := make([][3]bool, e.numFloors)
	for {
		time.Sleep(pollRate)
		for f := 0; f < e.numFloors; f++ {
			for b := ButtonType(0); b < 3; b++ {
				v := e.GetButton(b, f)
				if v != prev[f][b] && v {
					receiver <- ButtonEvent{f, b}
				}
				prev[f][b] = v
			}
		}
	}
}

func (e *ElevIODriver) GetButton(button ButtonType, floor int) bool {
	resp := e.read([4]byte{6, byte(button), byte(floor), 0})
	return toBool(resp[1])
}

func (e *ElevIODriver) GetFloor() int {
	resp := e.read([4]byte{7, 0, 0, 0})
	if resp[1] != 0 {
		return int(resp[2])
	}
	return -1
}

func (e *ElevIODriver) write(in [4]byte) {
	e.mtx.Lock()
	defer e.mtx.Unlock()

	_, err := e.conn.Write(in[:])
	if err != nil {
		panic("Lost connection to Elevator Server")
	}
}

func (e *ElevIODriver) read(in [4]byte) [4]byte {
	e.mtx.Lock()
	defer e.mtx.Unlock()

	_, err := e.conn.Write(in[:])
	if err != nil {
		panic("Lost connection to Elevator Server")
	}

	var out [4]byte
	_, err = e.conn.Read(out[:])
	if err != nil {
		panic("Lost connection to Elevator Server")
	}

	return out
}

func toByte(a bool) byte {
	if a {
		return 1
	}
	return 0
}

func toBool(a byte) bool {
	return a != 0
}
