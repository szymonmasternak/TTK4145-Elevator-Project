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
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/requestconfirmation"
)

var Log = logger.GetLogger()

type HallRequestAssigner struct {
	localID              string
	localNetworkListener *elevnet.ElevNetListen
	executableVersion    string

	eventChannel              chan<- elevevent.ElevatorEvent
	localStateRecieverChannel <-chan elevstate.ElevatorState
	localRequestsArrChannel   <-chan requestconfirmation.RequestArray
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

func NewHallRequestAssigner(localID string, localNetworkListener *elevnet.ElevNetListen, eventChannel chan<- elevevent.ElevatorEvent, localStateRecieverChannel <-chan elevstate.ElevatorState, localReqsArrCh <-chan requestconfirmation.RequestArray) *HallRequestAssigner {
	return &HallRequestAssigner{
		localID:                   localID,
		localNetworkListener:      localNetworkListener,
		executableVersion:         getExecutableVersion(),
		eventChannel:              eventChannel,
		localStateRecieverChannel: localStateRecieverChannel,
		localRequestsArrChannel:   localReqsArrCh,
	}
}

func (assigner *HallRequestAssigner) Start(ctx context.Context, waitGroup *sync.WaitGroup) {
	assignerTicker := time.NewTicker(100 * time.Millisecond)

	var latestState elevstate.ElevatorState
	var latestReqArray requestconfirmation.RequestArray

	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()
		defer assignerTicker.Stop()
		for {
			select {
			case <-ctx.Done():
				Log.Warn().Msgf("Assigner Start has been signaled to stop")
				return
			case <-assignerTicker.C:
				latestState = drainState(assigner.localStateRecieverChannel, latestState)
				latestReqArray = drainRequests(assigner.localRequestsArrChannel, latestReqArray)
				// Now process using latestState and latestReqArray
				Log.Debug().Msgf("HRA: latestReqArray: %v", latestReqArray)
				stateMap := assigner.localNetworkListener.GetElevatorStateMap()
				updatedLocalRequests := [elevconsts.N_FLOORS][elevconsts.N_BUTTONS]int{}
				for floor := 0; floor < elevconsts.N_FLOORS; floor++ {
					for btn := 0; btn < elevconsts.N_BUTTONS; btn++ {
						if latestReqArray[floor][btn].State == requestconfirmation.REQ_Confirmed {
							updatedLocalRequests[floor][btn] = 1
						}
					}
				}
				Log.Debug().Msgf("New local confirmedRequests: %v", updatedLocalRequests)
				latestState.ConfirmedRequests = updatedLocalRequests
				stateMap[assigner.localID] = latestState
				if len(stateMap) > 1 {
					input := getHallRequestAssignerInput(latestState, stateMap)
					optimalHallRequests := getOptimalHallRequests(assigner.executableVersion, input)
					Log.Debug().Msgf("HRA: optimal local hall requests: %v", optimalHallRequests)
					if optimalHallRequests == nil {
						Log.Warn().Msgf("HallRequestAssigner got nil optimal hall requests")
						continue
					}
					optimalLocalRequests := getOptimalLocalRequests(optimalHallRequests, latestState.ConfirmedRequests, assigner.localID)
					assigner.eventChannel <- elevevent.ElevatorEvent{
						Value: elevevent.UpdateHallRequestsEvent{Requests: optimalLocalRequests},
					}
				} else {
					assigner.eventChannel <- elevevent.ElevatorEvent{
						Value: elevevent.UpdateHallRequestsEvent{Requests: updatedLocalRequests},
					}
				}
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
	var executableName string
	switch runtime.GOOS {
	case "linux":
		executableName = "hall_request_assigner"
	case "windows":
		executableName = "hall_request_assigner.exe"
	case "darwin":
		executableName = "macOS_hall_request_assigner"
	default:
		panic("OS not supported")
	}
	return executableName

	// // If the executables are all in the same "hall_request_assigner" folder:
	// path := filepath.Join("libs", "hall_request_assigner", executableName)
	// absPath, err := filepath.Abs(path)
	// if err != nil {
	// 	panic("Unable to determine absolute path: " + err.Error())
	// }

	// return absPath
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

func drainState(ch <-chan elevstate.ElevatorState, current elevstate.ElevatorState) elevstate.ElevatorState {
	latest := current
	for {
		select {
		case s := <-ch:
			latest = s
		default:
			return latest
		}
	}
}

func drainRequests(ch <-chan requestconfirmation.RequestArray, current requestconfirmation.RequestArray) requestconfirmation.RequestArray {
	latest := current
	for {
		select {
		case r := <-ch:
			latest = r
		default:
			return latest
		}
	}
}
