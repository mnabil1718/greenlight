package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func (app *application) serve() error {
	server := http.Server{
		Addr:         fmt.Sprintf("%s:%d", app.config.host, app.config.port),
		Handler:      app.routes(),
		ErrorLog:     log.New(app.logger, "", 0),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	shutDownErr := make(chan error)

	// runs in the background waiting for syscall
	go func() {
		quit := make(chan os.Signal, 1) //  buffered channel, 1 empty slot ready to receive 1 data (signal)

		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		signalResult := <-quit
		app.logger.PrintInfo("shutting down server", map[string]string{
			"signal": signalResult.String(),
		})

		// give any in-flight requests a ‘grace period’ of 20 seconds
		// to complete before the application is terminated
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		shutDownErr <- server.Shutdown(ctx)
	}()

	app.logger.PrintInfo(fmt.Sprintf("starting %s server on %s", app.config.env, server.Addr), map[string]string{
		"addr": server.Addr,
		"env":  app.config.env,
	})

	err := server.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	err = <-shutDownErr
	if err != nil {
		return err
	}

	app.logger.PrintInfo("stopped server", map[string]string{
		"addr": server.Addr,
	})

	return nil

}
