package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
)

func (app *application) getIdFromRequestContext(request *http.Request) (int64, error) {
	params := httprouter.ParamsFromContext(request.Context())
	id, err := strconv.ParseInt(params.ByName("id"), 10, 64) // int64
	if err != nil || id < 1 {
		return 0, errors.New("invalid id")
	}

	return id, nil
}

type envelope map[string]interface{}

func (app *application) writeJSON(writer http.ResponseWriter, code int, data envelope, headers http.Header) error {
	resp, err := json.Marshal(data)
	if err != nil {
		return err
	}

	resp = append(resp, '\n')

	for key, value := range headers {
		writer.Header()[key] = value
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(code)
	writer.Write(resp)

	return nil

}
