package elevutils

import (
	_ "embed"
	"flag"
	"fmt"
	"os"
)

//go:generate sh -c "printf %s $(git rev-parse HEAD) > githash.txt"
//go:embed githash.txt
var gitHash string

func GetGitHash() string {
	return gitHash
}

func ProcessCmdArgs() {
	help := flag.Bool("help", false, "Show Help Window")
	version := flag.Bool("version", false, "Show Version")

	flag.Parse()

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
}
