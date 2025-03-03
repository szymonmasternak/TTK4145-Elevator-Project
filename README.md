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
- `internal/` directory contains different the different modules such as
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

As of currently there are

![PlantUML Diagram](https://www.plantuml.com/plantuml/svg/SoWkIImgAStDuNBAJrBGjLDmpCbCJbMmKiX8pSd9vt98pKi1IW80)

![DrawIO Diagram](/docs/diagrams/diagram.drawio.svg)