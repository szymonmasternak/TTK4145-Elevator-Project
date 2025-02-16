module github.com/szymonmasternak/TTK4145-Elevator-Project

go 1.23.4

require github.com/rs/zerolog v1.33.0

require (
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	golang.org/x/sys v0.12.0 // indirect
)

require (
	github.com/szymonmasternak/TTK4145-Elevator-Project/libs/Network-go v0.0.0
	github.com/xyproto/randomstring v1.2.0
)

replace github.com/szymonmasternak/TTK4145-Elevator-Project/libs/Network-go => ./libs/Network-go
