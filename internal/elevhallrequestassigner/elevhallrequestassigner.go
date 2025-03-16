package elevhallrequestassigner

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevconsts"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevnet"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevstate"
)

type HallRequestAssignerElevatorState struct {
	Behavior    string `json:"behaviour"`
	Floor       int    `json:"floor"`
	Direction   string `json:"direction"`
	CabRequests []bool `json:"cabRequests"`
}

type HallRequestAssignerInput struct {
	HallRequests [][2]bool                                   `json:"hallRequests"`
	States       map[string]HallRequestAssignerElevatorState `json:"states"`
}

func getHallRequestAssignerElevatorState(elevatorState *elevstate.ElevatorState) HallRequestAssignerElevatorState {
	var cabRequests [elevconsts.N_FLOORS]bool
	for floor := 0; floor < elevconsts.N_FLOORS; floor++ {
		if elevatorState.Requests[floor][elevconsts.Cab] != 0 {
			cabRequests[floor] = true
		}
	}
	return HallRequestAssignerElevatorState{
		Behavior:    elevatorState.Behaviour.String(),
		Floor:       elevatorState.Floor,
		Direction:   elevatorState.Dirn.String(),
		CabRequests: cabRequests[:],
	}
}

// TODO: check if a list of elevators is correct input
func getHallRequestAssignerInput(localElevatorState elevstate.ElevatorState, elevatorMessageMap map[string]elevnet.ElevatorMessage) HallRequestAssignerInput {

	hallRequests := [elevconsts.N_FLOORS][2]bool{}
	for floor := 0; floor < elevconsts.N_FLOORS; floor++ {
		hallRequests[floor][elevconsts.HallUp] = localElevatorState.Requests[floor][elevconsts.HallUp] != 0
		hallRequests[floor][elevconsts.HallDown] = localElevatorState.Requests[floor][elevconsts.HallDown] != 0
	}

	states := make(map[string]HallRequestAssignerElevatorState)
	for id, elevatorMsg := range elevatorMessageMap {
		states[id] = getHallRequestAssignerElevatorState(&elevatorMsg.ElevatorState)
	}
	return HallRequestAssignerInput{
		HallRequests: hallRequests[:],
		States:       states,
	}
}

func getExecutableVersion() string {
	executableVersion := ""
	switch runtime.GOOS {
	case "linux":
		executableVersion = "hall_request_assigner"
	case "windows":
		executableVersion = "hall_request_assigner.exe"
	case "darwin":
		executableVersion = "macOS_hall_request_assigner"
	default:
		panic("OS not supported")
	}
	return executableVersion
}

func getOptimalHallRequests(executableVersion string, input HallRequestAssignerInput) map[string][][2]bool {
	jsonBytes, err := json.Marshal(input)
	if err != nil {
		fmt.Println("json.Marshal error: ", err)
		return nil
	}

	ret, err := exec.Command("..../libs/hall_request_assigner/"+executableVersion, "-i", string(jsonBytes)).CombinedOutput()
	if err != nil {
		fmt.Println("exec.Command error: ", err)
		fmt.Println(string(ret))
		return nil
	}

	output := new(map[string][][2]bool)
	err = json.Unmarshal(ret, &output)
	if err != nil {
		fmt.Println("json.Unmarshal error: ", err)
		return nil
	}
	return (*output)
}

func ReassignAllHallRequests(listener *elevnet.ElevNetListen) {
	elevatorMessageMap := listener.GetElevatorMessageMap()
	localElevatorState := listener.ElevatorState

	executableVersion := getExecutableVersion()
	input := getHallRequestAssignerInput(*localElevatorState, elevatorMessageMap)
	optimalHallRequests := getOptimalHallRequests(executableVersion, input)

	// Update local elevator state (might move to separate function)
	for floor := 0; floor < elevconsts.N_FLOORS; floor++ {
		localElevatorState.Requests[floor][elevconsts.HallUp] = toInt(optimalHallRequests[listener.ElevMetaData.Identifier][floor][elevconsts.HallUp])
		localElevatorState.Requests[floor][elevconsts.HallDown] = toInt(optimalHallRequests[listener.ElevMetaData.Identifier][floor][elevconsts.HallDown])
	}

	// Find best way to send new hall requests to other elevators
	for id, elevator := range elevatorMessageMap {
		for floor := 0; floor < elevconsts.N_FLOORS; floor++ {
			//something to create a maeesage to be sent to the other elevators
		}
	}
}

func toInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
