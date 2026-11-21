package main

import (
	"context"
	"log"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/vulan1999/go-dag/internal/server"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Cannot find .env file, reading system enviroment variables")
	} else {
		log.Println(".env file found, loading enviroment variables")
	}
}

func main() {
	serverPort := server.GetEnvValue("PORT", "8080")
	serverCancleTimeout, err := strconv.Atoi(server.GetEnvValue("SERVER_CANCLE_TIMEOUT", "20"))

	if err != nil {
		log.Fatalf("Invalid server cancle timeout second, please set back the SERVER_CANCLE_TIMEOUT variable")
	}

	server := server.New(serverPort)

	go func() {
		log.Println("Listening on port :8080")
		if err := server.Start(); err != nil {
			log.Printf("Fail to start server: %v", err)
		}
	}()

	shutdown, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	<-shutdown.Done()

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(serverCancleTimeout)*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Shutdown with error: %v", err)
	}
	log.Println("Shutdown complete")
}
