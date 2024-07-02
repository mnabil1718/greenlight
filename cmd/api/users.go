package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/mnabil1718/greenlight/internal/data"
	"github.com/mnabil1718/greenlight/internal/validator"
)

type CreateUserRequest struct {
	Name     string `json:"name,omitempty"`
	Email    string `json:"email,omitempty"`
	Password string `json:"password,omitempty"`
}

type ActivateUserRequest struct {
	TokenPlainText string `json:"token"`
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

	err = app.models.Permissions.AddForUser(user.ID, "movies:read")
	if err != nil {
		app.serverErrorResponse(writer, request, err)
		return
	}

	token, err := app.models.Tokens.New(user.ID, 3*24*time.Hour, data.ScopeActivation)
	if err != nil {
		app.serverErrorResponse(writer, request, err)
		return
	}

	app.background(func() {
		err = app.mailer.Send(user.Email, "user_welcome.tmpl", map[string]interface{}{
			"activationToken": token.Plaintext,
			"userID":          user.ID,
		})
		if err != nil {
			app.logger.PrintError(err, nil)
		}
	})

	err = app.writeJSON(writer, http.StatusAccepted, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(writer, request, err)
	}
}

func (app *application) activateUserHandler(writer http.ResponseWriter, request *http.Request) {
	var activateUserRequest ActivateUserRequest
	err := app.readJSON(writer, request, &activateUserRequest)
	if err != nil {
		app.badRequestResponse(writer, request, err)
		return
	}

	v := validator.New()

	if data.ValidateTokenPlaintext(v, activateUserRequest.TokenPlainText); !v.Valid() {
		app.failedValidationResponse(writer, request, v.Errors)
		return
	}

	user, err := app.models.Users.GetForToken(data.ScopeActivation, activateUserRequest.TokenPlainText)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			v.AddError("token", "invalid or expired activation token")
			app.failedValidationResponse(writer, request, v.Errors)
		default:
			app.serverErrorResponse(writer, request, err)
		}
		return
	}

	user.Activated = true

	err = app.models.Users.Update(user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(writer, request)
		default:
			app.serverErrorResponse(writer, request, err)
		}
		return
	}

	err = app.models.Tokens.DeleteForAllUser(data.ScopeActivation, user.ID)
	if err != nil {
		app.serverErrorResponse(writer, request, err)
		return
	}

	err = app.writeJSON(writer, http.StatusOK, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(writer, request, err)
	}
}
