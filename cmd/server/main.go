package main

import (
	"log"

	"github.com/vulan1999/go-dag/internal/server"
)

func main() {
	server := server.New(":8080")
	if err := server.Start(); err != nil {
		log.Fatalf("Fail to start server: %v", err)
	}
}
