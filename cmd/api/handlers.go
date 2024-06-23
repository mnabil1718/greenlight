package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/mnabil1718/greenlight/internal/data"
	"github.com/mnabil1718/greenlight/internal/validator"
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
	// user cannot post data straight to Movie model
	// it would be unsafe. Instead use this decoy
	var createMovieRequest struct {
		Title   string       `json:"title"`
		Year    int32        `json:"year"`
		Runtime data.Runtime `json:"runtime"`
		Genres  []string     `json:"genres"`
	}

	err := app.readJSON(writer, request, &createMovieRequest)
	if err != nil {
		app.badRequestResponse(writer, request, err)
		return
	}

	movie := &data.Movie{
		Title:   createMovieRequest.Title,
		Year:    createMovieRequest.Year,
		Runtime: createMovieRequest.Runtime,
		Genres:  createMovieRequest.Genres,
	}

	v := validator.New()

	if data.ValidateMovie(v, movie); !v.Valid() {
		app.failedValidationResponse(writer, request, v.Errors)
		return
	}

	fmt.Fprintf(writer, "%+v\n", createMovieRequest)
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
