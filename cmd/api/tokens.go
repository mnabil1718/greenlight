package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/mnabil1718/greenlight/internal/data"
	"github.com/mnabil1718/greenlight/internal/validator"
)

type CreateAuthTokenRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (app *application) createAuthTokenHandler(w http.ResponseWriter, r *http.Request) {

	var createAuthTokenRequest CreateAuthTokenRequest

	err := app.readJSON(w, r, &createAuthTokenRequest)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()

	data.ValidateEmail(v, createAuthTokenRequest.Email)
	data.ValidatePasswordPlaintext(v, createAuthTokenRequest.Password)

	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	user, err := app.models.Users.GetByEmail(createAuthTokenRequest.Email)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.invalidCredentialsResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	ok, err := user.Password.Matches(createAuthTokenRequest.Password)
	if !ok {
		if err != nil {
			app.serverErrorResponse(w, r, err)
		} else {
			app.invalidCredentialsResponse(w, r)
		}
		return
	}

	token, err := app.models.Tokens.New(user.ID, 24*time.Hour, data.ScopeAuthentication)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusCreated, envelope{"authentication_token": token}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
