// +build ignore

package main

import (
	"fmt"
	"github.com/al-hudhud/backend/internal/agent"
)

func main() {
	// Test that the package compiles correctly
	fmt.Println("Agent package imports successfully")
	
	// Test JWT key constant
	fmt.Println("JWT Key:", agent.JWTKey)
	
	fmt.Println("All basic imports working!")
}