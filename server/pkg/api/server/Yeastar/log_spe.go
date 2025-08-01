package Yeastar

import (
	"fmt"
	"strings"
)

// ANSI color codes
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
)

// Log prints a message with the appropriate color based on log level
func Log(level string, message string) {
	var color string

	switch strings.ToLower(level) {
	case "info":
		color = ColorGreen
	case "warn", "warning":
		color = ColorYellow
	case "error", "err":
		color = ColorRed
	case "debug":
		color = ColorBlue
	default:
		color = ColorReset
	}

	fmt.Println(color + "[" + strings.ToUpper(level) + "] " + message + ColorReset)
}

// func main() {
// 	Log("info", "Server started successfully")
// 	Log("warn", "Disk space is getting low")
// 	Log("error", "Failed to connect to database")
// 	Log("debug", "Received heartbeat packet")
// }
