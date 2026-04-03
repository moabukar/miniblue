package main

import (
	"log"
	"os"

	"github.com/moabukar/local-azure/internal/server"
)

func main() {
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	
	srv := server.New()
	if err := srv.Run(); err != nil {
		log.Fatal(err)
	}
}
