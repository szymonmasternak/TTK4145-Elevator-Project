# TTK4145-Elevator-Project
Group project for the NTNU TTK4145 Real-time Programming Module

## How to Run the Project

To build and run the project one can run the following command in their command prompt

```terminal
make
```

Alternatively, the use of `make build` and `make run` will work also.

Running the program with `make all` will build, test and run the program.

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

# Project Structure

The structure of this project is as follows:

- `.github/workflows/*.yaml` contains CI/CD pipeline for the project 
- `Makefile` encompasses the build process
- `build/` directory encompasses binaries built during the build process
- `cmd/` houses the `main.go` files for different binaries.
- `internal/` directory contains different modules such as;
    - `elevator`: contains the root struct of the elevator and associated functions
    - `elevcmd`: Defines the elevator `command` type
    - `elevconsts`: Holds constant definitions used throughout the project
    - `elevevent`: Defines `event` type
    - `elevio`: Handles input/output interfaces with actual elevator (hardware I/O via `eleviodriver.go`).
    - `elevmetadata`: Contains metadata struct information for the elevator.
    - `elevnet`: Contains network functionality (broadcast, listen, etc.) with tests.
    - `elevstate`: Contains the elevator's state and FSM as well as corresponding tests.
    - `elevutils`: Provides functions that could not be placed elsewhere.
    - `logger`: Contains logging functionality and tests.

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