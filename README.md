# TTK4145-Elevator-Project

Group project for the NTNU TTK4145 Real-time Programming Module

# Project Structure

The structure of this project is as follows:

- `.github/workflows/*.yaml` contains CI/CD pipeline for the project
- `Makefile` encompasses the build process
- `build/` directory encompasses binaries built during the build process and the latest elevator state as a json file for retrieval in case of software crashes etc..
- `cmd/` houses the `main.go` file for running the elevator
- `internal/` directory contains different modules such as;
  - `elevator`: contains the root struct of the elevator and associated functions
  - `elevcmd`: Defines the elevator `command` type
  - `elevconsts`: Holds constant definitions used throughout the project
  - `elevevent`: Defines `event` type
  - `elevio`: Handles input/output interfaces with actual elevator (hardware I/O via `eleviodriver.go`).
  - `elevmetadata`: Contains metadata struct information for the elevator.
  - `elevnet`: Contains network functionality (broadcast, listen, etc.) with tests.
  - `elevstate`: Contains the elevator's state and FSM as well as corresponding tests.
  - `elevstatenetmsg` : Defines the message type sent between the network and state machine
  - `elevutils`: Provides functions that could not be placed elsewhere.
  - `logger`: Contains logging functionality and tests.

## State machine

The elevator state machine encapsulates the core behavior of an individual elevator. It maintains the current floor, direction, active requests (both cab and hall calls), and overall operational behavior (idle, moving, or door open). The state machine is designed to handle events from both local inputs (button presses, floor sensor readings, stop/obstruction signals) and network messages, coordinating these to determine the next actions for the elevator.

Key features include:

* **Event-Driven Architecture:**
  Processes local and network events (button presses, floor sensor events, stop and obstruction signals).
* **Initialization and Floor Detection:**
  Determines the initial floor (or defaults to moving down if between floors) using sensor events.
* **State Transitions:**
  Manages transitions between Idle, Moving, and DoorOpen states based on incoming events.
* **Request Handling:**
  Updates a 2D requests array for cab and hall calls and uses cost estimation to determine service order.
* **Command Generation:**
  Issues hardware commands (motor direction, door open/close, floor indicators, button lights) through a dedicated command channel.
* **Door Timeout Handling:**
  Monitors the door open duration and triggers door closure if the door remains open beyond the configured timeout.
* **Integration with Network Module:**
  Supports network events (e.g., NetworkButtonEvent) to synchronize shared hall calls with other elevators.
* **Cost Calculation:**
  Simulates movement to estimate the time to serve a request, helping determine the optimal elevator to delegate a hall call.

For more details, see [elevstate.go](./internal/elevstate/elevstate.go).

## Network functionality

The `elevnet` package implements a fully distributed peer-to-peer elevator network with UDP based message transmission. It's main features are as follows:

- **Heartbeat Mechanism:** Each elevator broadcasts periodic heartbeat messages over UDP multicast to announce its presence. Other nodes receive these heartbeats, update their records, and mark nodes as unresponsive if no heartbeat is received within a timeout period.
- **Direct Node-to-Node Communication:** In addition to multicast, a direct UDP socket is used to send state updates and request messages (such as hall call delegation and acknowledgment packets).
- **State Updates & Request Delegation:** Elevators send state update messages at a fixed interval. When a hall call is detected (via a button press), the network module uses a cost calculation—based on each node’s current state—to determine the best elevator for the call. If the best candidate is not the local elevator, a "do request" is sent to delegate the call.

* **Acknowledgment Mechanism:**
  * Each direct message is sent with a computed hash and the sender waits for an ACK (acknowledgment) containing the same hash.
  * The sender retries the message (up to a maximum of 3 attempts) if an ACK isn’t received within a 200ms timeout.
* **Reassignment of Requests:**
  * If a node does not acknowledge after the maximum retries, it is marked as unresponsive.
  * When a node is deemed offline, the system checks active nodes and, if the current node is the lowest (by identifier), it takes over by reassigning the unacknowledged hall call requests.

For more details, see [elevnet.go](./internal/elevnet/elevnet.go).

## Channels

There are a number of channels active.

### Internal channels between elevio and elevdriver

Data gets sent from the `elevdriver` to the `elevio` module.
The `elevdriver` polls all the buttons, floor sensor, obstruction switch as well as the stop switch and sends the events to the `elevio` module.

![Elevdriver and ElevIO driver channels](https://www.plantuml.com/plantuml/svg/VS_1JeH030RWUv-YH_UmBx07inensHC942_gOM2gJ3BJag6G-FOEJC34YCUstz-qrr5Dr2buyFIXg8BHVVQAraNgr0a3l9AdsOcDRgRuZcR4f-hsKbJRO6tTobocTKfhfsuUcW8WMpoVxvF12tQzOIR_EAaAl_6TVWrqmLmMCx6UZYBFNYJk2VUFh5M67ROY_b2MxQZn5mL88i7yGRAqdBKbThnyI_dW37zsImY6xP-9BjWJN8dj5Fmt)

### Channels between elevio and elevstate

The `elevIO` module takes different events and maps them to an `elevevent` structure which gets sent over to the `elevstate` module.
Inside the `elevstate` module, these events are processed accordingly according to the state diagrams.

If action is needed based on any events, then a command is sent through the command channel back to the `elevIO` module, encompassed in an `elevcmd` structure. This command structure is processed and an appropriate function executed on the `elevio` side. For example sending `DoorOpenCommand{}` command over the command channel will open the door.

![PlantUML Diagram](https://www.plantuml.com/plantuml/svg/PSw_JiCm40RmtKznwaI7la07r0YiAbAJICIm0CF9Fb8B_wdi2n92l3l1mIoL9_zydxyxPCR4itV2qHi3HqXsEZCOcqYpZK682-ftd0Wsqj47SamR-180pxHSTGoPyojWXhkX7rLr6ukrGmLFZ0OP2tTIDVKX41VhvNyuCp8L75MZPEMPhLkh7bLx6d_PnMcYLEmq78_oeSuk9t1n-IHxLLbxyxThrpNzlVWMeXrWjxjTcs0F9GZwZ26GUmat_7cXHUkNMr46IsH9xa57xp6OwyJQjXK72cRsxWS0)

In order to be able to process these commands and events, threads have to be run accordingly on both the `elevIO` as well as `elevstate` modules.

## How to Run the Project

To build and run the project one can run the following command in their command prompt

```terminal
make
```

Alternatively, the use of `make build` and `make run` will work also.

Running the program with `make all` will build, test and run the program.

To specify options such as identifier, IP address of the driver, port numbers for the network connections, etc. these can be specified when running `./elevator` from the build folder. The different options are as follows:

```terminal
Options:
  -clearupdownonarrival
        Clear the Up and Down requests at floor arrival. Defaults to false
  -driverip string
        Set the IP address of the driver. (default "localhost:15657")
  -help
        Show Help Window
  -id string
        Set the identifier of the elevator. Defaults to random string
  -port uint
        Set the port number that the elevator uses for direct communication. Should be unique for each elevator. (default 9999)
  -udpport uint
        Set the port number that the elevators broadcasts heartbeat messages on. Should be the same for all elevators. (default 53317)
  -version
        Show Version
```

This information can also be found running  `./elevator --help` .

## How to execute tests?

This project comes with some unit tests that test the software. These can be ran with

```terminal
make test
```

## How to cross compile for a different platform.

By default, golang will compile a binary for the host machine that you are running it off.

In order to change this, to another platform (ie. to run on linux machines in the laboratory), one has to set the following variables

```terminal
GOOS=linux GOARCH=amd64 make build
```

where `GOOS` is the target operating system and `GOARCH` is the target system architecture.

This will generate the corresponding binaries in the `build/` folder where they can be copied and ran from the target machine.

One can read more about this in the following [webpage](https://tip.golang.org/wiki/WindowsCrossCompiling#go-version--15) or this [webpage](https://gist.github.com/asukakenji/f15ba7e588ac42795f421b48b8aede63).
