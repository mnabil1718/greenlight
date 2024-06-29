package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/mnabil1718/greenlight/internal/data"
	"github.com/mnabil1718/greenlight/internal/jsonlog"
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
	limiter struct {
		rps     float64
		burst   int
		enabled bool
	}
}

type application struct {
	logger *jsonlog.Logger
	config config
	models data.Models
}

func main() {
	var cfg config

	flag.StringVar(&cfg.host, "host", "localhost", "Application Server Hostname")
	flag.IntVar(&cfg.port, "port", 8080, "Application Server Port Number")
	flag.StringVar(&cfg.db.dsn, "dsn", os.Getenv("GREENLIGHT_DB_DSN"), "PostgreSQL Connection String")
	flag.StringVar(&cfg.env, "env", "dev", "Application Environment: (dev|staging|prod)")
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", "15m", "PostgreSQL max connection idle time")
	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter maximum burst")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enable rate limiter")
	flag.Parse()

	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)

	db, err := openDB(cfg)
	if err != nil {
		logger.PrintFatal(err, nil)
	}
	defer db.Close()

	logger.PrintInfo("database connection pool established successfully.", nil)

	app := application{
		logger: logger,
		config: cfg,
		models: data.NewModels(db),
	}

	server := http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.host, cfg.port),
		Handler:      app.routes(),
		ErrorLog:     log.New(logger, "", 0),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	logger.PrintInfo(fmt.Sprintf("starting %s server on %s", cfg.env, server.Addr), map[string]string{
		"addr": server.Addr,
		"env":  cfg.env,
	})
	err = server.ListenAndServe()
	if err != nil {
		logger.PrintFatal(err, nil)
	}
}
