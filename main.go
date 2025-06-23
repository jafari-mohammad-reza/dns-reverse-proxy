package main

import (
	"log"
	"os"
)

func main() {
	conf, err := InitConf()
	if err != nil {
		log.Fatalf("[-] Failed to load configuration: %v", err)
		os.Exit(1)
	}
	logger := NewLogger(conf)
	server := NewServer(conf, logger)
	err = server.Start()
	if err != nil {
		log.Fatal("Failed to start server:", err)
		os.Exit(1)
	}
	defer func() {
		err := server.Stop()
		if err != nil {
			log.Fatal("Failed to stop server:", err)
			os.Exit(1)
		}
	}()
}
