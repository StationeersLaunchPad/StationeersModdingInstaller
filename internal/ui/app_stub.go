//go:build !windows

package ui

import "fmt"

// Run is a non-Windows stub so the project can still compile in Linux/macOS dev environments.
func Run() {
	fmt.Println("Stationeers Modding Installer GUI is currently Windows-only.")
}
