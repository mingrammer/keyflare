package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	fmt.Println("KeyFlare Redis Examples")
	fmt.Println("=======================")
	fmt.Println("Choose an example to run:")
	fmt.Println("1. Local Cache Policy - Shows basic integration")
	fmt.Println("2. Key Splitting Policy - Shows shard-based mitigation")
	fmt.Println("3. Monitoring Demo - Shows metrics and API usage")
	fmt.Print("\nEnter your choice (1-3): ")

	reader := bufio.NewReader(os.Stdin)
	choice, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal("Failed to read input:", err)
	}

	choice = strings.TrimSpace(choice)

	var runSimulation bool
	if choice != "3" { // Monitoring demo always includes simulation
		fmt.Print("\nRun traffic simulation? (y/n): ")
		simChoice, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal("Failed to read input:", err)
		}
		runSimulation = strings.TrimSpace(strings.ToLower(simChoice)) == "y"
	}

	switch choice {
	case "1":
		LocalCachePolicyExample(runSimulation)
	case "2":
		KeySplittingPolicyExample(runSimulation)
	case "3":
		MonitoringExample()
	default:
		fmt.Println("Invalid choice. Please run again and select 1, 2, or 3.")
	}
}
