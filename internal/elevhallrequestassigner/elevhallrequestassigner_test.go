package elevhallrequestassigner

import (
	"reflect"
	"testing"

	//"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevator"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevconsts"
	//"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevmetadata"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevstate"
)

func TestGetHallRequestAssignerElevatorInput(t *testing.T) {
	var requests [elevconsts.N_FLOORS][elevconsts.N_BUTTONS]int

	requests[0][elevconsts.Cab] = 1                     // floor 2 has a cab request
	requests[elevconsts.N_FLOORS-1][elevconsts.Cab] = 1 // floor 4 has a cab request

	elevatorState := &elevstate.ElevatorState{
		Floor:     1,
		Dirn:      elevconsts.Up, // using a directional constant
		Requests:  requests,
		Behaviour: elevconsts.Moving, // using an example behaviour constant
	}

	result := getHallRequestAssignerElevatorState(elevatorState)

	expectedBehavior := elevatorState.Behaviour.String()
	expectedFloor := elevatorState.Floor
	expectedDirection := elevatorState.Dirn.String()

	expectedCabRequests := make([]bool, elevconsts.N_FLOORS)
	expectedCabRequests[0] = true
	expectedCabRequests[elevconsts.N_FLOORS-1] = true

	if result.Behavior != expectedBehavior {
		t.Errorf("Expected Behavior %q, got %q", expectedBehavior, result.Behavior)
	}
	if result.Floor != expectedFloor {
		t.Errorf("Expected Floor %d, got %d", expectedFloor, result.Floor)
	}
	if result.Direction != expectedDirection {
		t.Errorf("Expected Direction %q, got %q", expectedDirection, result.Direction)
	}
	if !reflect.DeepEqual(result.CabRequests, expectedCabRequests) {
		t.Errorf("Expected CabRequests %v, got %v", expectedCabRequests, result.CabRequests)
	}
}

// // TODO: test is dependant in the executable, bad practice for unit tests?
// func TestGetHallRequestAssignerInput(t *testing.T) {
// 	dummyStateOne := &elevstate.ElevatorState{
// 		Floor:     2,
// 		Requests:  [elevconsts.N_FLOORS][elevconsts.N_BUTTONS]int{{0, 0, 0}, {1, 0, 0}, {0, 0, 1}, {0, 1, 1}},
// 		Behaviour: elevconsts.Moving,
// 		Dirn:      elevconsts.Up,
// 	}
// 	dummyStateTwo := &elevstate.ElevatorState{
// 		Floor:     0,
// 		Requests:  [elevconsts.N_FLOORS][elevconsts.N_BUTTONS]int{{0, 0, 0}, {1, 0, 0}, {0, 0, 0}, {0, 1, 0}},
// 		Behaviour: elevconsts.Idle,
// 		Dirn:      elevconsts.Stop,
// 	}

// 	elevOne := elevator.Elevator{
// 		MetaData: &elevmetadata.ElevMetaData{Identifier: "one"},
// 		State:    dummyStateOne,
// 	}
// 	elevTwo := elevator.Elevator{
// 		MetaData: &elevmetadata.ElevMetaData{Identifier: "two"},
// 		State:    dummyStateTwo,
// 	}

// 	elevatorList := []elevator.Elevator{elevOne, elevTwo}
// 	ReassignAllHallRequests(&elevatorList)

// 	expectedOne := [][2]int{
// 		{0, 0},
// 		{0, 0},
// 		{0, 0},
// 		{0, 1},
// 	}
// 	expectedTwo := [][2]int{
// 		{0, 0},
// 		{1, 0},
// 		{0, 0},
// 		{0, 0},
// 	}

// 	for floor := 0; floor < elevconsts.N_FLOORS; floor++ {
// 		actualOneUp := elevatorList[0].State.Requests[floor][elevconsts.HallUp]
// 		actualOneDown := elevatorList[0].State.Requests[floor][elevconsts.HallDown]
// 		if actualOneUp != expectedOne[floor][0] || actualOneDown != expectedOne[floor][1] {
// 			t.Errorf("Elevator 'one', floor %d: expected [%d, %d], got [%d, %d]",
// 				floor, expectedOne[floor][0], expectedOne[floor][1], actualOneUp, actualOneDown)
// 		}

// 		actualTwoUp := elevatorList[1].State.Requests[floor][elevconsts.HallUp]
// 		actualTwoDown := elevatorList[1].State.Requests[floor][elevconsts.HallDown]
// 		if actualTwoUp != expectedTwo[floor][0] || actualTwoDown != expectedTwo[floor][1] {
// 			t.Errorf("Elevator 'two', floor %d: expected [%d, %d], got [%d, %d]",
// 				floor, expectedTwo[floor][0], expectedTwo[floor][1], actualTwoUp, actualTwoDown)
// 		}
// 	}

// }
