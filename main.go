package main

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/yourusername/azure-pr-cli/cmd"
)

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	cmd.Execute()
}
