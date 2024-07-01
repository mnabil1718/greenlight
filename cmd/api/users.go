package main

import (
	"errors"
	"net/http"

	"github.com/mnabil1718/greenlight/internal/data"
	"github.com/mnabil1718/greenlight/internal/validator"
)

type CreateUserRequest struct {
	Name     string `json:"name,omitempty"`
	Email    string `json:"email,omitempty"`
	Password string `json:"password,omitempty"`
}

func (app *application) registerUserHandler(writer http.ResponseWriter, request *http.Request) {
	var createUserRequest CreateUserRequest
	err := app.readJSON(writer, request, &createUserRequest)
	if err != nil {
		app.badRequestResponse(writer, request, err)
		return
	}

	user := &data.User{
		Name:      createUserRequest.Name,
		Email:     createUserRequest.Email,
		Activated: false,
	}
	err = user.Password.Set(createUserRequest.Password)
	if err != nil {
		app.badRequestResponse(writer, request, err)
		return
	}

	v := validator.New()
	if data.ValidateUser(v, user); !v.Valid() {
		app.failedValidationResponse(writer, request, v.Errors)
		return
	}

	err = app.models.Users.Insert(user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateEmail):
			v.AddError("email", "email already exists")
			app.failedValidationResponse(writer, request, v.Errors)

		default:
			app.serverErrorResponse(writer, request, err)
		}
		return
	}

	app.background(func() {
		err = app.mailer.Send(user.Email, "user_welcome.tmpl", user)
		if err != nil {
			app.logger.PrintError(err, nil)
		}
	})

	err = app.writeJSON(writer, http.StatusAccepted, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(writer, request, err)
	}
}
