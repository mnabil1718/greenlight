package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/mnabil1718/greenlight/internal/data"
	"github.com/mnabil1718/greenlight/internal/validator"
)

// don't need json tags because we're not
// parsing JSON and put it in here, this is
// just strct to hold query string params
type ListMovieRequest struct {
	Title        string
	Genres       []string
	data.Filters // can be accessed like: list.Sort OR list.Filters.Sort
}

type CreateMovieRequest struct {
	Title   string       `json:"title"`
	Year    int32        `json:"year"`
	Runtime data.Runtime `json:"runtime"`
	Genres  []string     `json:"genres"`
}

type UpdateMovieRequest struct {
	Title   *string       `json:"title"` // using pointer field to detect if request JSON doesn't include field
	Year    *int32        `json:"year"`
	Runtime *data.Runtime `json:"runtime"`
	Genres  []string      `json:"genres"`
}

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

func (app *application) listMovieHandler(writer http.ResponseWriter, request *http.Request) {
	var listMovieRequest ListMovieRequest
	validator := validator.New()
	queryString := request.URL.Query()

	listMovieRequest.Title = app.readString(queryString, "title", "")
	listMovieRequest.Genres = app.readCSV(queryString, "genres", []string{})
	listMovieRequest.PageSize = app.readInt(queryString, "page_size", 20, validator)
	listMovieRequest.Page = app.readInt(queryString, "page", 1, validator)
	listMovieRequest.Sort = app.readString(queryString, "sort", "id")
	listMovieRequest.SortSafelist = []string{"id", "title", "year", "runtime", "-id", "-title", "-year", "-runtime"}

	if data.ValidateFilters(validator, &listMovieRequest.Filters); !validator.Valid() {
		app.failedValidationResponse(writer, request, validator.Errors)
		return
	}

	movies, metadata, err := app.models.Movies.GetAll(listMovieRequest.Title, listMovieRequest.Genres, listMovieRequest.Filters)
	if err != nil {
		app.serverErrorResponse(writer, request, err)
		return
	}

	err = app.writeJSON(writer, http.StatusOK, envelope{"metadata": metadata, "movies": movies}, nil)
	if err != nil {
		app.serverErrorResponse(writer, request, err)
	}
}

func (app *application) createMovieHandler(writer http.ResponseWriter, request *http.Request) {
	// user cannot post data straight to Movie model
	// it would be unsafe. Instead use this decoy
	var createMovieRequest CreateMovieRequest

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

	var updateMovieRequest UpdateMovieRequest
	err = app.readJSON(writer, request, &updateMovieRequest)
	if err != nil {
		app.badRequestResponse(writer, request, err)
		return
	}

	movie, err := app.models.Movies.Get(id)
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

	if request.Header.Get("X-Expected-Version") != "" {
		if strconv.FormatInt(int64(movie.Version), 32) != request.Header.Get("X-Expected-Version") {
			app.editConflictResponse(writer, request)
			return
		}
	}

	// if request field is nil, the value would be
	// the previous field value from DB
	if updateMovieRequest.Title != nil {
		movie.Title = *updateMovieRequest.Title
	}
	if updateMovieRequest.Year != nil {
		movie.Year = *updateMovieRequest.Year
	}
	if updateMovieRequest.Runtime != nil {
		movie.Runtime = *updateMovieRequest.Runtime
	}
	if updateMovieRequest.Genres != nil {
		movie.Genres = updateMovieRequest.Genres // Note that we don't need to dereference a slice.
	}

	v := validator.New()

	if data.ValidateMovie(v, movie); !v.Valid() {
		app.failedValidationResponse(writer, request, v.Errors)
		return
	}

	err = app.models.Movies.Update(movie)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(writer, request)
		default:
			app.serverErrorResponse(writer, request, err)
		}
		return
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
