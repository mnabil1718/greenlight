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
	db   struct {
		dsn          string
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  string
	}
}

type application struct {
	logger *log.Logger
	config config
}

func main() {
	var cfg config

	fmt.Println("env var", os.Getenv("GREENLIGHT_DB_DSN"))

	flag.StringVar(&cfg.host, "host", "localhost", "Application Server Hostname")
	flag.IntVar(&cfg.port, "port", 8080, "Application Server Port Number")
	flag.StringVar(&cfg.db.dsn, "dsn", os.Getenv("GREENLIGHT_DB_DSN"), "PostgreSQL Connection String")
	flag.StringVar(&cfg.env, "env", "dev", "Application Environment: (dev|staging|prod)")
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", "15m", "PostgreSQL max connection idle time")
	flag.Parse()
	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	db, err := openDB(cfg)
	if err != nil {
		logger.Fatal(err)
	}
	defer db.Close()

	logger.Printf("database connection pool established successfully.")

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

	logger.Printf("starting %s server on %s", cfg.env, server.Addr)
	err = server.ListenAndServe()
	if err != nil {
		logger.Fatal(err)
	}
}
