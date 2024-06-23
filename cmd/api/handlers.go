package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/mnabil1718/greenlight/internal/data"
)

func (app *application) healthcheckHandler(writer http.ResponseWriter, request *http.Request) {
	env := envelope{
		"status": "available",
		"system_info": map[string]string{
			"environment": app.config.env,
			"version":     version,
		},
	}

	err := app.writeJSON(writer, http.StatusOK, env, request.Header)

	if err != nil {
		app.serverErrorResponse(writer, request, err)
	}
}

func (app *application) createMovieHandler(writer http.ResponseWriter, request *http.Request) {
	fmt.Fprintln(writer, "create a new movie")
}

func (app *application) showMovieHandler(writer http.ResponseWriter, request *http.Request) {
	id, err := app.getIdFromRequestContext(request)
	if err != nil || id < 1 {
		app.notFoundResponse(writer, request)
		return
	}

	movie := data.Movie{
		ID:        id,
		CreatedAt: time.Now(),
		Title:     "Casablanca",
		Runtime:   102,
		Genres:    []string{"drama", "romance", "war"},
		Version:   1,
	}

	err = app.writeJSON(writer, http.StatusOK, envelope{"movie": movie}, request.Header)

	if err != nil {
		app.serverErrorResponse(writer, request, err)
	}
}
