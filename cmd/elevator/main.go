package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

const (
	heartbeatInterval = 1 * time.Second // Interval between heartbeats sent by the primary
	heartbeatTimeout  = 3 * time.Second // Increased timeout duration to reduce false positives
	defaultPort       = 9999            // Starting UDP port for communication
	missThreshold     = 1               // Number of consecutive missed heartbeats before failover
)

var (
	elevators []string // List of all elevators
	mutex     sync.Mutex
)

// Function to add an elevator to the list safely
func addElevator(role string, port int) {
	mutex.Lock()
	defer mutex.Unlock()
	entry := fmt.Sprintf("%s (Port: %d)", role, port)
	elevators = append(elevators, entry)
	fmt.Println("Elevator Added:", entry) // Log when an elevator is added
}

// Periodically print the current list of elevators
func monitorElevators() {
	for {
		time.Sleep(5 * time.Second) // Print every 5 seconds
		mutex.Lock()
		fmt.Println("Current Elevators List:", elevators)
		mutex.Unlock()
	}
}

// HTTP API to retrieve the list of elevators
func handleElevatorList(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()
	json.NewEncoder(w).Encode(elevators)
}

// Start an HTTP server to expose the elevator list
func startServer() {
	http.HandleFunc("/elevators", handleElevatorList)
	http.ListenAndServe(":8080", nil)
}

func main() {
	go monitorElevators() // Start monitoring in the background
	go startServer()      // Start the HTTP server

	role := "primary" // Default role is primary
	count := 1        // Start counting from 1
	port := defaultPort
	backupPort := defaultPort + 1

	// Check if the current process is a backup with a specific port
	if len(os.Args) > 1 && os.Args[1] == "backup" && len(os.Args) > 2 {
		role = "backup"
		port, _ = strconv.Atoi(os.Args[2])
	}

	// Add this instance to the elevator list
	addElevator(role, port)

	// Based on role, start the appropriate function
	if role == "primary" {
		go spawnBackup(backupPort)  // Spawn the first backup process
		time.Sleep(2 * time.Second) // Ensure the backup has time to start listening
		primary(count, backupPort)  // Start the primary process, sending to the backup
	} else {
		backup(port) // Start the backup process
	}
}

// Function to spawn a new backup process with a unique port
func spawnBackup(port int) {
	// Ensure the program runs the source file directly
	sourcePath, err := filepath.Abs("counting.go")
	if err != nil {
		fmt.Println("Failed to get absolute path of source file:", err)
		return
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println("Failed to get current working directory:", err)
		return
	}

	// Add new backup to elevator list
	addElevator("backup", port)

	cmd := exec.Command("osascript", "-e", `tell app "Terminal" to do script "cd \"`+cwd+`\" && go run \"`+sourcePath+`\" backup `+strconv.Itoa(port)+`"`)
	err = cmd.Start()
	if err != nil {
		fmt.Println("Failed to start backup:", err)
	}
}

// Primary process: Counts numbers and sends heartbeats to the backup
func primary(startCount, backupPort int) {
	raddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: backupPort}
	conn, err := net.DialUDP("udp", nil, raddr) // Using DialUDP instead of Dial
	if err != nil {
		fmt.Println("Error setting up primary connection:", err)
		return
	}
	defer conn.Close()

	count := startCount
	for {
		message := strconv.Itoa(count)
		_, err := conn.Write([]byte(message)) // Send heartbeat with the current count
		if err != nil {
			fmt.Println("Error sending heartbeat:", err)
		} else {
			fmt.Println("Primary sending heartbeat to port", backupPort, ":", message)
		}
		count++
		time.Sleep(heartbeatInterval) // Wait for the heartbeat interval
	}
}

// Backup process: Listens for heartbeats and takes over if the primary fails
func backup(port int) {
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:"+strconv.Itoa(port)) // Bind to loopback address explicitly
	if err != nil {
		fmt.Println("Error resolving address:", err)
		return
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Println("Error setting up backup listener:", err)
		return
	}
	defer conn.Close()

	fmt.Println("Backup listening on port:", port)

	buffer := make([]byte, 1024)
	lastCount := 0        // Track the last received count
	missedHeartbeats := 0 // Track consecutive missed heartbeats

	for {
		conn.SetReadDeadline(time.Now().Add(heartbeatTimeout)) // Set a read timeout
		n, addr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println("Error reading from UDP:", err)
			missedHeartbeats++
			if missedHeartbeats >= missThreshold {
				nextPort := port + 1
				fmt.Println("Primary is down after", missedHeartbeats, "missed heartbeats. Taking over...")

				// Add new backup to elevator list
				addElevator("backup", nextPort)

				go spawnBackup(nextPort)       // Spawn a new backup with the next port
				time.Sleep(2 * time.Second)    // Ensure the new backup has time to start listening
				primary(lastCount+1, nextPort) // Become the new primary, send to the next backup
				return
			}
			continue
		}

		// Successfully received heartbeat
		missedHeartbeats = 0 // Reset missed heartbeats counter
		count, _ := strconv.Atoi(string(buffer[:n]))
		lastCount = count // Update the last received count

		fmt.Printf("Backup received heartbeat: %d from %s\n", count, addr) // Log the heartbeat received
	}
}
