package elevhallrequestassigner

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevconsts"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevevent"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevnet"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevstate"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/logger"
)

var Log = logger.GetLogger()

type HallRequestAssigner struct {
	localID              string
	localNetworkListener *elevnet.ElevNetListen
	executableVersion    string

	eventChannel              chan<- elevevent.ElevatorEvent
	localStateRecieverChannel <-chan elevstate.ElevatorState
}

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

func NewHallRequestAssigner(localID string, localNetworkListener *elevnet.ElevNetListen, eventChannel chan<- elevevent.ElevatorEvent, localStateRecieverChannel <-chan elevstate.ElevatorState) *HallRequestAssigner {
	return &HallRequestAssigner{
		localID:                   localID,
		localNetworkListener:      localNetworkListener,
		executableVersion:         getExecutableVersion(),
		eventChannel:              eventChannel,
		localStateRecieverChannel: localStateRecieverChannel,
	}
}

func (assigner *HallRequestAssigner) Start(ctx context.Context, waitGroup *sync.WaitGroup) {
	Log.Debug().Msgf("HallRequestAssigner started")
	assignerTicker := time.NewTicker(100 * time.Millisecond)

	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()
		defer assignerTicker.Stop()
		Log.Debug().Msgf("HallRequestAssigner goroutine started")
		for {
			select {
			case <-ctx.Done():
				Log.Warn().Msgf("Assigner Start has been signaled to stop")
				return
			case <-assignerTicker.C:
				localState := <-assigner.localStateRecieverChannel
				Log.Debug().Msgf("HallRequestAssigner got local state")
				stateMap := assigner.localNetworkListener.GetElevatorStateMap()
				input := getHallRequestAssignerInput(localState, stateMap)
				if len(input.States) != 0 {
					optimalHallRequests := getOptimalHallRequests(assigner.executableVersion, input)
					Log.Debug().Msgf("HallRequestAssigner got optimal hall requests")
					optimalLocalRequests := getOptimalLocalRequests(optimalHallRequests, localState.ConfirmedRequests, assigner.localID)
					assigner.eventChannel <- elevevent.ElevatorEvent{Value: elevevent.UpdateHallRequestsEvent{Requests: optimalLocalRequests}}
				}

				// if localState.Requests != optimalLocalRequests {
				// 	assigner.eventChannel <- elevevent.ElevatorEvent{Value: elevevent.UpdateHallRequestsEvent{Requests: optimalLocalRequests}}
				// }
			}
		}
	}()
}

func getHallRequestAssignerElevatorState(elevatorState *elevstate.ElevatorState) HallRequestAssignerElevatorState {
	var cabRequests [elevconsts.N_FLOORS]bool
	for floor := 0; floor < elevconsts.N_FLOORS; floor++ {
		if elevatorState.ConfirmedRequests[floor][elevconsts.Cab] != 0 {
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

// TODO: check if localelevator is in map
func getHallRequestAssignerInput(localElevatorState elevstate.ElevatorState, elevatorStateMap map[string]elevstate.ElevatorState) HallRequestAssignerInput {

	hallRequests := [elevconsts.N_FLOORS][2]bool{}
	for floor := 0; floor < elevconsts.N_FLOORS; floor++ {
		hallRequests[floor][elevconsts.HallUp] = localElevatorState.ConfirmedRequests[floor][elevconsts.HallUp] != 0
		hallRequests[floor][elevconsts.HallDown] = localElevatorState.ConfirmedRequests[floor][elevconsts.HallDown] != 0
	}

	states := make(map[string]HallRequestAssignerElevatorState)
	for id, state := range elevatorStateMap {
		states[id] = getHallRequestAssignerElevatorState(&state)
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

	ret, err := exec.Command("../libs/hall_request_assigner/"+executableVersion, "-i", string(jsonBytes)).CombinedOutput()
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

func getOptimalLocalRequests(optimalHallRequests map[string][][2]bool, originalLocalRequests [elevconsts.N_FLOORS][elevconsts.N_BUTTONS]int, localID string) [elevconsts.N_FLOORS][elevconsts.N_BUTTONS]int {
	localRequests := [elevconsts.N_FLOORS][elevconsts.N_BUTTONS]int{}
	for floor := 0; floor < elevconsts.N_FLOORS; floor++ {
		localRequests[floor][elevconsts.HallUp] = toInt(optimalHallRequests[localID][floor][elevconsts.HallUp])
		localRequests[floor][elevconsts.HallDown] = toInt(optimalHallRequests[localID][floor][elevconsts.HallDown])
		localRequests[floor][elevconsts.Cab] = originalLocalRequests[floor][elevconsts.Cab]
	}
	return localRequests
}

func toInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
