package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

const apiVersionURLPrefix string = "/v1"

func (app *application) routes() *httprouter.Router {
	router := httprouter.New()
	router.NotFound = http.HandlerFunc(app.notFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

	router.HandlerFunc(http.MethodGet, apiVersionURLPrefix+"/healthcheck", app.healthcheckHandler)
	router.HandlerFunc(http.MethodPost, apiVersionURLPrefix+"/movies", app.createMovieHandler)
	router.HandlerFunc(http.MethodGet, apiVersionURLPrefix+"/movies/:id", app.showMovieHandler)

	return router
}
