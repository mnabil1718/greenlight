package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

const version = "1.0.0"

type config struct {
	port int
	env  string
	host string
}

type application struct {
	logger *log.Logger
	config config
}

func main() {
	var cfg config

	flag.StringVar(&cfg.host, "host", "localhost", "Application Server Hostname")
	flag.IntVar(&cfg.port, "port", 8080, "Application Server Port Number")
	flag.StringVar(&cfg.env, "env", "dev", "Application Environment: (dev|staging|prod)")
	flag.Parse()
	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)
	app := application{
		logger: logger,
		config: cfg,
	}

	server := http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.host, cfg.port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	err := server.ListenAndServe()
	if err != nil {
		logger.Fatal(err)
	}
}
