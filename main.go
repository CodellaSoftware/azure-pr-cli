package main

import (
	"log"

	"github.com/CodellaSoftware/azure-pr-cli/cmd"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	cmd.Execute()
}
