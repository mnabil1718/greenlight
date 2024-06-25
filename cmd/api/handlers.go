package main

import (
	"errors"
	"fmt"
	"net/http"

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

	err = app.models.Movies.Insert(movie)
	if err != nil {
		app.serverErrorResponse(writer, request, err)
		return
	}

	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/movies/%d", movie.ID))

	err = app.writeJSON(writer, http.StatusCreated, envelope{"movie": movie}, headers)
	if err != nil {
		app.serverErrorResponse(writer, request, err)
	}
}

func (app *application) showMovieHandler(writer http.ResponseWriter, request *http.Request) {
	id, err := app.getIdFromRequestContext(request)
	if err != nil || id < 1 {
		app.notFoundResponse(writer, request)
		return
	}

	movie, err := app.models.Movies.Get(id)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.notFoundResponse(writer, request)
			return
		}

		app.serverErrorResponse(writer, request, err)
		return
	}

	err = app.writeJSON(writer, http.StatusOK, envelope{"movie": movie}, request.Header)

	if err != nil {
		app.serverErrorResponse(writer, request, err)
	}
}

func (app *application) updateMovieHandler(writer http.ResponseWriter, request *http.Request) {

	id, err := app.getIdFromRequestContext(request)
	if err != nil || id < 1 {
		app.notFoundResponse(writer, request)
		return
	}

	var updateMovieRequest struct {
		Title   string       `json:"title"`
		Year    int32        `json:"year"`
		Runtime data.Runtime `json:"runtime"`
		Genres  []string     `json:"genres"`
	}
	err = app.readJSON(writer, request, &updateMovieRequest)
	if err != nil {
		app.badRequestResponse(writer, request, err)
		return
	}

	_, err = app.models.Movies.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(writer, request)
			return

		default:
			app.serverErrorResponse(writer, request, err)
			return
		}
	}

	movie := &data.Movie{
		ID:      id,
		Title:   updateMovieRequest.Title,
		Year:    updateMovieRequest.Year,
		Runtime: updateMovieRequest.Runtime,
		Genres:  updateMovieRequest.Genres,
	}

	v := validator.New()

	if data.ValidateMovie(v, movie); !v.Valid() {
		app.failedValidationResponse(writer, request, v.Errors)
		return
	}

	err = app.models.Movies.Update(movie)
	if err != nil {
		app.serverErrorResponse(writer, request, err)
	}

	err = app.writeJSON(writer, http.StatusOK, envelope{"movie": movie}, nil)
	if err != nil {
		app.serverErrorResponse(writer, request, err)
	}
}

func (app *application) deleteMovieHandler(writer http.ResponseWriter, request *http.Request) {
	id, err := app.getIdFromRequestContext(request)
	if err != nil || id < 1 {
		app.notFoundResponse(writer, request)
		return
	}

	err = app.models.Movies.Delete(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(writer, request)
		default:
			app.serverErrorResponse(writer, request, err)
		}
		return
	}

	err = app.writeJSON(writer, http.StatusOK, envelope{"message": "movie deleted successfully."}, nil)

	if err != nil {
		app.serverErrorResponse(writer, request, err)
	}
}
