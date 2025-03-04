package elevhallrequestassigner

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevator"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevconsts"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevstate"
)

// TODO: improve names

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

// TODO: pointer or not as input?
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
func getHallRequestAssignerInput(elevatorList []elevator.Elevator) HallRequestAssignerInput {
	hallRequests := [elevconsts.N_FLOORS][2]bool{}
	for floor := 0; floor < elevconsts.N_FLOORS; floor++ {
		hallRequests[floor][elevconsts.HallUp] = elevatorList[0].State.Requests[floor][elevconsts.HallUp] != 0
		hallRequests[floor][elevconsts.HallDown] = elevatorList[0].State.Requests[floor][elevconsts.HallDown] != 0
	}
	states := make(map[string]HallRequestAssignerElevatorState)
	for _, elevator := range elevatorList {
		states[elevator.MetaData.Identifier] = getHallRequestAssignerElevatorState(elevator.State)
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

func ReassignAllHallRequests(elevatorList *[]elevator.Elevator) {
	executableVersion := getExecutableVersion()
	input := getHallRequestAssignerInput(*elevatorList)
	optimalHallRequests := getOptimalHallRequests(executableVersion, input)
	for _, elevator := range *elevatorList {
		for floor := 0; floor < elevconsts.N_FLOORS; floor++ {
			elevator.State.Requests[floor][elevconsts.HallUp] = toInt(optimalHallRequests[elevator.MetaData.Identifier][floor][elevconsts.HallUp])
			elevator.State.Requests[floor][elevconsts.HallDown] = toInt(optimalHallRequests[elevator.MetaData.Identifier][floor][elevconsts.HallDown])
		}
	}
}

func toInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
