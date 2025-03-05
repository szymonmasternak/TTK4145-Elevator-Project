package elevutils

import (
	_ "embed"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
)

//go:generate sh -c "printf %s $(git rev-parse HEAD) > githash.txt"
//go:embed githash.txt
var gitHash string

func GetGitHash() string {
	return gitHash
}

func ProcessCmdArgs() (string, uint16, bool) {
	help := flag.Bool("help", false, "Show Help Window")
	version := flag.Bool("version", false, "Show Version")
	identifier := flag.String("id", "", "Set the identifier of the elevator. Defaults to random string")
	portNumber := flag.Uint64("port", 9999, "Set the port number of the elevator. Defaults to 9999")
	clearUpDownOnArrival := flag.Bool("clearupdownonarrival", false, "Clear the Up and Down requests at floor arrival. Defaults to false")

	flag.Parse()

	if *portNumber > 65535 || *portNumber < 1 {
		fmt.Println("Port number must be between 1 and 65535")
		os.Exit(1)
	}

	if *version {
		fmt.Println("Version:", GetGitHash())
		os.Exit(0)
	}

	if *help {
		fmt.Println("Usage: ./elevator [OPTIONS]")
		fmt.Println("TTK4145 Elevator Project")
		fmt.Println()
		fmt.Println("Options:")
		flag.PrintDefaults()
		fmt.Println()
		fmt.Println("Authors:")
		fmt.Println("	Szymon Masternak")
		fmt.Println("	Denisa Petraru")
		fmt.Println("	Kristina Nordang")
		fmt.Println()
		fmt.Println("License:")
		fmt.Println("	Copyright (c) 2025 All Rights Reserved")
		os.Exit(0)
	}

	return *identifier, uint16(*portNumber), *clearUpDownOnArrival
}

var localIP string //local string, not to be accessed anywhere

func GetLocalIP() string {
	if localIP == "" {
		conn, err := net.DialTCP("tcp4", nil, &net.TCPAddr{IP: []byte{8, 8, 8, 8}, Port: 53})
		if err != nil {
			panic(err)
		}
		defer conn.Close()
		localIP = strings.Split(conn.LocalAddr().String(), ":")[0]
	}
	return localIP
}
