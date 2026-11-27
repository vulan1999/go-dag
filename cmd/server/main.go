package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"github.com/vulan1999/go-dag/internal/pipeline"
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

	if err != nil || serverCancleTimeout <= 0 {
		log.Fatalf("Invalid server cancle timeout second, the value of SERVER_CANCLE_TIMEOUT variable must be a positive integer and greater than 0")
	}

	dbPath := server.GetEnvValue("DB_PATH", "flow.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to open SQLite database: %v", err)
	}
	defer db.Close()

	pipelineRepo, err := pipeline.NewSQLiteRepository(db)
	if err != nil {
		log.Fatalf("Failed to initialize pipeline repository: %v", err)
	}

	server := server.New(":"+serverPort, pipelineRepo)

	go func() {
		log.Printf("Listening on port :%s", serverPort)
		if err := server.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
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
	} else {
		log.Println("Shutdown complete")
	}
}
